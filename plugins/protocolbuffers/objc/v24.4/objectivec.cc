#include <google/protobuf/compiler/objectivec/generator.h>
#include <google/protobuf/compiler/plugin.h>

int main(int argc, char *argv[]) {
  google::protobuf::compiler::objectivec::ObjectiveCGenerator generator;
  return google::protobuf::compiler::PluginMain(argc, argv, &generator);
}
