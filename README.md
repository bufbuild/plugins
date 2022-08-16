# plugins

## Description

This repository contains plugins published to the Buf Schema Registry.
Each plugin is comprised of a `buf.plugin.yaml` file containing metadata and a `Dockerfile` used to build and publish the plugin for remote execution.
Plugins are uniquely identified by the combination of the `name` and `plugin_version` specified in `buf.plugin.yaml` and an auto-generated revision which is incremented each time a modified plugin is pushed to the BSR.

## Building

The build requires the following:

* [Go](https://go.dev/dl/) (1.18+)
* [Buf](https://github.com/bufbuild/buf)
* [yq](https://github.com/mikefarah/yq)

### Makefile targets

* Build all plugin Docker images (including prereqs): `make` or `make build`.
* Run integration tests against Docker images: `make test`.

## Creating a new plugin

To create a new plugin, add a new folder matching the last component of the plugin's name and its version (i.e. `mkdir -p my-plugin-name/vX.Y.Z`) and add a `buf.plugin.yaml` / `Dockerfile` to the newly created directory.
To verify the plugin builds properly, run `make` to build the Docker image and `make test` to verify code generation for the plugin using some basic APIs stored in `tests/testdata/images/`.

When a plugin is executed for the first time, it will create the following file(s):

* `tests/testdata/buf.build/library/<plugin-name>/<plugin-version>/<image>/plugin.sum`

After verifying the generated code from the plugin in `tests/testdata/buf.build/library/<plugin-name>/<plugin-version>/<image>/gen`, these file(s) should be checked into source control to ensure the CI tests pass.
This file contains a directory checksum of the generated code for the plugin and is checked in to ensure that generated code matches the expected output.

## Community

For help and discussion regarding Protobuf plugins, join us on [Slack](https://buf.build/links/slack).

For feature requests, bugs, or technical questions, email us at [dev@buf.build](dev@buf.build). 
