# Contributing

## Description

Currently, the Buf development team supports this project and creates new plugins (or new versions) for users of the Buf Schema Registry.
This document provides details on how to contribute to this repository.
Prior to opening a PR to add a new plugin to this repository, open an issue using the "Plugin Request for Buf Schema Registry" template first.

## Building

The build requires the following:

* [Go](https://go.dev/dl/) (1.22+)
* [Buf](https://github.com/bufbuild/buf) (1.28.1+)
* [yq](https://github.com/mikefarah/yq)

### Makefile targets

* Build and test individual plugin(s):
  * Specific version: `make test PLUGINS="connectrpc/go:v0.4.0"`
  * Latest versions: `make test PLUGINS="connectrpc/go:latest,connectrpc/es:latest"`.
  * All versions: `make PLUGINS="connectrpc/go"`.
* Remove intermediate state from previous builds: `make clean`.
* Push plugins to the BSR (locked down to CI/CD): `make push`.

## Creating a new plugin

Plugins are found in the top-level [plugins](plugins) directory. To create a new plugin:

1. Add a new folder with the plugin's organization, name, and version (i.e. `mkdir -p plugins/<org>/<name>/<version>`) and add a `buf.plugin.yaml` / `Dockerfile` / `.dockerignore` to the newly created directory.
  Ensure the version begins with the `v` prefix.
2. Build the plugin's Docker image with `make PLUGINS="<org>/<name>"`.
3. Verify the plugin with `make test PLUGINS="<org>/<name>"`.
   This runs the plugin against images stored in `tests/testdata/images/` and verifies that the plugin contains essential information in the `buf.plugin.yaml` file. See [Plugin Verification](#plugin-verification) for more details.
4. Create a [source.yaml](#sourceyaml) at the top-level directory (i.e. `plugins/<org>/<name>`) with information on how to detect new plugin versions.

### Plugin Verification

When a plugin is executed for the first time with `make test`, it will create the following file(s):

* `tests/testdata/buf.build/<org>/<name>/<version>/<image>/plugin.sum`

After verifying the generated code from the plugin in `tests/testdata/buf.build/<org>/<name>/<version>/<image>/gen`, these file(s) should be checked into source control to ensure the CI tests pass.
This file contains a directory checksum of the generated code for the plugin and is checked in to ensure that generated code is consistent across multiple executions of the plugin.

### source.yaml

The `source.yaml` file for each plugin defines how new versions of the plugins should be detected.
Supported sources include:

**github**

```yaml
source:
  github:
    owner: <owner>
    repository: <repo>
```

**dart_flutter**
```yaml
source:
  dart_flutter:
    name: <package_name>
```

**goproxy**
```yaml
source:
  goproxy:
    name: <module_name>
```

**npm_registry**
```yaml
source:
  npm_registry:
    name: <package_name>
```

**maven**
```yaml
source:
  maven:
    group: <groupId>
    name: <artifactId>
```

**crates**
```yaml
source:
  crates:
    crate_name: <crate_name>
```

## Plugin Authoring Best Practices

* Use multi-stage builds to optimize image size. (Recommended to use `scratch` or [distroless](https://github.com/GoogleContainerTools/distroless) as runtime images).
* Always include a [.dockerignore](https://docs.docker.com/engine/reference/builder/#dockerignore-file) alongside the Dockerfile to minimize the Docker build context size (and avoid cache misses during builds). See the following for some examples based on the Protobuf plugin language type:
    * Generic: [plugins/protocolbuffers/go/v1.31.0/.dockerignore](plugins/protocolbuffers/go/v1.31.0/.dockerignore)
    * NPM/Node: [plugins/connectrpc/es/v1.1.4/.dockerignore](plugins/connectrpc/es/v1.1.4/.dockerignore)
* Builds should be reproducible. All Docker images used for builds should use a specific tag (i.e. `debian:bullseye-YYYYMMDD` instead of `debian:bullseye`, `debian`, or `latest`). Distroless builds don't have tags so should depend on the sha256 of the image.
    * NPM/Node: A `package.json` and `package-lock.json` file should be checked in and `npm ci` should be used during installation to ensure consistent dependencies are installed.
    * Python: A `requirements.txt` should be checked in (created initially within a virtualenv with `pip freeze`).
    * Go: Compilation should use `-trimpath`.

### `buf.plugin.yaml` file

A `buf.plugin.yaml` file captures metadata about the plugin. It includes mandatory and optional
fields that are displayed on the individual plugin page and the BSR plugin homepage at
https://buf.build/plugins.

Note, although some fields are optional, it is suggested to include as many as possible.

**Mandatory:**

* `version`: The YAML configuration version, must be `v1`.
* `name`: the plugin identity with format `{remote}/{organization_name}/{plugin_name}`.
* `plugin_version`: the plugin version with format`v{semver}`, the `v` prefix is required and the
  version must be valid [semantic versioning](https://semver.org/).

**Optional:**

* `source_url`: URL to the source code of the Protobuf plugin.
* `description`: Description of the plugin.
* `output_languages`: The output language types generated by the plugin. See the [PluginLanguage](https://buf.build/bufbuild/buf/docs/main:buf.alpha.registry.v1alpha1#buf.alpha.registry.v1alpha1.PluginLanguage) enum for existing languages. Open a GitHub issue in the [Buf CLI](https://github.com/bufbuild/buf) if the output language isn't found for a plugin.
* `spdx_license_id`: The license id for the plugin from https://spdx.org/licenses/.
* `license_url`: The URL to the plugin's license (should be unique for each release of the plugin).
* `integration_guide_url`: The URL to the integration guide for the plugin.
* `deps`: A list of dependencies on other plugins used by [Generated SDKs]. Each dependency contains:
  * `plugin` (required): The reference to the plugin dependency: `<name>:<plugin_version>`.
  * `revision`: If specified, the dependency will be to a specific version of a plugin.
    It is recommended to leave this off (the plugin will depend on the latest revision at time of publishing).
* `registry`: Configuration to enable a plugin for use with Generated SDKs.
  Must specify one of `go`, `npm`, `maven`, `python`, or `swift`.
  * `opts`: Options supplied to the plugin when generating code for the SDK.
  * `go`: Generated SDK configuration for a Go plugin.
    * `min_version`: The minimum Go version required by the plugin (e.g. `1.19`), used as the [go directive] in the `go.mod` file.
    * `deps`: A list of Go module requirements. Each requirement corresponds to a [require directive] in the `go.mod` file.
      * `module`: Go module name.
      * `version`: Go module version.
  * `npm`: Generated SDK configuration for a JavaScript/TypeScript plugin.
    * `rewrite_import_path_suffix`: The suffix used in the generated files and their imported dependencies (supported by [@bufbuild/protoplugin] plugins).
    * `deps`: NPM package dependencies for the Generated SDK.
      * `package`: The name of the NPM package dependency.
      * `version`: The version of the NPM package dependency (see [npm semantic versioning] for more details).
    * `import_style` (required): One of either `module` or `commonjs`.
  * `maven`:
    * `compiler`:
      * `java`: Java compiler settings.
        * `encoding`: Encoding of source files (default: UTF-8).
        * `release`: Target Java release (default: 8).
        * `source`: Source bytecode level (default: 8).
        * `target`: Target bytecode level (default: 8).
      * `kotlin`: Kotlin compiler settings.
        * `api_version`: Kotlin API version.
        * `jvm_target`: JVM bytecode target version (default: 1.8).
        * `language_version`: Kotlin version source compatibility.
        * `version` (required): Version of the Kotlin compiler.
    * `deps`: Runtime dependencies.
      * Dependencies of the generated Java/Kotlin code (in GAV format).
    * `additional_runtimes`: Configures additional supported runtimes.
      * `name`: The name of the additional runtime. The only known name at this time is `lite` for Protobuf lite runtime support.
      * `deps`: Dependencies for the runtime. These override `registry -> maven -> deps` if specified.
      * `opts`: Plugin options for the additional runtime.
  * `python`:
    * `deps`: Runtime dependencies of the generated code.
    * `requires_python`: Specifies the `Requires-Python` of the generated package.
    * `package_type`: One of `runtime` or `stub-only`.
  * `swift`:
    * `deps`: Dependencies of the generated code.
      * `source`: URL of the Swift package.
      * `package`: Name of the Swift package.
      * `version`: Version of the Swift package.
      * `products`: Products to import.
      * `platforms`:
        * `macos`: Version of the macOS platform.
        * `ios`: Version of the iOS platform.
        * `tvos`: Version of the tvOS platform.
        * `watchos`: Version of the watchOS platform.
      * `swift_versions`: Versions of Swift the package supports.
  * `cargo`:
    * `rust_version`: Minimum Supported Rust Version (MSRV) of the generated crate.
    * `deps`: Runtime dependencies of the generated code.
      * `name`: Name of the dependency.
      * `req`: Version Requirement of the dependency.
      * `default_features`: If [default features][cargo default features] of the dependency are enabled.
      * `features`: List of enabled features.
  * `nuget`:
    * `target_frameworks`: Target Frameworks to build.
    * `deps`: Runtime dependencies of the generated code.
      * `name`: Name of the dependency.
      * `version`: Version of the dependency.
      * `target_frameworks`: Optional list of Target Frameworks this dependency applies to.
  * `cmake`: no current options, but enables the plugin for the CMake registry

## Generated SDK Plugins

Some plugins are compatible with [Generated SDKs], while others require patches.
When building a plugin with support for Generated SDKs, consider the following requirements:

* **Go**
  * Plugins must output files to a separate directory since each Generated SDK contains a Go module containing the plugin's code, and multiple Go modules can't provide the same Go package. This is supported automatically with `connectrpc/go`, but `grpc/go` and `grpc-ecosystem/gateway` require patches to support outputting code to a separate directory.
  * Dependencies are required for any runtime dependencies used by the generated code for `go get` and other tools to work.
* **JavaScript/TypeScript**
  * We strongly recommend use of the `@bufbuild/protoplugin` package and using `import_style=module`.
* **Maven**
  * It is required that all dependencies used by the generated code are included as runtime dependencies in order for the code to compile.
  * Ensure the Java/Kotlin compiler settings are accurate to avoid compilation failures.

## CI/CD

Builds use [tj-actions/changed-files](https://github.com/tj-actions/changed-files) to determine which plugin(s) need to be rebuilt.
See [.github/workflows/pr.yaml](.github/workflows/pr.yml) and [internal/cmd/changed-plugins/main.go](internal/cmd/changed-plugins/main.go) for more details.

We use a combination of a custom command ([internal/cmd/fetcher/main.go](internal/cmd/fetcher/main.go)) and Dependabot to keep dependencies up to date in the project.
The `fetcher` command will use `source.yaml` files in each plugin to determine if new plugin versions are available.
Dependabot is used to determine if base Docker images are up-to-date with bug/security fixes.

### Caching

Main branch builds publish Docker images to:

* `ghcr.io/bufbuild/plugins-<org>-<name>:<version>` (Plugin image)

These images are used only for caching - the authoritative images used for plugin execution are pushed to the BSR.
Untagged versions of these cached images can be cleaned up at any time (only the latest tagged versions are used).

## Updates

### Creating a new plugin version

If the `fetcher` command opens a PR for a new version of an existing plugin, most steps are automated but make sure to review the following:

* The versions of dependencies on plugins and runtime dependencies under `registry:`.
* Versions of dependencies in `package.json` and `requirements.txt` files.
* The plugin's generated code from `make test` (stored as an artifact of the "Fetch latest versions" workflow).

### Updating Docker Base Images

Docker base images are tracked in [.github/docker](.github/docker) and kept updated with Dependabot.
When new versions of a plugin are detected, they'll automatically be built with the latest base images.
When creating a new plugin, ensure that it starts with the latest version of the base image in the `.github/docker` directory.

## Local Testing

When testing locally, you may wish to build for a different architecture or push plugins to a different instance of the BSR.
For example:

```
$ make push BUF_PLUGIN_PUSH_ARGS="--override-remote bufbuild.internal"
```

This command can also be used to publish to other instances of the BSR.
This will build with the default architecture of the system by default.
To specify a different architecture (i.e. x86_64), specify the `DOCKER_BUILD_EXTRA_ARGS="--platform linux/amd64"` argument to the build.

[@bufbuild/protoplugin]: https://www.npmjs.com/package/@bufbuild/protoplugin
[go directive]: https://go.dev/ref/mod#go-mod-file-go
[npm semantic versioning]: https://docs.npmjs.com/about-semantic-versioning#using-semantic-versioning-to-specify-update-types-your-package-can-accept
[require directive]: https://go.dev/ref/mod#go-mod-file-require
[Generated SDKs]: https://buf.build/docs/bsr/generated-sdks/overview
[cargo default features]: https://doc.rust-lang.org/cargo/reference/features.html#the-default-feature
