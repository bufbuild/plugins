---
name: Plugin request for Buf Schema Registry
about: Request for plugin
title: Plugin request for Buf Schema Registry
labels: Feature
assignees: ''

---

Not ready to open an issue, but want to chat about your plugin? Come find us on our Public Slack channel:

https://buf.build/links/slack

## Mandatory

**Where is the source code for the plugin?**

Example, the source code for the `protoc-gen-connect-go` plugin is found here:

https://github.com/bufbuild/connect-go/tree/main/cmd/protoc-gen-connect-go


## Optional

**Does the plugin have a valid semver version?**

What is the latest version, and where did you get this value from?

**Does the plugin have a dependency on another plugin?**

Example, the `protoc-gen-connect-go` plugin has a dependency on the base types produced by `protoc-gen-go` which can be found here:

https://github.com/protocolbuffers/protobuf-go/tree/master/cmd/protoc-gen-go

**Does the output of this plugin depend on any other external libraries?**

Example, the generated code for the `protoc-gen-connect-go` plugin has a runtime dependency on the Go module:

https://github.com/bufbuild/connect-go
