package resolver

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoteSourceLookup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/v1/plugins/blackstork/sqlite/versions", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		assert.Equal(t, "test/0.1", r.Header.Get("User-Agent"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"versions": [
				{
					"version": "1.0.0",
					"platforms": [
						{
							"os": "` + runtime.GOOS + `",
							"arch": "` + runtime.GOARCH + `"
						}
					]
				},
				{
					"version": "1.0.1",
					"platforms": [
						{
							"os": "` + runtime.GOOS + `",
							"arch": "` + runtime.GOARCH + `"
						}
					]
				}
			]
		}`))
	}))
	defer srv.Close()
	source := NewRemote(RemoteOptions{
		BaseURL:   srv.URL,
		UserAgent: "test/0.1",
	})
	versions, err := source.Lookup(context.Background(), Name{"blackstork", "sqlite"})
	assert.NoError(t, err)
	assert.Equal(t, []Version{
		mustVersion(t, "1.0.0"),
		mustVersion(t, "1.0.1"),
	}, versions)
}

func TestRemoteSourceLookupError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{
			"error": {
				"code": "not_found",
				"message": "plugin not found"
			}
		}`))
	}))
	defer srv.Close()
	source := NewRemote(RemoteOptions{
		BaseURL: srv.URL,
	})
	versions, err := source.Lookup(context.Background(), Name{"blackstork", "sqlite"})
	assert.EqualError(t, err, "plugin not found")
	assert.Nil(t, versions)
}

func mockTarGz(t *testing.T, files map[string]string) ([]byte, []Checksum) {
	t.Helper()
	checksums := []Checksum{}
	buf := bytes.NewBuffer(nil)
	gz := gzip.NewWriter(buf)
	w := tar.NewWriter(gz)
	for name, content := range files {
		hdr := &tar.Header{
			Name: name,
			Size: int64(len(content)),
		}
		if err := w.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
		h := sha256.New()
		h.Write([]byte(content))
		checksums = append(checksums, Checksum{
			OS:     runtime.GOOS,
			Arch:   runtime.GOARCH,
			Object: "binary",
			Sum:    h.Sum(nil),
		})
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	gz.Flush()
	gz.Close()
	h := sha256.New()
	if _, err := io.Copy(h, bytes.NewReader(buf.Bytes())); err != nil {
		t.Fatal(err)
	}
	checksums = append(checksums, Checksum{
		OS:     runtime.GOOS,
		Arch:   runtime.GOARCH,
		Object: "archive",
		Sum:    h.Sum(nil),
	})
	return buf.Bytes(), checksums
}

func TestRemoteSourceResolve(t *testing.T) {
	archive, checksums := mockTarGz(t, map[string]string{
		"sqlite@1.0.0": "plugin-binary",
	})
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/plugins/blackstork/sqlite/1.0.0/checksums":
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Accept"))
			assert.Equal(t, "test/0.1", r.Header.Get("User-Agent"))
			w.Header().Add("Content-Type", "application/json")
			checksumsJSON := []string{}
			for _, c := range checksums {
				raw, err := c.MarshalJSON()
				assert.NoError(t, err)
				checksumsJSON = append(checksumsJSON, string(raw))
			}
			w.Write([]byte(`{
			"checksums": [
				` + strings.Join(checksumsJSON, ",") + `
			]
		}`))
		case "/v1/plugins/blackstork/sqlite/1.0.0/download/" + runtime.GOOS + "/" + runtime.GOARCH:
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Accept"))
			assert.Equal(t, "test/0.1", r.Header.Get("User-Agent"))
			w.Header().Add("Content-Type", "application/json")
			w.Write([]byte(`{
				"os": "` + runtime.GOOS + `",
				"arch": "` + runtime.GOARCH + `",
				"download_url": "` + srv.URL + `/download"
			}`))
		case "/download":
			assert.Equal(t, "GET", r.Method)
			w.Header().Add("Content-Type", "octet/stream")
			w.Header().Add("Content-Disposition", "attachment; filename=plugin.tar.gz")
			w.Write(archive)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()
	source := NewRemote(RemoteOptions{
		BaseURL:     srv.URL,
		UserAgent:   "test/0.1",
		DownloadDir: t.TempDir(),
	})
	// without checksums input
	plugin, err := source.Resolve(context.Background(), Name{"blackstork", "sqlite"}, mustVersion(t, "1.0.0"), nil)
	require.NoError(t, err)
	assert.Equal(t, checksums, plugin.Checksums)
	assert.Equal(t, filepath.Join(source.downloadDir, "blackstork/sqlite@1.0.0"), plugin.BinaryPath)
	// pass with valid checksums input
	plugin, err = source.Resolve(context.Background(), Name{"blackstork", "sqlite"}, mustVersion(t, "1.0.0"), checksums)
	require.NoError(t, err)
	assert.Equal(t, checksums, plugin.Checksums)
	assert.Equal(t, filepath.Join(source.downloadDir, "blackstork/sqlite@1.0.0"), plugin.BinaryPath)
	// fail with invalid checksums input
	plugin, err = source.Resolve(context.Background(), Name{"blackstork", "sqlite"}, mustVersion(t, "1.0.0"), []Checksum{
		{
			OS:     runtime.GOOS,
			Arch:   runtime.GOARCH,
			Object: "archive",
			Sum:    []byte("other"),
		},
		{
			OS:     runtime.GOOS,
			Arch:   runtime.GOARCH,
			Object: "binary",
			Sum:    []byte("other"),
		},
	})
	require.Nil(t, plugin)
	require.Error(t, err, "failed to resolve plugin: checksum mismatch")
}
