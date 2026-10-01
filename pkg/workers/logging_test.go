package workers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"natasha-audrey/lastfm-collage-generator/pkg/logging"
	"natasha-audrey/lastfm-collage-generator/pkg/model"
)

type loggingTransport func(*http.Request) (*http.Response, error)

func (f loggingTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestArtworkWarningsAndSuccessfulSummary(t *testing.T) {
	gradient, err := os.ReadFile("../../static/black-gradient.png")
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir())
	if err := os.Mkdir("static", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("static/black-gradient.png", gradient, 0600); err != nil {
		t.Fatal(err)
	}
	for _, reason := range []string{"missing_url", "download_failed", "decode_failed"} {
		t.Run(reason, func(t *testing.T) {
			var buf bytes.Buffer
			logger, _ := logging.New(&buf, "info")
			ctx, finish := logging.Start(logging.WithLogger(context.Background(), logger), "cli")
			var albums []model.Album
			for i := 0; i < 10; i++ {
				album := model.Album{Name: fmt.Sprintf("Album %d", i), Artist: "Artist", LocalImage: filepath.Join(t.TempDir(), "tile")}
				if reason != "missing_url" {
					album.Image = "https://login:artwork-password@example.org/art?signature=artwork-token"
				}
				albums = append(albums, album)
			}
			client := &http.Client{Transport: loggingTransport(func(r *http.Request) (*http.Response, error) {
				if reason == "download_failed" {
					return nil, fmt.Errorf("artwork request %s failed", r.URL)
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("not an image"))}, nil
			})}
			_, err := Render(ctx, client, albums, 4)
			finish(logging.Outcome(err), err)
			if err != nil {
				t.Fatal(err)
			}
			output := buf.String()
			if strings.Count(output, "level=WARN") != 10 || strings.Count(output, "reason="+reason) != 10 || strings.Count(output, "Generation completed") != 1 || !strings.Contains(output, "outcome=success") {
				t.Fatal(output)
			}
			if strings.Contains(output, "!BADKEY") || strings.Contains(output, "artwork-password") || strings.Contains(output, "artwork-token") {
				t.Fatal(output)
			}
			ids := regexp.MustCompile(`attempt_id=([^ ]+)`).FindAllStringSubmatch(output, -1)
			if len(ids) != 11 {
				t.Fatal(output)
			}
			for _, id := range ids {
				if id[1] != ids[0][1] {
					t.Fatal(output)
				}
			}
		})
	}
}
