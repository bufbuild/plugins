package fetchclient

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pypiRelease mirrors the file-level shape returned by the PyPI JSON API.
// Shape verified against https://pypi.org/pypi/mypy-protobuf/json.
type pypiRelease struct {
	Yanked bool `json:"yanked"`
}

func TestFetchPyPI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		releases       map[string][]pypiRelease
		ignoreVersions map[string]struct{}
		maxVersion     string
		wantVersion    string
		wantErr        string
	}{
		{
			name: "returns latest semver version",
			releases: map[string][]pypiRelease{
				"3.5.0": {{Yanked: false}},
				"3.6.0": {{Yanked: false}},
				"5.0.0": {{Yanked: false}},
				// Go semver accepts these as v1.0.0 and v2.10.0, but 5.0.0 is still highest.
				"1.0":  {{Yanked: false}},
				"2.10": {{Yanked: false}},
				// Python-style pre-release: invalid Go semver, filtered out.
				"2.0.0b7": {{Yanked: false}},
			},
			wantVersion: "v5.0.0",
		},
		{
			name: "skips fully yanked releases",
			releases: map[string][]pypiRelease{
				"3.6.0": {{Yanked: false}},
				"5.0.0": {{Yanked: true}, {Yanked: true}},
			},
			wantVersion: "v3.6.0",
		},
		{
			name: "version with at least one non-yanked file is available",
			releases: map[string][]pypiRelease{
				"3.6.0": {{Yanked: false}},
				"5.0.0": {{Yanked: true}, {Yanked: false}},
			},
			wantVersion: "v5.0.0",
		},
		{
			name: "skips releases with empty file list",
			releases: map[string][]pypiRelease{
				"3.6.0": {{Yanked: false}},
				"5.0.0": {},
			},
			wantVersion: "v3.6.0",
		},
		{
			name: "skips pre-release versions",
			releases: map[string][]pypiRelease{
				"1.2.5":   {{Yanked: false}},
				"2.0.0b7": {{Yanked: false}},
			},
			wantVersion: "v1.2.5",
		},
		{
			name: "respects ignore_versions",
			releases: map[string][]pypiRelease{
				"3.6.0": {{Yanked: false}},
				"5.0.0": {{Yanked: false}},
			},
			ignoreVersions: map[string]struct{}{"v5.0.0": {}},
			wantVersion:    "v3.6.0",
		},
		{
			name: "respects max_version exclusive upper bound",
			releases: map[string][]pypiRelease{
				"3.6.0": {{Yanked: false}},
				"5.0.0": {{Yanked: false}},
			},
			maxVersion:  "v5.0.0",
			wantVersion: "v3.6.0",
		},
		{
			name: "error when no valid versions remain",
			releases: map[string][]pypiRelease{
				// Python-style pre-releases: invalid Go semver, all filtered out.
				"2.0.0b7":  {{Yanked: false}},
				"2.0.0rc1": {{Yanked: false}},
			},
			wantErr: "no versions found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(struct {
					Releases map[string][]pypiRelease `json:"releases"`
				}{Releases: tt.releases}); err != nil {
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
