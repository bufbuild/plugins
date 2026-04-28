package fetchclient

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pypiTestFile mirrors the file entry in the PyPI Simple API JSON response.
// Shape verified against https://pypi.org/simple/mypy-protobuf/ with
// Accept: application/vnd.pypi.simple.v1+json.
type pypiTestFile struct {
	Filename string          `json:"filename"`
	Yanked   json.RawMessage `json:"yanked"`
}

// notYanked returns a file entry that is not yanked.
func notYanked(filename string) pypiTestFile {
	return pypiTestFile{Filename: filename, Yanked: json.RawMessage("false")}
}

// yankedFile returns a file entry yanked with the given reason string.
// The reason is JSON-encoded as a string literal.
func yankedFile(filename, reason string) pypiTestFile {
	return pypiTestFile{
		Filename: filename,
		Yanked:   json.RawMessage(`"` + reason + `"`),
	}
}

func TestFetchPyPI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		files          []pypiTestFile
		ignoreVersions map[string]struct{}
		maxVersion     string
		wantVersion    string
		wantErr        string
	}{
		{
			name: "returns latest semver version",
			files: []pypiTestFile{
				notYanked("mypy-protobuf-3.5.0.tar.gz"),
				notYanked("mypy-protobuf-3.6.0.tar.gz"),
				notYanked("mypy_protobuf-5.0.0-py3-none-any.whl"),
				notYanked("mypy_protobuf-5.0.0.tar.gz"),
				// Go semver accepts "1.0" as v1.0.0 but 5.0.0 is still highest.
				notYanked("mypy-protobuf-1.0.tar.gz"),
				// Python-style pre-release: invalid Go semver, filtered out.
				notYanked("mypy_protobuf-2.0.0b7-py3-none-any.whl"),
			},
			wantVersion: "v5.0.0",
		},
		{
			name: "skips fully yanked releases",
			files: []pypiTestFile{
				notYanked("mypy-protobuf-3.6.0.tar.gz"),
				// Both files for 5.0.0 are yanked.
				yankedFile("mypy_protobuf-5.0.0-py3-none-any.whl", "bad release"),
				yankedFile("mypy_protobuf-5.0.0.tar.gz", "bad release"),
			},
			wantVersion: "v3.6.0",
		},
		{
			name: "version with at least one non-yanked file is available",
			files: []pypiTestFile{
				notYanked("mypy-protobuf-3.6.0.tar.gz"),
				// wheel yanked but sdist not: version is still available
				yankedFile("mypy_protobuf-5.0.0-py3-none-any.whl", "bad wheel"),
				notYanked("mypy_protobuf-5.0.0.tar.gz"),
			},
			wantVersion: "v5.0.0",
		},
		{
			name: "skips pre-release versions",
			files: []pypiTestFile{
				notYanked("mypy-protobuf-1.2.5.tar.gz"),
				notYanked("mypy_protobuf-2.0.0b7-py3-none-any.whl"),
			},
			wantVersion: "v1.2.5",
		},
		{
			name: "respects ignore_versions",
			files: []pypiTestFile{
				notYanked("mypy-protobuf-3.6.0.tar.gz"),
				notYanked("mypy_protobuf-5.0.0.tar.gz"),
			},
			ignoreVersions: map[string]struct{}{"v5.0.0": {}},
			wantVersion:    "v3.6.0",
		},
		{
			name: "respects max_version exclusive upper bound",
			files: []pypiTestFile{
				notYanked("mypy-protobuf-3.6.0.tar.gz"),
				notYanked("mypy_protobuf-5.0.0.tar.gz"),
			},
			maxVersion:  "v5.0.0",
			wantVersion: "v3.6.0",
		},
		{
			name: "error when no valid versions remain",
			files: []pypiTestFile{
				// Python-style pre-releases: invalid Go semver, all filtered out.
				notYanked("mypy_protobuf-2.0.0b7-py3-none-any.whl"),
				notYanked("mypy_protobuf-2.0.0rc1-py3-none-any.whl"),
			},
			wantErr: "no versions found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/vnd.pypi.simple.v1+json")
				if err := json.NewEncoder(w).Encode(struct {
					Files []pypiTestFile `json:"files"`
				}{Files: tt.files}); err != nil {
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

func TestPyPIVersionFromFilename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		filename string
		pkg      string
		want     string
	}{
		// sdist, hyphenated package name (older style)
		{"mypy-protobuf-3.6.0.tar.gz", "mypy-protobuf", "3.6.0"},
		// sdist, underscored package name (normalized)
		{"mypy_protobuf-5.0.0.tar.gz", "mypy-protobuf", "5.0.0"},
		// wheel
		{"mypy_protobuf-5.0.0-py3-none-any.whl", "mypy-protobuf", "5.0.0"},
		// pre-release (extraction still works; semver filter rejects it later)
		{"mypy_protobuf-2.0.0b7-py3-none-any.whl", "mypy-protobuf", "2.0.0b7"},
		// package with dots in name
		{"betterproto-1.2.5.tar.gz", "betterproto", "1.2.5"},
		// wrong package: no match
		{"other_pkg-1.0.0.tar.gz", "mypy-protobuf", ""},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			t.Parallel()
			got := pypiVersionFromFilename(tt.filename, tt.pkg)
			assert.Equal(t, tt.want, got)
		})
	}
}
