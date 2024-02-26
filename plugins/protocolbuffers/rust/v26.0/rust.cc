#include <google/protobuf/compiler/rust/generator.h>
#include <google/protobuf/compiler/plugin.h>

// Standalone binary to generate Rust files
int main(int argc, char *argv[]) {
  google::protobuf::compiler::rust::RustGenerator generator;
  return google::protobuf::compiler::PluginMain(argc, argv, &generator);
}
