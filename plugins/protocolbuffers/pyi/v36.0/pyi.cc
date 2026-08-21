#include <google/protobuf/compiler/python/pyi_generator.h>
#include <google/protobuf/compiler/plugin.h>

// Standalone binary to generate Python .pyi files
int main(int argc, char *argv[]) {
  google::protobuf::compiler::python::PyiGenerator generator;
  return google::protobuf::compiler::PluginMain(argc, argv, &generator);
}
