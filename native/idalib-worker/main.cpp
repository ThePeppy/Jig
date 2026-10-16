#include "json.hpp"
#include "tools.hpp"

#include <arpa/inet.h>
#include <netdb.h>
#include <poll.h>
#include <sys/socket.h>
#include <unistd.h>

#include <cerrno>
#include <cstring>
#include <iostream>
#include <string>
#include <vector>

namespace {

struct Options {
  std::string host = "127.0.0.1";
  int port = 9399;
  std::string path;
  std::string instance_id;
  std::string ida_dir;
};

void usage() {
  std::cerr << "usage: jig-idalib --path FILE --instance-id ID [--center-host HOST] [--center-port PORT] [--ida-dir DIR]\n";
}

bool take(int &i, int argc, char **argv, std::string &out) {
  if (i + 1 >= argc) return false;
  out = argv[++i];
  return true;
}

bool parse_args(int argc, char **argv, Options &opt) {
  for (int i = 1; i < argc; ++i) {
    std::string arg = argv[i];
    if (arg == "--center-host") {
      if (!take(i, argc, argv, opt.host)) return false;
    } else if (arg == "--center-port") {
      std::string raw;
      if (!take(i, argc, argv, raw)) return false;
      opt.port = std::stoi(raw);
    } else if (arg == "--path") {
      if (!take(i, argc, argv, opt.path)) return false;
    } else if (arg == "--instance-id") {
      if (!take(i, argc, argv, opt.instance_id)) return false;
    } else if (arg == "--ida-dir") {
      if (!take(i, argc, argv, opt.ida_dir)) return false;
    } else if (arg == "--auto-analysis" || arg == "--alias" || arg == "--worker-id" || arg == "--ida-arg") {
      if (i + 1 >= argc) return false;
      ++i;
    } else {
      return false;
    }
  }
  return !opt.path.empty() && !opt.instance_id.empty();
}

int connect_host(const std::string &host, int port) {
  addrinfo hints{};
  hints.ai_family = AF_UNSPEC;
  hints.ai_socktype = SOCK_STREAM;
  addrinfo *res = nullptr;
  std::string service = std::to_string(port);
  if (getaddrinfo(host.c_str(), service.c_str(), &hints, &res) != 0) return -1;
  int fd = -1;
  for (addrinfo *it = res; it != nullptr; it = it->ai_next) {
    fd = socket(it->ai_family, it->ai_socktype, it->ai_protocol);
    if (fd < 0) continue;
    if (connect(fd, it->ai_addr, it->ai_addrlen) == 0) break;
    close(fd);
    fd = -1;
  }
  freeaddrinfo(res);
  return fd;
}

bool write_all(int fd, const void *data, size_t n) {
  const char *p = static_cast<const char *>(data);
  while (n > 0) {
    ssize_t w = send(fd, p, n, 0);
    if (w < 0) {
      if (errno == EINTR) continue;
      return false;
    }
    p += w;
    n -= static_cast<size_t>(w);
  }
  return true;
}

bool write_message(int fd, const jig::Json &msg) {
  std::string body = jig::dump(msg);
  uint32_t len = htonl(static_cast<uint32_t>(body.size()));
  return write_all(fd, &len, 4) && write_all(fd, body.data(), body.size());
}

bool read_full(int fd, void *data, size_t n) {
  char *p = static_cast<char *>(data);
  while (n > 0) {
    ssize_t r = recv(fd, p, n, 0);
    if (r == 0) return false;
    if (r < 0) {
      if (errno == EINTR) continue;
      return false;
    }
    p += r;
    n -= static_cast<size_t>(r);
  }
  return true;
}

bool read_message(int fd, jig::Json &out, std::string &error) {
  uint32_t len_be = 0;
  if (!read_full(fd, &len_be, 4)) return false;
  uint32_t len = ntohl(len_be);
  if (len == 0 || len > 64 * 1024 * 1024) {
    error = "invalid frame";
    return false;
  }
  std::string body(len, '\0');
  if (!read_full(fd, &body[0], len)) return false;
  try {
    out = jig::parse(body);
  } catch (const std::exception &exc) {
    error = exc.what();
    return false;
  }
  return true;
}

void send_result(int fd, const std::string &instance_id, const std::string &id, bool ok, const jig::Json &result, const std::string &error) {
  jig::Json msg = jig::Json::obj();
  msg["type"] = jig::Json::str("tool_result");
  msg["id"] = jig::Json::str(id);
  msg["instanceId"] = jig::Json::str(instance_id);
  jig::Json payload = jig::Json::obj();
  payload["ok"] = jig::Json::boolean(ok);
  payload["result"] = ok ? result : jig::Json::nul();
  payload["error"] = error.empty() ? jig::Json::nul() : jig::Json::str(error);
  msg["payload"] = std::move(payload);
  write_message(fd, msg);
}

}  // namespace

int main(int argc, char **argv) {
  Options opt;
  if (!parse_args(argc, argv, opt)) {
    usage();
    return 2;
  }
  if (!opt.ida_dir.empty()) {
    setenv("IDADIR", opt.ida_dir.c_str(), 1);
  }

  std::string open_error;
  if (!jig::open_database(opt.path, open_error)) {
    std::cerr << "[jig-idalib] open_database failed: " << open_error << "\n";
    return 2;
  }

  int fd = connect_host(opt.host, opt.port);
  if (fd < 0) {
    std::cerr << "[jig-idalib] cannot connect to " << opt.host << ":" << opt.port << "\n";
    jig::close_database();
    return 3;
  }

  jig::Json hello = jig::Json::obj();
  hello["type"] = jig::Json::str("hello");
  hello["instanceId"] = jig::Json::str(opt.instance_id);
  hello["payload"] = jig::execute_tool("target.get_metadata", jig::Json::obj(), opt.instance_id, opt.path);
  if (!write_message(fd, hello)) {
    close(fd);
    jig::close_database();
    return 3;
  }
  std::cerr << "[jig-idalib] connected " << opt.host << ":" << opt.port << " as " << opt.instance_id << "\n";

  bool stop = false;
  while (!stop) {
    pollfd pfd{};
    pfd.fd = fd;
    pfd.events = POLLIN;
    int pr = poll(&pfd, 1, 50);
    if (pr < 0) {
      if (errno == EINTR) continue;
      break;
    }
    if (pr == 0) {
      jig::pump_auto_steps(8);
      continue;
    }
    jig::Json msg;
    std::string error;
    if (!read_message(fd, msg, error)) break;
    std::string type = msg.str_or("type");
    if (type == "shutdown") break;
    if (type != "tool_call") continue;
    std::string id = msg.str_or("id");
    const jig::Json *payload = msg.find("payload");
    jig::Json args = jig::Json::obj();
    std::string tool;
    if (payload != nullptr) {
      tool = payload->str_or("tool");
      if (const jig::Json *raw = payload->find("arguments"); raw != nullptr) args = *raw;
    }
    try {
      jig::Json result = jig::execute_tool(tool, args, opt.instance_id, opt.path);
      if (const jig::Json *exit_flag = result.find("exit"); exit_flag != nullptr && exit_flag->type == jig::Json::BOOL && exit_flag->b) {
        stop = true;
      }
      send_result(fd, opt.instance_id, id, true, result, "");
    } catch (const std::exception &exc) {
      send_result(fd, opt.instance_id, id, false, jig::Json::nul(), exc.what());
    }
  }

  close(fd);
  jig::close_database();
  return 0;
}
