package fetchclient

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchPyPI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		versions       []string
		ignoreVersions map[string]struct{}
		maxVersion     string
		wantVersion    string
		wantErr        string
	}{
		{
			name:        "returns latest semver version",
			versions:    []string{"3.5.0", "3.6.0", "5.0.0", "1.0"},
			wantVersion: "v5.0.0",
		},
		{
			name:        "skips pre-release versions",
			versions:    []string{"1.2.5", "2.0.0b7"},
			wantVersion: "v1.2.5",
		},
		{
			name:           "respects ignore_versions",
			versions:       []string{"3.6.0", "5.0.0"},
			ignoreVersions: map[string]struct{}{"v5.0.0": {}},
			wantVersion:    "v3.6.0",
		},
		{
			name:        "respects max_version exclusive upper bound",
			versions:    []string{"3.6.0", "5.0.0"},
			maxVersion:  "v5.0.0",
			wantVersion: "v3.6.0",
		},
		{
			name:     "error when no valid versions remain",
			versions: []string{"2.0.0b7", "2.0.0rc1"},
			wantErr:  "no versions found",
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
			ignoreVersions := tt.ignoreVersions
			if ignoreVersions == nil {
				ignoreVersions = map[string]struct{}{}
			}
			got, err := c.fetchPyPI(t.Context(), "mypy-protobuf", ignoreVersions, tt.maxVersion)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantVersion, got)
		})
	}
}
