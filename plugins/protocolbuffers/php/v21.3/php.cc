#include <google/protobuf/compiler/php/php_generator.h>
#include <google/protobuf/compiler/plugin.h>

int main(int argc, char *argv[]) {
  google::protobuf::compiler::php::Generator generator;
  return google::protobuf::compiler::PluginMain(argc, argv, &generator);
}