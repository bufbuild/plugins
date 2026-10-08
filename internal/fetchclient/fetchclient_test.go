package fetchclient

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchPyPI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		versions     []string
		wantVersions []string
	}{
		{
			name:         "returns all semver versions with v prefix",
			versions:     []string{"3.5.0", "3.6.0", "5.0.0", "1.0"},
			wantVersions: []string{"v3.5.0", "v3.6.0", "v5.0.0", "v1.0"},
		},
		{
			name:         "skips pre-release versions",
			versions:     []string{"1.2.5", "2.0.0b7", "2.0.0rc1"},
			wantVersions: []string{"v1.2.5"},
		},
		{
			name:         "skips versions with build metadata",
			versions:     []string{"1.2.5", "1.2.5+1", "1.3.0+build.7"},
			wantVersions: []string{"v1.2.5"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/vnd.pypi.simple.v1+json")
				if err := json.NewEncoder(w).Encode(struct {
					Versions []string `json:"versions"`
				}{Versions: tt.versions}); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
			}))
			t.Cleanup(srv.Close)

			c := &Client{
				httpClient:  srv.Client(),
				pypiBaseURL: srv.URL,
			}
			got, err := c.fetchPyPI(t.Context(), "mypy-protobuf")
			require.NoError(t, err)
			assert.Equal(t, tt.wantVersions, got)
		})
	}
}

func TestFetchGoProxy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		list         string
		mod          string
		wantVersions []string
		wantErr      string
	}{
		{
			name:         "returns tagged versions",
			list:         "v1.1.0\nv1.0.0\n",
			mod:          "module github.com/Example/mod\n",
			wantVersions: []string{"v1.1.0", "v1.0.0"},
		},
		{
			name:         "skips prerelease and incompatible versions",
			list:         "v1.0.0\nv1.1.0-rc.1\nv2.0.0+incompatible\n",
			mod:          "module github.com/Example/mod\n",
			wantVersions: []string{"v1.0.0"},
		},
		{
			name: "skips versions retracted by the highest version",
			list: "v1.0.0\nv1.1.0\nv1.2.0\nv1.3.0\nv1.4.0\n",
			mod: `module github.com/Example/mod

retract (
	v1.4.0 // retracts itself
	[v1.1.0, v1.2.0]
)
`,
			wantVersions: []string{"v1.0.0", "v1.3.0"},
		},
		{
			name:         "empty list",
			list:         "",
			wantVersions: nil,
		},
		{
			name:    "invalid go.mod",
			list:    "v1.0.0\n",
			mod:     "retract (\n",
			wantErr: "failed to parse",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body string
				switch {
				case r.URL.Path == "/github.com/!example/mod/@v/list":
					body = tt.list
				case strings.HasPrefix(r.URL.Path, "/github.com/!example/mod/@v/") && strings.HasSuffix(r.URL.Path, ".mod"):
					body = tt.mod
				default:
					http.NotFound(w, r)
					return
				}
				_, _ = io.WriteString(w, body)
			}))
			t.Cleanup(srv.Close)

			c := &Client{
				httpClient:     srv.Client(),
				goProxyBaseURL: srv.URL,
			}
			got, err := c.fetchGoProxy(t.Context(), "github.com/Example/mod")
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantVersions, got)
		})
	}
}
