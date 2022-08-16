#include <google/protobuf/compiler/java/java_kotlin_generator.h>
#include <google/protobuf/compiler/plugin.h>

int main(int argc, char *argv[]) {
  google::protobuf::compiler::java::KotlinGenerator generator;
  return google::protobuf::compiler::PluginMain(argc, argv, &generator);
}