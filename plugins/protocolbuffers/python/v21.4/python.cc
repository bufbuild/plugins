#include <google/protobuf/compiler/python/python_generator.h>
#include <google/protobuf/compiler/plugin.h>

int main(int argc, char *argv[]) {
  google::protobuf::compiler::python::Generator generator;
  return google::protobuf::compiler::PluginMain(argc, argv, &generator);
}