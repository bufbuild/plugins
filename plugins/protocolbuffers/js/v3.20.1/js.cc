#include <google/protobuf/compiler/js/js_generator.h>
#include <google/protobuf/compiler/plugin.h>

int main(int argc, char *argv[]) {
  google::protobuf::compiler::js::Generator generator;
  return google::protobuf::compiler::PluginMain(argc, argv, &generator);
}