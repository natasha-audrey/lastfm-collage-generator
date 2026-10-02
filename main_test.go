package main

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"natasha-audrey/lastfm-collage-generator/pkg/logging"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"natasha-audrey/lastfm-collage-generator/pkg/flags"
)

func cachedCLIAlbum(t *testing.T) {
	t.Helper()
	t.Chdir(t.TempDir())
	if err := os.Mkdir("generated", 0755); err != nil {
		t.Fatal(err)
	}
	var tile bytes.Buffer
	if err := png.Encode(&tile, image.NewRGBA(image.Rect(0, 0, 300, 300))); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("generated/Artist_Album.png", tile.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
}

const cachedAlbumResponse = `{"topalbums":{"album":[{"name":"Album","artist":{"name":"Artist"},"image":[{"#text":"https://example.org/art.png"}]}]}}`

func TestGenerateCollageContext_RedactsInvalidUpstreamURLs(t *testing.T) {
	t.Setenv("API_KEY", "test-api-key")
	for _, tc := range []struct {
		name, address, diagnostic string
	}{
		{"unsupported scheme", "ftp://service:private-password@example.org/?signature=private-signature", "unsupported protocol scheme"},
		{"invalid userinfo", "https://service:private password-suffix@example.org/", "invalid userinfo"},
		{"quoted userinfo", "https://service:private\"password-suffix@example.org/", "invalid userinfo"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("BASE_URL", tc.address)
			var stderr bytes.Buffer
			cmd := flags.NewCommandContext("v1", generateCollageContext)
			cmd.SetErr(&stderr)
			cmd.SetArgs([]string{"--log-level", "debug"})
			if err := cmd.Execute(); err == nil || !logging.IsReported(err) {
				t.Fatal("expected a reported upstream error")
			}
			output := stderr.String()
			for _, secret := range []string{"service", "private-password", "private-signature", "password-suffix", "test-api-key"} {
				if strings.Contains(output, secret) {
					t.Error("configured URL credential leaked")
				}
			}
			if !strings.Contains(output, tc.diagnostic) || !strings.Contains(output, "outcome=upstream_error") || strings.Count(output, "Generation completed") != 1 {
				t.Error("expected one upstream summary preserving the diagnostic")
			}
		})
	}
}

func TestGenerateCollageReturnsRenderingError(t *testing.T) {
	cachedCLIAlbum(t)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(cachedAlbumResponse))
	}))
	defer upstream.Close()
	t.Setenv("API_KEY", "test")
	t.Setenv("BASE_URL", upstream.URL)
	output := filepath.Join(t.TempDir(), "missing", "collage.png")
	err := generateCollage(&flags.Flags{User: "listener", Size: 3, Path: output})
	var pathErr *os.PathError
	if !errors.As(err, &pathErr) || pathErr.Path != output {
		t.Fatalf("expected collage save error, got %v", err)
	}
}

func TestGenerateCollageContext_EndToEndLogging(t *testing.T) {
	cachedCLIAlbum(t)
	t.Setenv("API_KEY", "cli-private-key")
	for _, tc := range []struct {
		name, body, outcome string
		fail                bool
	}{
		{"success", cachedAlbumResponse, "success", false},
		{"empty", `{"topalbums":{"album":[]}}`, "no_albums", true},
		{"unknown", `{"error":6,"message":"User not found"}`, "user_not_found", true},
		{"upstream", `{"error":10,"message":"bad api_key=cli-private-key"}`, "upstream_error", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(tc.body)) }))
			defer upstream.Close()
			t.Setenv("BASE_URL", upstream.URL)
			var stdout, stderr bytes.Buffer
			cmd := flags.NewCommandContext("v1", func(ctx context.Context, f *flags.Flags) error { return generateCollageContext(ctx, f) })
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)
			cmd.SetArgs([]string{"--log-level", "debug", "--path", filepath.Join(t.TempDir(), "collage.png")})
			err := cmd.Execute()
			if (err != nil) != tc.fail {
				t.Fatalf("error: %v", err)
			}
			if err != nil && !logging.IsReported(err) {
				t.Fatal("failure could be printed twice")
			}
			output := stderr.String()
			if strings.Count(output, "Generation completed") != 1 || !strings.Contains(output, "outcome="+tc.outcome) || strings.Contains(output, "cli-private-key") || stdout.Len() != 0 {
				t.Fatalf("stdout=%s stderr=%s", &stdout, output)
			}
			if !tc.fail {
				for _, stage := range []string{"lastfm_fetch", "album_parsing", "artwork_preparation", "composition", "png_writing"} {
					if !strings.Contains(output, "stage="+stage) {
						t.Errorf("missing %s: %s", stage, output)
					}
				}
				if !strings.Contains(output, "Artwork cache hit") {
					t.Fatal(output)
				}
			}
		})
	}
}
