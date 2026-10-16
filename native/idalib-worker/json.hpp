#pragma once

#include <cstdio>

#include <cctype>
#include <cmath>
#include <map>
#include <sstream>
#include <stdexcept>
#include <string>
#include <vector>

namespace jig {

struct Json {
  enum Type { NUL, BOOL, NUM, STR, ARR, OBJ } type = NUL;
  bool b = false;
  double n = 0;
  bool integral = false;
  std::string s;
  std::vector<Json> a;
  std::map<std::string, Json> o;

  static Json nul() { return Json(); }
  static Json boolean(bool v) {
    Json j;
    j.type = BOOL;
    j.b = v;
    return j;
  }
  static Json number(double v, bool whole = false) {
    Json j;
    j.type = NUM;
    j.n = v;
    j.integral = whole;
    return j;
  }
  static Json integer(long long v) { return number(static_cast<double>(v), true); }
  static Json str(std::string v) {
    Json j;
    j.type = STR;
    j.s = std::move(v);
    return j;
  }
  static Json arr() {
    Json j;
    j.type = ARR;
    return j;
  }
  static Json obj() {
    Json j;
    j.type = OBJ;
    return j;
  }

  Json &operator[](const std::string &key) { type = OBJ; return o[key]; }
  const Json *find(const std::string &key) const {
    if (type != OBJ) return nullptr;
    auto it = o.find(key);
    return it == o.end() ? nullptr : &it->second;
  }
  void push(Json v) { type = ARR; a.push_back(std::move(v)); }

  std::string str_or(const std::string &key, const std::string &fallback = "") const {
    const Json *v = find(key);
    if (v == nullptr || v->type != STR) return fallback;
    return v->s;
  }
  bool bool_or(const std::string &key, bool fallback) const {
    const Json *v = find(key);
    if (v == nullptr) return fallback;
    if (v->type == BOOL) return v->b;
    if (v->type == STR) {
      std::string t = v->s;
      for (char &c : t) c = static_cast<char>(std::tolower(static_cast<unsigned char>(c)));
      return t == "1" || t == "true" || t == "yes" || t == "on";
    }
    if (v->type == NUM) return v->n != 0;
    return fallback;
  }
};

inline void dump_string(std::ostream &out, const std::string &s) {
  out << '"';
  for (unsigned char c : s) {
    switch (c) {
    case '"': out << "\\\""; break;
    case '\\': out << "\\\\"; break;
    case '\b': out << "\\b"; break;
    case '\f': out << "\\f"; break;
    case '\n': out << "\\n"; break;
    case '\r': out << "\\r"; break;
    case '\t': out << "\\t"; break;
    default:
      if (c < 0x20) {
        char buf[8];
        std::snprintf(buf, sizeof(buf), "\\u%04x", c);
        out << buf;
      } else {
        out << static_cast<char>(c);
      }
    }
  }
  out << '"';
}

inline void dump(std::ostream &out, const Json &j) {
  switch (j.type) {
  case Json::NUL: out << "null"; break;
  case Json::BOOL: out << (j.b ? "true" : "false"); break;
  case Json::NUM:
    if (j.integral && std::isfinite(j.n) && j.n == std::trunc(j.n)) {
      out << static_cast<long long>(j.n);
    } else {
      out << j.n;
    }
    break;
  case Json::STR: dump_string(out, j.s); break;
  case Json::ARR:
    out << '[';
    for (size_t i = 0; i < j.a.size(); ++i) {
      if (i) out << ',';
      dump(out, j.a[i]);
    }
    out << ']';
    break;
  case Json::OBJ:
    out << '{';
    bool first = true;
    for (const auto &kv : j.o) {
      if (!first) out << ',';
      first = false;
      dump_string(out, kv.first);
      out << ':';
      dump(out, kv.second);
    }
    out << '}';
    break;
  }
}

inline std::string dump(const Json &j) {
  std::ostringstream out;
  dump(out, j);
  return out.str();
}

class Parser {
 public:
  explicit Parser(const std::string &in) : in_(in) {}

  Json parse() {
    skip();
    Json v = value();
    skip();
    if (i_ != in_.size()) throw std::runtime_error("trailing json");
    return v;
  }

 private:
  const std::string &in_;
  size_t i_ = 0;

  void skip() {
    while (i_ < in_.size() && std::isspace(static_cast<unsigned char>(in_[i_]))) ++i_;
  }
  char peek() const { return i_ < in_.size() ? in_[i_] : 0; }
  char get() { return i_ < in_.size() ? in_[i_++] : 0; }

  Json value() {
    skip();
    char c = peek();
    if (c == '{') return object();
    if (c == '[') return array();
    if (c == '"') return Json::str(string());
    if (c == 't' || c == 'f') return Json::boolean(boolean());
    if (c == 'n') {
      literal("null");
      return Json::nul();
    }
    if (c == '-' || std::isdigit(static_cast<unsigned char>(c))) return number();
    throw std::runtime_error("invalid json value");
  }

  Json object() {
    get();
    Json j = Json::obj();
    skip();
    if (peek() == '}') {
      get();
      return j;
    }
    while (true) {
      skip();
      if (peek() != '"') throw std::runtime_error("object key");
      std::string key = string();
      skip();
      if (get() != ':') throw std::runtime_error("object colon");
      j.o[key] = value();
      skip();
      char c = get();
      if (c == '}') break;
      if (c != ',') throw std::runtime_error("object comma");
    }
    return j;
  }

  Json array() {
    get();
    Json j = Json::arr();
    skip();
    if (peek() == ']') {
      get();
      return j;
    }
    while (true) {
      j.a.push_back(value());
      skip();
      char c = get();
      if (c == ']') break;
      if (c != ',') throw std::runtime_error("array comma");
    }
    return j;
  }

  std::string string() {
    if (get() != '"') throw std::runtime_error("string");
    std::string out;
    while (i_ < in_.size()) {
      char c = get();
      if (c == '"') return out;
      if (c != '\\') {
        out.push_back(c);
        continue;
      }
      char e = get();
      switch (e) {
      case '"':
      case '\\':
      case '/': out.push_back(e); break;
      case 'b': out.push_back('\b'); break;
      case 'f': out.push_back('\f'); break;
      case 'n': out.push_back('\n'); break;
      case 'r': out.push_back('\r'); break;
      case 't': out.push_back('\t'); break;
      case 'u': {
        int cp = 0;
        for (int k = 0; k < 4; ++k) {
          char h = get();
          cp <<= 4;
          if (h >= '0' && h <= '9') cp += h - '0';
          else if (h >= 'a' && h <= 'f') cp += h - 'a' + 10;
          else if (h >= 'A' && h <= 'F') cp += h - 'A' + 10;
          else throw std::runtime_error("bad unicode escape");
        }
        if (cp < 0x80) {
          out.push_back(static_cast<char>(cp));
        } else if (cp < 0x800) {
          out.push_back(static_cast<char>(0xC0 | (cp >> 6)));
          out.push_back(static_cast<char>(0x80 | (cp & 0x3F)));
        } else {
          out.push_back(static_cast<char>(0xE0 | (cp >> 12)));
          out.push_back(static_cast<char>(0x80 | ((cp >> 6) & 0x3F)));
          out.push_back(static_cast<char>(0x80 | (cp & 0x3F)));
        }
        break;
      }
      default: throw std::runtime_error("bad escape");
      }
    }
    throw std::runtime_error("unterminated string");
  }

  bool boolean() {
    if (peek() == 't') {
      literal("true");
      return true;
    }
    literal("false");
    return false;
  }

  void literal(const char *lit) {
    for (const char *p = lit; *p; ++p) {
      if (get() != *p) throw std::runtime_error("bad literal");
    }
  }

  Json number() {
    size_t start = i_;
    if (peek() == '-') get();
    if (!std::isdigit(static_cast<unsigned char>(peek()))) throw std::runtime_error("bad number");
    while (std::isdigit(static_cast<unsigned char>(peek()))) get();
    bool whole = true;
    if (peek() == '.') {
      whole = false;
      get();
      while (std::isdigit(static_cast<unsigned char>(peek()))) get();
    }
    if (peek() == 'e' || peek() == 'E') {
      whole = false;
      get();
      if (peek() == '+' || peek() == '-') get();
      while (std::isdigit(static_cast<unsigned char>(peek()))) get();
    }
    double v = std::strtod(in_.c_str() + start, nullptr);
    return Json::number(v, whole);
  }
};

inline Json parse(const std::string &in) { return Parser(in).parse(); }

}  // namespace jig
