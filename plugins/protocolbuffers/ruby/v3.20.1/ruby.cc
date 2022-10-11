#include <google/protobuf/compiler/ruby/ruby_generator.h>
#include <google/protobuf/compiler/plugin.h>

int main(int argc, char *argv[]) {
  google::protobuf::compiler::ruby::Generator generator;
  return google::protobuf::compiler::PluginMain(argc, argv, &generator);
}