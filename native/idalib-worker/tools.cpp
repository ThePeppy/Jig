#include "tools.hpp"

#include <idalib.hpp>
#include <auto.hpp>
#include <err.h>
#include <bytes.hpp>
#include <funcs.hpp>
#include <hexrays.hpp>
#include <idp.hpp>
#include <kernwin.hpp>
#include <lines.hpp>
#include <diskio.hpp>
#include <nalt.hpp>
#include <name.hpp>
#include <segment.hpp>
#include <strlist.hpp>
#include <ua.hpp>
#include <xref.hpp>
#include <iostream>


#include <algorithm>
#include <cctype>
#include <chrono>
#include <filesystem>
#include <map>
#include <regex>
#include <sstream>
#include <stdexcept>

namespace jig {
namespace {

bool g_open = false;
bool g_dirty = false;
bool g_pump = false;
bool g_hexrays = false;

int clamp_int(const Json *value, int fallback, int lo, int hi) {
  int parsed = fallback;
  if (value != nullptr && value->type == Json::NUM) parsed = static_cast<int>(value->n);
  if (value != nullptr && value->type == Json::STR) {
    try {
      parsed = std::stoi(value->s);
    } catch (...) {
      parsed = fallback;
    }
  }
  return std::max(lo, std::min(hi, parsed));
}

int clamp_arg(const Json &args, const char *a, const char *b, int fallback, int lo, int hi) {
  const Json *value = args.find(a);
  if (value == nullptr && b != nullptr) value = args.find(b);
  return clamp_int(value, fallback, lo, hi);
}

std::string hex_ea(ea_t ea) {
  std::ostringstream out;
  out << "0x" << std::hex << std::nouppercase << static_cast<unsigned long long>(ea);
  return out.str();
}

std::string lower_copy(std::string s) {
  for (char &c : s) c = static_cast<char>(std::tolower(static_cast<unsigned char>(c)));
  return s;
}

std::string func_name(ea_t ea) {
  qstring name;
  if (get_func_name(&name, ea) > 0 && !name.empty()) return name.c_str();
  std::ostringstream out;
  out << "sub_" << std::hex << std::nouppercase << static_cast<unsigned long long>(ea);
  return out.str();
}

ea_t parse_address(const Json *query) {
  if (query == nullptr) throw std::runtime_error("address query must be a non-empty string or integer");
  if (query->type == Json::NUM) return static_cast<ea_t>(query->n);
  if (query->type != Json::STR || query->s.find_first_not_of(" \t") == std::string::npos) {
    throw std::runtime_error("address query must be a non-empty string or integer");
  }
  std::string text = query->s;
  text.erase(0, text.find_first_not_of(" \t"));
  text.erase(text.find_last_not_of(" \t") + 1);
  char *end = nullptr;
  unsigned long long value = std::strtoull(text.c_str(), &end, 0);
  if (end != text.c_str() && *end == 0) return static_cast<ea_t>(value);
  if (text.rfind("sub_", 0) == 0) {
    value = std::strtoull(text.c_str() + 4, &end, 16);
    if (end != text.c_str() + 4 && *end == 0) return static_cast<ea_t>(value);
  }
  ea_t ea = get_name_ea(BADADDR, text.c_str());
  if (ea == BADADDR) throw std::runtime_error("unable to resolve address: " + text);
  return ea;
}

const Json *addr_arg(const Json &args) {
  const Json *v = args.find("address");
  if (v == nullptr) v = args.find("query");
  return v;
}

ea_t parse_function_start(const Json &args) {
  ea_t ea = parse_address(addr_arg(args));
  func_t *func = get_func(ea);
  if (func != nullptr) {
    if (func->start_ea == BADADDR) throw std::runtime_error("invalid function address: " + hex_ea(ea));
    return func->start_ea;
  }
  if (!args.bool_or("create", false)) {
    throw std::runtime_error("not in a function: " + hex_ea(ea));
  }
  if (!is_code(get_flags(ea))) {
    if (create_insn(ea) <= 0) throw std::runtime_error("无法在 " + hex_ea(ea) + " 定义为代码");
  }
  if (!add_func(ea, BADADDR)) throw std::runtime_error("无法在 " + hex_ea(ea) + " 创建函数");
  func = get_func(ea);
  if (func == nullptr || func->start_ea == BADADDR) throw std::runtime_error("创建函数后仍无法解析: " + hex_ea(ea));
  return func->start_ea;
}

bool load_decompiler(const char *name) {
  if (name == nullptr || name[0] == 0 || load_plugin(name) == nullptr) return false;
  return init_hexrays_plugin();
}

bool ensure_hexrays() {
  if (g_hexrays) return true;
  const char *primary = nullptr;
  const char *fallback = nullptr;
  switch (PH.id) {
  case PLFM_386:
    primary = inf_is_64bit() ? "hexx64" : "hexrays";
    fallback = "hexx64";
    break;
  case PLFM_ARM:
    primary = inf_is_64bit() ? "hexarm64" : "hexarm";
    fallback = "hexarm";
    break;
  case PLFM_PPC:
    primary = inf_is_64bit() ? "hexppc64" : "hexppc";
    fallback = "hexppc";
    break;
  case PLFM_MIPS:
    primary = inf_is_64bit() ? "hexmips64" : "hexmips";
    fallback = "hexmips";
    break;
  case PLFM_RISCV:
    primary = "hexrv";
    break;
  case PLFM_ARC:
    primary = "hexarc";
    break;
  case PLFM_NEC_V850X:
    primary = "hexv850";
    break;
  default:
    break;
  }
  g_hexrays = load_decompiler(primary) || load_decompiler(fallback);
  return g_hexrays;
}

std::string auto_state_name(atype_t state) {
  struct Item { atype_t value; const char *name; };
  const Item items[] = {
      {AU_NONE, "AU_NONE"}, {AU_UNK, "AU_UNK"}, {AU_CODE, "AU_CODE"}, {AU_WEAK, "AU_WEAK"},
      {AU_PROC, "AU_PROC"}, {AU_TAIL, "AU_TAIL"}, {AU_FCHUNK, "AU_FCHUNK"}, {AU_USED, "AU_USED"},
      {AU_USD2, "AU_USD2"}, {AU_TYPE, "AU_TYPE"}, {AU_LIBF, "AU_LIBF"}, {AU_LBF2, "AU_LBF2"},
      {AU_LBF3, "AU_LBF3"}, {AU_CHLB, "AU_CHLB"}, {AU_FINAL, "AU_FINAL"},
  };
  for (const Item &item : items) {
    if (item.value == state) return item.name;
  }
  return "AU_" + std::to_string(state);
}

std::string alias_from_name(const std::string &name) {
  std::filesystem::path path(name);
  std::string base = path.stem().string();
  if (base.empty()) base = name;
  std::string out;
  for (char c : base) {
    if (std::isalnum(static_cast<unsigned char>(c))) out.push_back(static_cast<char>(std::tolower(static_cast<unsigned char>(c))));
    else out.push_back('_');
  }
  while (!out.empty() && out.front() == '_') out.erase(out.begin());
  while (!out.empty() && out.back() == '_') out.pop_back();
  return out.empty() ? "ida" : out;
}

std::string file_ext(const std::string &path) {
  std::string ext = lower_copy(std::filesystem::path(path).extension().string());
  if (!ext.empty() && ext.front() == '.') ext.erase(ext.begin());
  return ext;
}

// Ask the installed loaders. An empty list means none accepted the file.
// Database extensions are not loader inputs; open_database handles those.
std::string loaders_or_error(const std::string &path) {
  std::string ext = file_ext(path);
  if (!ext.empty() && is_database_ext(ext.c_str())) return "";
  linput_t *li = open_linput(path.c_str(), false);
  if (li == nullptr) {
    const char *why = qstrerror(get_qerrno());
    if (why != nullptr && why[0] != 0) return std::string("打不开这个文件: ") + why;
    return "打不开这个文件";
  }
  load_info_t *list = build_loaders_list(li, path.c_str());
  close_linput(li);
  if (list == nullptr) return "没有装载器接受这个文件";
  free_loaders_list(list);
  return "";
}
struct msg_sink : event_listener_t {
  std::string text;
  ssize_t idaapi on_event(ssize_t code, va_list va) override {
    if (code != ui_msg) return 0;
    const char *fmt = va_arg(va, const char *);
    if (fmt == nullptr || fmt[0] == 0) return 0;
    va_list incoming = va_arg(va, va_list);
    va_list args;
    va_copy(args, incoming);
    qstring line;
    line.vsprnt(fmt, args);
    va_end(args);
    if (line.empty()) return 0;
    if (!text.empty() && text.back() != '\n') text.push_back('\n');
    text += line.c_str();
    return 0;
  }
};

std::string trim_copy(std::string text) {
  while (!text.empty() && (text.back() == '\n' || text.back() == '\r' || text.back() == ' ')) text.pop_back();
  return text;
}

std::string output_lines() {
  qstrvec_t lines;
  msg_get_lines(&lines, -1);
  std::string text;
  for (ssize_t i = static_cast<ssize_t>(lines.size()) - 1; i >= 0; --i) {
    if (!text.empty() && text.back() != '\n') text.push_back('\n');
    text += lines[i].c_str();
  }
  return trim_copy(text);
}

Json function_row(func_t *func) {
  Json row = Json::obj();
  row["addr"] = Json::str(hex_ea(func->start_ea));
  row["name"] = Json::str(func_name(func->start_ea));
  row["size"] = Json::integer(static_cast<long long>(func->end_ea - func->start_ea));
  row["end"] = Json::str(hex_ea(func->end_ea));
  return row;
}

Json list_functions(const Json &args) {
  int offset = clamp_arg(args, "offset", nullptr, 0, 0, 1000000);
  int count = clamp_arg(args, "count", nullptr, 100, 1, 1000);
  std::string filter = lower_copy(args.str_or("filter"));
  std::vector<Json> rows;
  size_t qty = get_func_qty();
  for (size_t i = 0; i < qty; ++i) {
    func_t *func = getn_func(i);
    if (func == nullptr) continue;
    std::string name = func_name(func->start_ea);
    if (!filter.empty() && lower_copy(name).find(filter) == std::string::npos) continue;
    rows.push_back(function_row(func));
  }
  Json page = Json::arr();
  for (int i = offset; i < static_cast<int>(rows.size()) && i < offset + count; ++i) page.push(rows[static_cast<size_t>(i)]);
  Json out = Json::obj();
  out["data"] = std::move(page);
  out["nextOffset"] = (offset + count < static_cast<int>(rows.size())) ? Json::integer(offset + count) : Json::nul();
  out["total"] = Json::integer(static_cast<long long>(rows.size()));
  return out;
}

Json get_function(const Json &args) {
  ea_t start = parse_function_start(args);
  func_t *func = get_func(start);
  if (func == nullptr) throw std::runtime_error("no function at " + hex_ea(start));
  return function_row(func);
}

Json lvar_row(const lvar_t &lv) {
  Json row = Json::obj();
  row["name"] = Json::str(lv.name.c_str());
  qstring ts;
  if (!lv.tif.empty()) lv.tif.print(&ts);
  row["type"] = ts.empty() ? Json::nul() : Json::str(ts.c_str());
  row["isArg"] = Json::boolean(lv.is_arg_var());
  row["isResult"] = Json::boolean(lv.is_result_var());
  row["userName"] = Json::boolean(lv.has_user_name());
  row["niceName"] = Json::boolean(lv.has_nice_name());
  row["width"] = Json::integer(lv.width);
  return row;
}

Json decompile_function(const Json &args) {
  ea_t start = parse_function_start(args);
  Json out = Json::obj();
  out["addr"] = Json::str(hex_ea(start));
  if (!ensure_hexrays()) {
    out["code"] = Json::nul();
    out["error"] = Json::str("Hex-Rays plugin is not available for this processor");
    return out;
  }
  func_t *func = get_func(start);
  if (func == nullptr) {
    out["code"] = Json::nul();
    out["error"] = Json::str("no function at " + hex_ea(start));
    return out;
  }
  hexrays_failure_t hf;
  cfuncptr_t cfunc = decompile(func, &hf, 0);
  if (cfunc == nullptr) {
    qstring why = hf.desc();
    out["code"] = Json::nul();
    out["error"] = Json::str(why.empty() ? "Hex-Rays returned no decompilation" : why.c_str());
    return out;
  }
  const strvec_t &lines = cfunc->get_pseudocode();
  std::string code;
  for (size_t i = 0; i < lines.size(); ++i) {
    qstring plain;
    tag_remove(&plain, lines[i].line);
    if (i) code.push_back('\n');
    code += plain.c_str();
  }
  out["code"] = Json::str(code);
  Json lvars = Json::arr();
  lvars_t *vars = cfunc->get_lvars();
  if (vars != nullptr) {
    for (size_t i = 0; i < vars->size(); ++i) lvars.push(lvar_row((*vars)[i]));
  }
  out["lvars"] = std::move(lvars);
  return out;
}

Json disassemble(const Json &args) {
  ea_t start = parse_function_start(args);
  int max_lines = clamp_arg(args, "maxLines", nullptr, 200, 1, 5000);
  func_t *func = get_func(start);
  if (func == nullptr) throw std::runtime_error("no function at " + hex_ea(start));
  Json lines = Json::arr();
  ea_t ea = func->start_ea;
  while (ea < func->end_ea && static_cast<int>(lines.a.size()) < max_lines) {
    qstring text;
    generate_disasm_line(&text, ea, GENDSM_REMOVE_TAGS);
    Json row = Json::obj();
    row["addr"] = Json::str(hex_ea(ea));
    row["text"] = Json::str(text.c_str());
    lines.push(std::move(row));
    ea_t next = next_head(ea, func->end_ea);
    if (next <= ea) break;
    ea = next;
  }
  Json out = Json::obj();
  out["addr"] = Json::str(hex_ea(func->start_ea));
  out["lines"] = std::move(lines);
  out["truncated"] = Json::boolean(ea < func->end_ea);
  return out;
}


Json xref_row(const char *direction, const char *kind, ea_t source, ea_t target) {
  Json row = Json::obj();
  row["direction"] = Json::str(direction);
  row["kind"] = Json::str(kind);
  row["from"] = Json::str(hex_ea(source));
  row["to"] = Json::str(hex_ea(target));
  qstring from_name = get_name(source);
  qstring to_name = get_name(target);
  row["fromName"] = from_name.empty() ? Json::nul() : Json::str(from_name.c_str());
  row["toName"] = to_name.empty() ? Json::nul() : Json::str(to_name.c_str());
  func_t *func = get_func(source);
  row["function"] = func == nullptr ? Json::nul() : Json::str(func_name(func->start_ea));
  return row;
}

Json xrefs(const Json &args) {
  ea_t ea = parse_address(addr_arg(args));
  std::string direction = lower_copy(args.str_or("direction", "both"));
  std::string kind = lower_copy(args.str_or("kind", "both"));
  int max_items = clamp_arg(args, "max", nullptr, 200, 1, 5000);
  Json rows = Json::arr();
  auto push_code_to = [&]() {
    ea_t ref = get_first_cref_to(ea);
    while (ref != BADADDR && static_cast<int>(rows.a.size()) < max_items) {
      rows.push(xref_row("to", "code", ref, ea));
      ref = get_next_cref_to(ea, ref);
    }
  };
  auto push_data_to = [&]() {
    ea_t ref = get_first_dref_to(ea);
    while (ref != BADADDR && static_cast<int>(rows.a.size()) < max_items) {
      rows.push(xref_row("to", "data", ref, ea));
      ref = get_next_dref_to(ea, ref);
    }
  };
  auto push_code_from = [&]() {
    ea_t ref = get_first_cref_from(ea);
    while (ref != BADADDR && static_cast<int>(rows.a.size()) < max_items) {
      rows.push(xref_row("from", "code", ea, ref));
      ref = get_next_cref_from(ea, ref);
    }
  };
  auto push_data_from = [&]() {
    ea_t ref = get_first_dref_from(ea);
    while (ref != BADADDR && static_cast<int>(rows.a.size()) < max_items) {
      rows.push(xref_row("from", "data", ea, ref));
      ref = get_next_dref_from(ea, ref);
    }
  };
  bool want_code = kind == "code" || kind == "both";
  bool want_data = kind == "data" || kind == "both";
  if (direction == "to" || direction == "both") {
    if (want_code) push_code_to();
    if (want_data) push_data_to();
  }
  if (direction == "from" || direction == "both") {
    if (want_code) push_code_from();
    if (want_data) push_data_from();
  }
  Json out = Json::obj();
  out["addr"] = Json::str(hex_ea(ea));
  out["direction"] = Json::str(direction);
  out["kind"] = Json::str(kind);
  out["data"] = std::move(rows);
  out["truncated"] = Json::boolean(static_cast<int>(out.find("data")->a.size()) >= max_items);
  return out;
}

cfuncptr_t decompile_ready(ea_t start) {
  if (!ensure_hexrays()) throw std::runtime_error("Hex-Rays plugin is not available for this processor");
  func_t *func = get_func(start);
  if (func == nullptr) throw std::runtime_error("no function at " + hex_ea(start));
  hexrays_failure_t hf;
  cfuncptr_t cfunc = decompile(func, &hf, 0);
  if (cfunc == nullptr) {
    qstring why = hf.desc();
    throw std::runtime_error(why.empty() ? "Hex-Rays returned no decompilation" : why.c_str());
  }
  return cfunc;
}

struct call_collector : public ctree_parentee_t {
  cfunc_t *fn;
  Json rows = Json::arr();
  int max_items = 200;
  explicit call_collector(cfunc_t *f) : ctree_parentee_t(false), fn(f) {}
  int idaapi visit_expr(cexpr_t *e) override {
    if (e == nullptr || e->op != cot_call) return 0;
    if (static_cast<int>(rows.a.size()) >= max_items) return 1;
    Json row = Json::obj();
    row["callSite"] = Json::str(hex_ea(e->ea));
    ea_t callee = BADADDR;
    std::string callee_name;
    if (e->x != nullptr) {
      if (e->x->op == cot_obj) {
        callee = e->x->obj_ea;
        callee_name = func_name(callee);
      } else if (e->x->op == cot_helper && e->x->helper != nullptr) {
        callee_name = e->x->helper;
      } else {
        qstring pretty;
        e->x->print1(&pretty, fn);
        tag_remove(&pretty);
        callee_name = pretty.c_str();
      }
    }
    row["callee"] = callee == BADADDR ? Json::nul() : Json::str(hex_ea(callee));
    row["calleeName"] = Json::str(callee_name);
    cexpr_t *p = parent_expr();
    if (p != nullptr && is_assignment(p->op) && p->y == e && p->x != nullptr && p->x->op == cot_var) {
      lvars_t *lvars = fn->get_lvars();
      int idx = p->x->v.idx;
      if (lvars != nullptr && idx >= 0 && idx < static_cast<int>(lvars->size())) {
        const lvar_t &lv = (*lvars)[static_cast<size_t>(idx)];
        row["assignedTo"] = Json::str(lv.name.c_str());
        row["assignedNice"] = Json::boolean(lv.has_user_name() || lv.has_nice_name());
      }
    }
    Json args = Json::arr();
    if (e->a != nullptr) {
      for (size_t i = 0; i < e->a->size(); ++i) {
        qstring pretty;
        (*e->a)[i].print1(&pretty, fn);
        tag_remove(&pretty);
        args.push(Json::str(pretty.c_str()));
      }
    }
    row["args"] = std::move(args);
    rows.push(std::move(row));
    return 0;
  }
};

Json collect_calls(const Json &args) {
  ea_t start = parse_function_start(args);
  cfuncptr_t cfunc = decompile_ready(start);
  call_collector walk(cfunc);
  walk.max_items = clamp_arg(args, "max", nullptr, 200, 1, 2000);
  walk.apply_to(&cfunc->body, nullptr);
  Json out = Json::obj();
  out["addr"] = Json::str(hex_ea(start));
  out["calls"] = std::move(walk.rows);
  out["truncated"] = Json::boolean(static_cast<int>(out.find("calls")->a.size()) >= walk.max_items);
  return out;
}

Json collect_callees(const Json &args) {
  Json calls = collect_calls(args);
  std::map<std::string, Json> unique;
  const Json *rows = calls.find("calls");
  if (rows != nullptr) {
    for (const Json &row : rows->a) {
      const Json *name = row.find("calleeName");
      const Json *ea = row.find("callee");
      std::string key = ea != nullptr && ea->type == Json::STR ? ea->s : (name != nullptr ? name->s : "");
      if (key.empty() || unique.count(key)) continue;
      Json one = Json::obj();
      one["callee"] = ea != nullptr && ea->type == Json::STR ? Json::str(ea->s) : Json::nul();
      one["calleeName"] = name != nullptr ? Json::str(name->s) : Json::str("");
      unique.emplace(key, std::move(one));
    }
  }
  Json data = Json::arr();
  for (auto &kv : unique) data.push(std::move(kv.second));
  Json out = Json::obj();
  const Json *addr = calls.find("addr");
  if (addr != nullptr) out["addr"] = Json::str(addr->s);
  out["callees"] = std::move(data);
  return out;
}

Json list_lvars(const Json &args) {
  ea_t start = parse_function_start(args);
  cfuncptr_t cfunc = decompile_ready(start);
  Json lvars = Json::arr();
  lvars_t *vars = cfunc->get_lvars();
  if (vars != nullptr) {
    for (size_t i = 0; i < vars->size(); ++i) lvars.push(lvar_row((*vars)[i]));
  }
  Json out = Json::obj();
  out["addr"] = Json::str(hex_ea(start));
  out["lvars"] = std::move(lvars);
  return out;
}

Json list_strings(const Json &args) {
  int offset = clamp_arg(args, "offset", nullptr, 0, 0, 1000000);
  int count = clamp_arg(args, "count", nullptr, 100, 1, 1000);
  std::string filter = lower_copy(args.str_or("filter"));
  build_strlist();
  std::vector<Json> rows;
  size_t qty = get_strlist_qty();
  for (size_t i = 0; i < qty; ++i) {
    string_info_t item;
    if (!get_strlist_item(&item, i)) continue;
    qstring text;
    get_strlit_contents(&text, item.ea, static_cast<size_t>(item.length), item.type, 0);
    if (!filter.empty() && lower_copy(text.c_str()).find(filter) == std::string::npos) continue;
    Json row = Json::obj();
    row["addr"] = Json::str(hex_ea(item.ea));
    row["text"] = Json::str(text.c_str());
    rows.push_back(std::move(row));
  }
  Json page = Json::arr();
  for (int i = offset; i < static_cast<int>(rows.size()) && i < offset + count; ++i) page.push(rows[static_cast<size_t>(i)]);
  Json out = Json::obj();
  out["data"] = std::move(page);
  out["nextOffset"] = (offset + count < static_cast<int>(rows.size())) ? Json::integer(offset + count) : Json::nul();
  out["total"] = Json::integer(static_cast<long long>(rows.size()));
  return out;
}

struct ImportAcc {
  std::string filter;
  std::string module;
  std::vector<Json> *rows;
};

int idaapi import_cb(ea_t ea, const char *name, uval_t ordinal, void *param) {
  auto *acc = static_cast<ImportAcc *>(param);
  std::string import_name = name != nullptr && name[0] != 0 ? name : ("#" + std::to_string(static_cast<unsigned long long>(ordinal)));
  std::string folded = lower_copy(import_name);
  std::string module_folded = lower_copy(acc->module);
  if (acc->filter.empty() || folded.find(acc->filter) != std::string::npos || module_folded.find(acc->filter) != std::string::npos) {
    Json row = Json::obj();
    row["addr"] = Json::str(hex_ea(ea));
    row["name"] = Json::str(import_name);
    row["module"] = Json::str(acc->module);
    acc->rows->push_back(std::move(row));
  }
  return 1;
}

Json list_imports(const Json &args) {
  int offset = clamp_arg(args, "offset", nullptr, 0, 0, 1000000);
  int count = clamp_arg(args, "count", nullptr, 100, 1, 1000);
  std::vector<Json> rows;
  ImportAcc acc{lower_copy(args.str_or("filter")), "", &rows};
  uint modules = get_import_module_qty();
  for (uint i = 0; i < modules; ++i) {
    qstring module;
    if (!get_import_module_name(&module, static_cast<int>(i)) || module.empty()) module = "<unnamed>";
    acc.module = module.c_str();
    enum_import_names(static_cast<int>(i), import_cb, &acc);
  }
  Json page = Json::arr();
  for (int i = offset; i < static_cast<int>(rows.size()) && i < offset + count; ++i) page.push(rows[static_cast<size_t>(i)]);
  Json out = Json::obj();
  out["data"] = std::move(page);
  out["nextOffset"] = (offset + count < static_cast<int>(rows.size())) ? Json::integer(offset + count) : Json::nul();
  out["total"] = Json::integer(static_cast<long long>(rows.size()));
  return out;
}

Json get_bytes_tool(const Json &args) {
  ea_t ea = parse_address(addr_arg(args));
  int size = clamp_arg(args, "size", nullptr, 16, 1, 1048576);
  std::vector<unsigned char> buf(static_cast<size_t>(size));
  ssize_t got = get_bytes(buf.data(), size, ea);
  if (got < 0) throw std::runtime_error("unable to read bytes at " + hex_ea(ea));
  buf.resize(static_cast<size_t>(got));
  static const char *hexd = "0123456789abcdef";
  std::string hex;
  std::string ascii;
  hex.reserve(buf.size() * 2);
  for (unsigned char b : buf) {
    hex.push_back(hexd[b >> 4]);
    hex.push_back(hexd[b & 0xF]);
    ascii.push_back(b >= 32 && b < 127 ? static_cast<char>(b) : '.');
  }
  Json out = Json::obj();
  out["addr"] = Json::str(hex_ea(ea));
  out["size"] = Json::integer(static_cast<long long>(buf.size()));
  out["hex"] = Json::str(hex);
  out["ascii"] = Json::str(ascii);
  return out;
}

Json rename_tool(const Json &args) {
  std::string kind = lower_copy(args.str_or("kind", "function"));
  std::string name = args.str_or("newName");
  while (!name.empty() && std::isspace(static_cast<unsigned char>(name.front()))) name.erase(name.begin());
  while (!name.empty() && std::isspace(static_cast<unsigned char>(name.back()))) name.pop_back();
  if (name.empty()) throw std::runtime_error("newName is required");
  Json out = Json::obj();
  out["kind"] = Json::str(kind);
  out["newName"] = Json::str(name);
  if (kind == "lvar") {
    ea_t start = parse_function_start(args);
    std::string old_name = args.str_or("oldName");
    if (old_name.empty()) throw std::runtime_error("oldName is required for kind=lvar");
    if (!ensure_hexrays()) throw std::runtime_error("Hex-Rays plugin is not available for this processor");
    bool ok = rename_lvar(start, old_name.c_str(), name.c_str());
    if (ok) g_dirty = true;
    out["addr"] = Json::str(hex_ea(start));
    out["oldName"] = Json::str(old_name);
    out["ok"] = Json::boolean(ok);
    return out;
  }
  ea_t ea = parse_address(addr_arg(args));
  bool ok = set_name(ea, name.c_str(), SN_CHECK);
  if (ok) g_dirty = true;
  out["addr"] = Json::str(hex_ea(ea));
  out["ok"] = Json::boolean(ok);
  return out;
}

Json set_comment_tool(const Json &args) {
  ea_t ea = parse_address(addr_arg(args));
  std::string text = args.str_or("text");
  bool repeatable = args.bool_or("repeatable", false);
  bool ok = set_cmt(ea, text.c_str(), repeatable);
  if (ok) g_dirty = true;
  Json out = Json::obj();
  out["addr"] = Json::str(hex_ea(ea));
  out["repeatable"] = Json::boolean(repeatable);
  out["ok"] = Json::boolean(ok);
  return out;
}

void strip_tags(const std::string &tagged, std::string &text, std::vector<char> &comment) {
  text.clear();
  comment.clear();
  bool in_comment = false;
  const char codes[] = {SCOLOR_REGCMT[0], SCOLOR_RPTCMT[0], SCOLOR_AUTOCMT[0], SCOLOR_COLLAPSED[0]};
  for (size_t i = 0; i < tagged.size();) {
    char ch = tagged[i];
    if (ch == COLOR_ON && i + 1 < tagged.size()) {
      char code = tagged[i + 1];
      in_comment = code == codes[0] || code == codes[1] || code == codes[2] || code == codes[3];
      i += 2;
      continue;
    }
    if (ch == COLOR_OFF) {
      in_comment = false;
      i += (i + 1 < tagged.size()) ? 2 : 1;
      continue;
    }
    text.push_back(ch);
    comment.push_back(in_comment ? 1 : 0);
    ++i;
  }
}

struct Span { size_t start; size_t end; };

std::vector<Span> literal_spans(const std::string &text, const std::string &needle) {
  std::vector<Span> spans;
  if (needle.empty()) return spans;
  size_t offset = 0;
  while (offset <= text.size()) {
    size_t index = text.find(needle, offset);
    if (index == std::string::npos) break;
    spans.push_back({index, index + needle.size()});
    offset = index + needle.size();
  }
  return spans;
}

Json search_text(const Json &args) {
  std::string pattern = args.str_or("pattern");
  auto empty = [&](const std::string &error, bool with_scan) {
    Json out = Json::obj();
    out["pattern"] = Json::str(pattern);
    out["n"] = Json::integer(0);
    out["hits"] = Json::arr();
    Json cursor = Json::obj();
    cursor["done"] = Json::boolean(true);
    out["cursor"] = std::move(cursor);
    if (!error.empty()) out["error"] = Json::str(error);
    if (with_scan) {
      out["scanned"] = Json::integer(0);
      out["partial"] = Json::boolean(false);
    }
    return out;
  };
  if (pattern.empty()) return empty("pattern is required", false);
  int limit = clamp_arg(args, "limit", "max", 30, 1, 500);
  int scan_limit = clamp_arg(args, "scanLimit", "scan_limit", 25000, 100, 250000);
  int time_limit_ms = clamp_arg(args, "timeLimitMs", "time_limit_ms", 10000, 250, 60000);
  bool regex_mode = args.bool_or("regex", false);
  bool case_sensitive = args.bool_or("caseSensitive", args.bool_or("case_sensitive", false));
  std::string include = lower_copy(args.str_or("include", "all"));
  bool code_only = args.bool_or("codeOnly", args.bool_or("code_only", true));
  if (include != "disasm" && include != "comments" && include != "all") return empty("invalid include: '" + include + "'", false);
  bool want_disasm = include == "disasm" || include == "all";
  bool want_comments = include == "comments" || include == "all";

  std::unique_ptr<std::regex> compiled;
  std::string needle = case_sensitive ? pattern : lower_copy(pattern);
  if (regex_mode) {
    try {
      auto flags = std::regex::ECMAScript | (case_sensitive ? std::regex::flag_type{} : std::regex::icase);
      compiled = std::make_unique<std::regex>(pattern, flags);
    } catch (const std::regex_error &exc) {
      return empty(std::string("invalid regex: ") + exc.what(), false);
    }
  }

  struct Range { ea_t start; ea_t end; };
  std::vector<Range> segments;
  for (int i = 0; i < get_segm_qty(); ++i) {
    segment_t *seg = getnseg(i);
    if (seg == nullptr) continue;
    if (code_only && (seg->perm & SEGPERM_EXEC) == 0) continue;
    segments.push_back({seg->start_ea, seg->end_ea});
  }
  if (segments.empty()) return empty("", true);

  ea_t cursor = segments[0].start;
  if (const Json *start = args.find("start"); start != nullptr && start->type == Json::STR && !start->s.empty()) {
    try {
      cursor = parse_address(start);
    } catch (const std::exception &exc) {
      return empty(std::string("invalid start: ") + exc.what(), false);
    }
  }
  size_t segment_index = 0;
  while (segment_index < segments.size() && segments[segment_index].end <= cursor) ++segment_index;
  if (segment_index < segments.size() && cursor < segments[segment_index].start) cursor = segments[segment_index].start;

  Json hits = Json::arr();
  int scanned = 0;
  bool partial = false;
  std::string stop_reason;
  ea_t next_cursor = BADADDR;
  auto deadline = std::chrono::steady_clock::now() + std::chrono::milliseconds(time_limit_ms);

  while (segment_index < segments.size() && static_cast<int>(hits.a.size()) < limit) {
    Range seg = segments[segment_index];
    ea_t head = std::max(seg.start, cursor);
    while (head < seg.end && static_cast<int>(hits.a.size()) < limit) {
      if (scanned >= scan_limit) {
        next_cursor = head;
        stop_reason = "scanLimit";
        partial = true;
        break;
      }
      if (std::chrono::steady_clock::now() >= deadline) {
        next_cursor = head;
        stop_reason = "timeLimit";
        partial = true;
        break;
      }
      ++scanned;
      qstrvec_t generated;
      int lnnum = 0;
      generate_disassembly(&generated, &lnnum, head, 1, 0);
      Json matches = Json::arr();
      for (size_t i = 0; i < generated.size(); ++i) {
        std::string text;
        std::vector<char> mask;
        strip_tags(generated[i].c_str(), text, mask);
        std::vector<Span> spans;
        if (compiled) {
          for (std::sregex_iterator it(text.begin(), text.end(), *compiled), end; it != end; ++it) {
            spans.push_back({static_cast<size_t>(it->position()), static_cast<size_t>(it->position() + it->length())});
          }
        } else {
          spans = literal_spans(case_sensitive ? text : lower_copy(text), needle);
        }
        if (text.empty() || spans.empty()) continue;
        bool disasm_hit = false;
        bool comment_hit = false;
        for (const Span &span : spans) {
          size_t from = std::min(span.start, mask.size());
          size_t to = std::max(from, std::min(span.end, mask.size()));
          if (from == to) {
            if (from < mask.size()) to = from + 1;
            else if (!mask.empty()) from = mask.size() - 1;
          }
          for (size_t k = from; k < to && k < mask.size(); ++k) {
            if (mask[k]) comment_hit = true;
            else disasm_hit = true;
          }
        }
        if (disasm_hit && want_disasm) {
          Json line = Json::obj();
          line["kind"] = Json::str("disasm");
          line["text"] = Json::str(text);
          matches.push(std::move(line));
        }
        if (comment_hit && want_comments) {
          Json line = Json::obj();
          line["kind"] = Json::str("comment");
          line["text"] = Json::str(text);
          matches.push(std::move(line));
        }
      }
      asize_t step = std::max<asize_t>(1, get_item_size(head));
      if (!matches.a.empty()) {
        Json hit = Json::obj();
        hit["addr"] = Json::str(hex_ea(head));
        hit["matches"] = matches;
        std::string joined;
        for (size_t i = 0; i < matches.a.size(); ++i) {
          if (i) joined.push_back('\n');
          joined += matches.a[i].str_or("text");
        }
        hit["text"] = Json::str(joined);
        func_t *func = get_func(head);
        if (func != nullptr) hit["function"] = Json::str(func_name(func->start_ea));
        segment_t *segm = getseg(head);
        if (segm != nullptr) {
          qstring seg_name;
          get_segm_name(&seg_name, segm);
          if (!seg_name.empty()) hit["segment"] = Json::str(seg_name.c_str());
        }
        hits.push(std::move(hit));
        if (static_cast<int>(hits.a.size()) >= limit) {
          next_cursor = head + step;
          partial = true;
        }
      }
      head += step;
    }
    if (partial) break;
    ++segment_index;
    if (segment_index < segments.size()) cursor = segments[segment_index].start;
  }

  Json out = Json::obj();
  out["pattern"] = Json::str(pattern);
  out["n"] = Json::integer(static_cast<long long>(hits.a.size()));
  out["hits"] = std::move(hits);
  Json cursor_obj = Json::obj();
  if (partial) cursor_obj["next"] = Json::str(hex_ea(next_cursor));
  else cursor_obj["done"] = Json::boolean(true);
  out["cursor"] = std::move(cursor_obj);
  out["truncated"] = Json::boolean(partial);
  out["partial"] = Json::boolean(partial);
  out["stopReason"] = stop_reason.empty() ? Json::nul() : Json::str(stop_reason);
  out["scanned"] = Json::integer(scanned);
  out["scanLimit"] = Json::integer(scan_limit);
  out["timeLimitMs"] = Json::integer(time_limit_ms);
  out["engine"] = Json::str("bounded-scan");
  return out;
}

Json metadata(const std::string &instance_id, const std::string &input_path) {
  char root[1024] = {};
  get_root_filename(root, sizeof(root));
  std::string binary = root[0] ? root : ("idalib-" + std::to_string(getpid()));
  char input[4096] = {};
  get_input_file_path(input, sizeof(input));
  char version[128] = {};
  get_kernel_version(version, sizeof(version));
  const char *idb = get_path(PATH_TYPE_IDB);
  atype_t state = get_auto_state();
  std::string state_name = auto_state_name(state);
  bool complete = auto_is_ok();
  Json out = Json::obj();
  out["instanceId"] = Json::str(instance_id);
  out["alias"] = Json::str(alias_from_name(binary));
  out["processId"] = Json::integer(getpid());
  out["binaryName"] = Json::str(binary);
  out["inputPath"] = input[0] ? Json::str(input) : Json::str(input_path);
  out["databasePath"] = idb != nullptr && idb[0] ? Json::str(idb) : Json::nul();
  out["idaVersion"] = version[0] ? Json::str(version) : Json::nul();
  out["platform"] = Json::str("darwin");
  out["backend"] = Json::str("idalib");
  out["sessionState"] = Json::str("ready");
  out["databaseDirty"] = Json::boolean(g_dirty);
  out["workerId"] = Json::nul();
  out["autoAnalysisEnabled"] = Json::boolean(g_pump);
  out["autoAnalysisComplete"] = Json::boolean(complete);
  out["autoAnalysisState"] = Json::str(state_name);
  out["autoAnalysisRunning"] = Json::boolean((g_pump && !complete) || state_name != "AU_NONE");
  Json caps = Json::obj();
  caps["decompiler"] = Json::boolean(ensure_hexrays());
  caps["write"] = Json::boolean(true);
  caps["save"] = Json::boolean(true);
  out["capabilities"] = std::move(caps);
  return out;
}

Json save_tool() {
  if (!g_open) throw std::runtime_error("no database is open");
  const char *path = get_path(PATH_TYPE_IDB);
  if (!save_database(path, 0)) throw std::runtime_error("save_database failed");
  g_dirty = false;
  Json out = Json::obj();
  out["ok"] = Json::boolean(true);
  out["databasePath"] = path != nullptr && path[0] ? Json::str(path) : Json::nul();
  return out;
}

Json set_pump(bool enabled, const std::string &instance_id, const std::string &input_path) {
  if (!g_open) throw std::runtime_error("no database is open");
  bool complete = auto_is_ok();
  if (enabled && !complete) {
    enable_auto(true);
    g_pump = true;
  } else {
    g_pump = false;
    enable_auto(false);
    complete = auto_is_ok();
  }
  Json out = Json::obj();
  out["ok"] = Json::boolean(true);
  out["pumping"] = Json::boolean(g_pump);
  out["complete"] = Json::boolean(complete);
  out["metadata"] = metadata(instance_id, input_path);
  return out;
}

}  // namespace

bool open_database(const std::string &path, std::string &error) {
  char arg0[] = "jig-idalib";
  char *argv[] = {arg0, nullptr};
  if (init_library(1, argv) != 0) {
    error = "init_library failed";
    return false;
  }
  enable_console_messages(false);
  msg_clear();
  msg_sink sink;
  hook_event_listener(HT_UI, &sink, nullptr, HKCB_GLOBAL);
  std::string rejected = loaders_or_error(path);
  if (!rejected.empty()) {
    unhook_event_listener(HT_UI, &sink);
    error = rejected;
    return false;
  }
  int code = ::open_database(path.c_str(), false, "-A");
  std::string why = trim_copy(sink.text);
  if (why.empty()) why = output_lines();
  unhook_event_listener(HT_UI, &sink);
  if (code != 0) {
    if (why.empty()) why = "open_database 失败 (code=" + std::to_string(code) + ")";
    else why += " (code=" + std::to_string(code) + ")";
    error = why;
    return false;
  }
  g_open = true;
  g_dirty = false;
  ensure_hexrays();
  return true;
}

void close_database() {
  if (!g_open) return;
  if (g_hexrays) {
    term_hexrays_plugin();
    g_hexrays = false;
  }
  // Open leaves .id0/.id1/.id2/.nam/.til beside the .i64. Pack them back
  // and delete the spread files. IDA's default PACK_DATABASE is store.
  if (inf_get_pack_mode() == IDB_UNPACKED) inf_set_pack_mode(IDB_PACKED);
  set_database_flag(DBFL_KILL, true);
  const char *path = get_path(PATH_TYPE_IDB);
  if (!save_database(path != nullptr && path[0] ? path : nullptr, DBFL_KILL)) {
    std::cerr << "[jig-idalib] save_database failed; unpacked components may remain\n";
  }
  ::close_database(false);
  g_open = false;
  g_pump = false;
}

int pump_auto_steps(int max_steps) {
  if (!g_open || !g_pump) return 0;
  if (auto_is_ok()) {
    g_pump = false;
    enable_auto(false);
    return 0;
  }
  ea_t min_ea = inf_get_min_ea();
  ea_t max_ea = inf_get_max_ea();
  int processed = 0;
  for (int i = 0; i < max_steps; ++i) {
    if (!auto_make_step(min_ea, max_ea)) break;
    ++processed;
    if (auto_is_ok()) {
      g_pump = false;
      enable_auto(false);
      break;
    }
  }
  return processed;
}

Json execute_tool(const std::string &name, const Json &args, const std::string &instance_id, const std::string &input_path) {
  if (name == "target.ping") {
    Json out = Json::obj();
    out["status"] = Json::str("ok");
    out["instanceId"] = Json::str(instance_id);
    out["backend"] = Json::str("idalib");
    return out;
  }
  if (name == "target.get_metadata") return metadata(instance_id, input_path);
  if (name == "session.start_auto_analysis") return set_pump(true, instance_id, input_path);
  if (name == "session.stop_auto_analysis") return set_pump(false, instance_id, input_path);
  if (name == "session.save") return save_tool();
  if (name == "session.close") {
    bool save = args.bool_or("save", false);
    Json out = Json::obj();
    if (save) save_tool();
    close_database();
    out["ok"] = Json::boolean(true);
    out["saved"] = Json::boolean(save);
    out["exit"] = Json::boolean(true);
    return out;
  }
  if (name == "analysis.list_functions") return list_functions(args);
  if (name == "analysis.get_function") return get_function(args);
  if (name == "analysis.decompile") return decompile_function(args);
  if (name == "analysis.disassemble") return disassemble(args);
  if (name == "analysis.xrefs") return xrefs(args);
  if (name == "analysis.calls") return collect_calls(args);
  if (name == "analysis.callees") return collect_callees(args);
  if (name == "analysis.lvars") return list_lvars(args);
  if (name == "analysis.list_strings") return list_strings(args);
  if (name == "analysis.list_imports") return list_imports(args);
  if (name == "analysis.get_bytes") return get_bytes_tool(args);
  if (name == "analysis.rename") return rename_tool(args);
  if (name == "analysis.set_comment") return set_comment_tool(args);
  if (name == "analysis.search_text") return search_text(args);
  throw std::runtime_error("unknown plugin tool: " + name);
}

}  // namespace jig
