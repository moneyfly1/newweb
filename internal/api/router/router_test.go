package router

import (
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	ginzip "github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func TestPrecompressedAssetsBypassGlobalGzip(t *testing.T) {
	gin.SetMode(gin.TestMode)

	root := t.TempDir()
	assetPath := filepath.Join(root, "app.js")
	original := []byte("console.log('asset payload');")
	if err := os.WriteFile(assetPath, original, 0o644); err != nil {
		t.Fatal(err)
	}

	var gzipBody bytes.Buffer
	gz := gzip.NewWriter(&gzipBody)
	if _, err := gz.Write(original); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(assetPath+".gz", gzipBody.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	brBody := []byte("precompressed-brotli-placeholder")
	if err := os.WriteFile(assetPath+".br", brBody, 0o644); err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	r.Use(ginzip.Gzip(
		ginzip.DefaultCompression,
		ginzip.WithExcludedPathsRegexs([]string{`^/assets/.*`}),
	))
	r.GET("/assets/*filepath", servePrecompressedAsset(root))

	t.Run("brotli", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
		req.Header.Set("Accept-Encoding", "br, gzip")

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
		}
		if got := w.Header().Get("Content-Encoding"); got != "br" {
			t.Fatalf("Content-Encoding = %q, want br", got)
		}
		if !bytes.Equal(w.Body.Bytes(), brBody) {
			t.Fatalf("served body did not match .br file")
		}
		if got := w.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
			t.Fatalf("Cache-Control = %q", got)
		}
	})

	t.Run("gzip", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
		req.Header.Set("Accept-Encoding", "gzip")

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
		}
		if got := w.Header().Get("Content-Encoding"); got != "gzip" {
			t.Fatalf("Content-Encoding = %q, want gzip", got)
		}
		if !bytes.Equal(w.Body.Bytes(), gzipBody.Bytes()) {
			t.Fatalf("served body did not match .gz file")
		}
	})
}
