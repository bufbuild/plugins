package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPluginKeyFromPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		path string
		key  pluginNameVersion
		ok   bool
	}{
		{
			name: "yaml",
			path: "plugins/bufbuild/connect-go/v1.0.0/buf.plugin.yaml",
			key:  pluginNameVersion{name: "bufbuild/connect-go", version: "v1.0.0"},
			ok:   true,
		},
		{
			name: "dockerfile",
			path: "plugins/grpc-ecosystem/grpc-gateway/v2.15.0/Dockerfile",
			ok:   false,
		},
		{
			name: "invalid_version",
			path: "plugins/bufbuild/connect-go/foo/buf.plugin.yaml",
			ok:   false,
		},
		{
			name: "outside_plugins_dir",
			path: "README.md",
			ok:   false,
		},
		{
			name: "empty",
			path: "",
			ok:   false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, ok := pluginKeyFromPath(tc.path)
			assert.Equal(t, tc.ok, ok)
			if tc.ok {
				assert.Equal(t, tc.key, got)
			}
		})
	}
}
