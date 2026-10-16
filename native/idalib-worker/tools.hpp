#pragma once

#include "json.hpp"

#include <string>

namespace jig {

bool open_database(const std::string &path, std::string &error);
void close_database();
Json execute_tool(const std::string &name, const Json &args, const std::string &instance_id, const std::string &input_path);
int pump_auto_steps(int max_steps);

}  // namespace jig
