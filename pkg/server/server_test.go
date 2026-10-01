package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"natasha-audrey/lastfm-collage-generator/pkg/config/timeframe"
)

func request(h http.Handler, target string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", target, nil))
	return w
}

func TestValidation(t *testing.T) {
	h := newHandler(func(context.Context, options) ([]byte, error) { t.Fatal("generated invalid request"); return nil, nil }, time.Second)
	for _, query := range []string{"", "user=", "user=%20", "user=a&size=2", "user=a&size=11", "user=a&size=x", "user=a&size=", "user=a&timeframe=", "user=a&timeframe=no", "user=a&user=b", "user=a&size=5&size=5", "user=a&path=foo", "user=%zz", "user=a;size=5"} {
		t.Run(query, func(t *testing.T) {
			w := request(h, "/generate?"+query)
			if w.Code != 400 {
				t.Fatalf("status %d: %s", w.Code, w.Body.String())
			}
			var body struct{ Error struct{ Code string } }
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body.Error.Code != "invalid_request" {
				t.Fatalf("invalid error: %s", w.Body.String())
			}
		})
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("POST", "/generate?user=a", nil))
	if w.Code != 405 || w.Header().Get("Allow") != "GET" {
		t.Fatalf("method response: %v", w)
	}
	if w := request(h, "/other"); w.Code != 404 {
		t.Fatalf("route status %d", w.Code)
	}
}

func TestOptionsAndPNG(t *testing.T) {
	for _, tc := range []struct {
		query string
		want  options
	}{
		{"user=listener", options{"listener", timeframe.Week, 5}},
		{"user=listener&timeframe=overall&size=10", options{"listener", timeframe.Overall, 10}},
	} {
		h := newHandler(func(_ context.Context, got options) ([]byte, error) {
			if got != tc.want {
				t.Errorf("options %+v, want %+v", got, tc.want)
			}
			var buf bytes.Buffer
			_ = png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 1, 1)))
			return buf.Bytes(), nil
		}, time.Second)
		w := request(h, "/generate?"+tc.query)
		if w.Code != 200 || w.Header().Get("Content-Type") != "image/png" || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("response: %v", w)
		}
		if _, err := png.Decode(w.Body); err != nil {
			t.Fatal(err)
		}
	}
}

func TestBusyAndDeadline(t *testing.T) {
	started, cancelled, release, exited := make(chan struct{}), make(chan struct{}), make(chan struct{}), make(chan struct{})
	defer close(release)
	h := newHandler(func(ctx context.Context, _ options) ([]byte, error) {
		close(started)
		<-ctx.Done()
		close(cancelled)
		<-release
		close(exited)
		return nil, ctx.Err()
	}, 100*time.Millisecond)
	finished := make(chan *httptest.ResponseRecorder, 1)
	go func() { finished <- request(h, "/generate?user=a") }()
	<-started
	if w := request(h, "/generate?user=b"); w.Code != 503 {
		t.Fatalf("busy status %d", w.Code)
	}
	select {
	case w := <-finished:
		if w.Code != 504 {
			t.Fatalf("timeout status %d", w.Code)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("deadline did not respond")
	}
	<-cancelled
	if w := request(h, "/generate?user=b"); w.Code != 503 {
		t.Fatalf("slot released before generation stopped: %d", w.Code)
	}
}

func TestDisconnectCancelsGeneration(t *testing.T) {
	started, stopped := make(chan struct{}), make(chan struct{})
	h := newHandler(func(ctx context.Context, _ options) ([]byte, error) {
		close(started)
		<-ctx.Done()
		close(stopped)
		return nil, ctx.Err()
	}, time.Minute)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	finished := make(chan struct{})
	go func() {
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/generate?user=a", nil).WithContext(ctx))
		close(finished)
	}()
	<-started
	cancel()
	for _, ch := range []chan struct{}{stopped, finished} {
		select {
		case <-ch:
		case <-time.After(time.Second):
			t.Fatal("cancellation not propagated")
		}
	}
}

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestUpstreamFailures(t *testing.T) {
	t.Setenv("API_KEY", "test-secret")
	for _, tc := range []struct {
		name, body   string
		status, want int
	}{
		{"no albums", `{"topalbums":{"album":[]}}`, 200, 422},
		{"unknown user", `{"error":6,"message":"User not found"}`, 200, 404},
		{"unknown user HTTP error", `{"error":6,"message":"User not found"}`, 400, 404},
		{"other invalid parameter", `{"error":6,"message":"Invalid parameters"}`, 200, 502},
		{"bad key", `{"error":10,"message":"secret details"}`, 200, 502},
		{"invalid json", `oops`, 200, 502},
		{"HTTP failure", `oops`, 503, 502},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(tc.body)), Header: make(http.Header)}, nil
			})}
			w := request(newHandler(generator(client), time.Second), "/generate?user=a")
			if w.Code != tc.want {
				t.Fatalf("status %d want %d: %s", w.Code, tc.want, w.Body.String())
			}
			if strings.Contains(w.Body.String(), "secret") {
				t.Fatal("upstream detail leaked")
			}
		})
	}
	h := newHandler(func(context.Context, options) ([]byte, error) { return nil, errors.New("/private/file: test-secret") }, time.Second)
	if w := request(h, "/generate?user=a"); w.Code != 500 || strings.Contains(w.Body.String(), "private") {
		t.Fatalf("internal error response %v", w)
	}
}

func TestGeneratorUsesCacheWithoutSavingCollage(t *testing.T) {
	t.Setenv("API_KEY", "test")
	t.Chdir(t.TempDir())
	if err := os.Mkdir("generated", 0755); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(filepath.Join("generated", "Artist_Album.png"))
	if err != nil {
		t.Fatal(err)
	}
	tile := image.NewRGBA(image.Rect(0, 0, 300, 300))
	tile.Set(0, 0, color.White)
	if err := png.Encode(file, tile); err != nil {
		t.Fatal(err)
	}
	_ = file.Close()
	client := &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Query().Get("method") != "user.gettopalbums" {
			t.Error("cached artwork downloaded")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"topalbums":{"album":[{"name":"Album","artist":{"name":"Artist"},"image":[{"#text":"https://example.test/art.png"}]}]}}`))}, nil
	})}
	w := request(newHandler(generator(client), time.Second), "/generate?user=a&size=3")
	if w.Code != 200 {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	img, err := png.Decode(w.Body)
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds() != image.Rect(0, 0, 900, 900) {
		t.Fatal(img.Bounds())
	}
	if r, _, _, _ := img.At(0, 0).RGBA(); r != 65535 {
		t.Fatal("cached tile missing")
	}
	if r, g, b, _ := img.At(500, 500).RGBA(); r != 0 || g != 0 || b != 0 {
		t.Fatal("unused tile not black")
	}
	entries, err := os.ReadDir(".")
	if err != nil || len(entries) != 1 || entries[0].Name() != "generated" {
		t.Fatalf("unexpected output files: %v %v", entries, err)
	}
}

func TestMissingKey(t *testing.T) {
	t.Setenv("API_KEY", "")
	if err := Run(context.Background(), "invalid"); err == nil || err.Error() != "API_KEY is required" {
		t.Fatalf("startup error: %v", err)
	}
}

func TestGeneratorCancelsUpstream(t *testing.T) {
	for _, artwork := range []bool{false, true} {
		t.Run(map[bool]string{false: "Last.fm", true: "artwork"}[artwork], func(t *testing.T) {
			t.Setenv("API_KEY", "test")
			t.Chdir(t.TempDir())
			if err := os.Mkdir("generated", 0755); err != nil {
				t.Fatal(err)
			}
			started, stopped := make(chan struct{}), make(chan struct{})
			client := &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
				if artwork && r.URL.Query().Get("method") == "user.gettopalbums" {
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"topalbums":{"album":[{"name":"Album","artist":{"name":"Artist"},"image":[{"#text":"https://example.test/art.png"}]}]}}`))}, nil
				}
				close(started)
				<-r.Context().Done()
				close(stopped)
				return nil, r.Context().Err()
			})}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			finished := make(chan struct{})
			generate := generator(client)
			go func() { defer close(finished); _, _ = generate(ctx, options{"a", timeframe.Week, 3}) }()
			<-started
			cancel()
			for _, ch := range []chan struct{}{stopped, finished} {
				select {
				case <-ch:
				case <-time.After(time.Second):
					t.Fatal("upstream was not cancelled")
				}
			}
		})
	}
}

func TestArtworkFailureUsesBlackTile(t *testing.T) {
	t.Setenv("API_KEY", "test")
	gradient, err := os.ReadFile(filepath.Join("..", "..", "static", "black-gradient.png"))
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir())
	if err := os.Mkdir("static", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join("static", "black-gradient.png"), gradient, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir("generated", 0755); err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Query().Get("method") == "user.gettopalbums" {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"topalbums":{"album":[{"name":"Album","artist":{"name":"Artist"},"image":[{"#text":"https://example.test/art.png"}]}]}}`))}, nil
		}
		return &http.Response{StatusCode: 503, Body: io.NopCloser(strings.NewReader("down"))}, nil
	})}
	w := request(newHandler(generator(client), 10*time.Second), "/generate?user=a")
	if w.Code != 200 {
		t.Fatalf("artwork failure: %d %s", w.Code, w.Body.String())
	}
	img, err := png.Decode(w.Body)
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds() != image.Rect(0, 0, 1500, 1500) {
		t.Fatal(img.Bounds())
	}
	if r, g, b, _ := img.At(299, 299).RGBA(); r != 0 || g != 0 || b != 0 {
		t.Fatal("fallback background is not black")
	}
	labeled := false
	for y := 0; y < 300; y++ {
		for x := 0; x < 300; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			if r != 0 || g != 0 || b != 0 {
				labeled = true
			}
		}
	}
	if !labeled {
		t.Fatal("fallback tile lost its labels")
	}
}

func TestRunStopsWhenCancelled(t *testing.T) {
	t.Setenv("API_KEY", "test")
	t.Chdir(t.TempDir())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	finished := make(chan error, 1)
	go func() { finished <- Run(ctx, "127.0.0.1:0") }()
	select {
	case err := <-finished:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("server did not stop")
	}
}
