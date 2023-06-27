# plugins

## Description

This repository contains plugins published to the Buf Schema Registry.
Each plugin consists of a `buf.plugin.yaml` file containing metadata and a `Dockerfile` used to build and publish the plugin for remote execution.
Plugins are uniquely identified by the combination of the `name` and `plugin_version` specified in `buf.plugin.yaml` and an auto-generated revision which is incremented each time a modified plugin is pushed to the BSR.

If you'd like a Protobuf plugin to be added to the Buf Schema Registry, open an issue using the 'Plugin Request for Buf Schema Registry' issue template and our team will follow up.

## Community

For help and discussion regarding Protobuf plugins, join us on [Slack](https://buf.build/links/slack).

For feature requests, bugs, or technical questions, email us at [dev@buf.build](dev@buf.build).

## Manual action caveats

When triggering a manual execution of the fetch-versions workflow, you may want to disable the scheduled execution 
temporarily to ensure that any in-flight generated PR is not overridden by the scheduled execution.
