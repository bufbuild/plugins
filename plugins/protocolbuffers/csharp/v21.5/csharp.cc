#include <google/protobuf/compiler/csharp/csharp_generator.h>
#include <google/protobuf/compiler/plugin.h>

int main(int argc, char *argv[]) {
  google::protobuf::compiler::csharp::Generator generator;
  return google::protobuf::compiler::PluginMain(argc, argv, &generator);
}