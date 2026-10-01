package main

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"natasha-audrey/lastfm-collage-generator/pkg/flags"
)

func TestGenerateCollageReturnsRenderingError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"topalbums":{"album":[]}}`))
	}))
	defer upstream.Close()
	t.Setenv("API_KEY", "test")
	t.Setenv("BASE_URL", upstream.URL)
	output := filepath.Join(t.TempDir(), "missing", "collage.png")
	if err := generateCollage(&flags.Flags{User: "listener", Size: 3, Path: output}); err == nil {
		t.Fatal("CLI discarded the collage save error")
	}
}
