#include <google/protobuf/compiler/cpp/generator.h>
#include <google/protobuf/compiler/plugin.h>

int main(int argc, char *argv[]) {
  google::protobuf::compiler::cpp::CppGenerator generator;
  return google::protobuf::compiler::PluginMain(argc, argv, &generator);
}
