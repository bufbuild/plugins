#include <google/protobuf/compiler/kotlin/generator.h>
#include <google/protobuf/compiler/plugin.h>

int main(int argc, char *argv[]) {
  google::protobuf::compiler::kotlin::KotlinGenerator generator;
  return google::protobuf::compiler::PluginMain(argc, argv, &generator);
}
