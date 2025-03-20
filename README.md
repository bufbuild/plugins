# plugins

## Description

This repository contains plugins published to the Buf Schema Registry.
Each plugin consists of a `buf.plugin.yaml` file containing metadata and a `Dockerfile` used to build and publish the plugin for remote execution.
Plugins are uniquely identified by the combination of the `name` and `plugin_version` specified in `buf.plugin.yaml` and an auto-generated revision which is incremented each time a modified plugin is pushed to the BSR.

If you'd like a Protobuf plugin to be added to the Buf Schema Registry, open an issue using the 'Plugin Request for Buf Schema Registry' issue template and our team will follow up. Please note that we are more likely to accept plugins that are widely adopted, stable, well-documented, and well-maintained with clear owners.

If you have a plugin that you'd like to use but that we don't include here, you can upload [custom plugins](https://buf.build/docs/bsr/remote-plugins/custom-plugins) if you are on Pro or Enterprise plan.

## Community

For help and discussion regarding Protobuf plugins, join us on [Slack](https://buf.build/links/slack).

For feature requests, bugs, or technical questions, email us at [dev@buf.build](dev@buf.build).

