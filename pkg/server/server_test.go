package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"natasha-audrey/lastfm-collage-generator/pkg/config/timeframe"
	"natasha-audrey/lastfm-collage-generator/pkg/logging"
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

type logBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *logBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}
func (b *logBuffer) String() string { b.mu.Lock(); defer b.mu.Unlock(); return b.buf.String() }

func loggedContext(t *testing.T, level string) (context.Context, *logBuffer) {
	t.Helper()
	buf := &logBuffer{}
	logger, err := logging.New(buf, level)
	if err != nil {
		t.Fatal(err)
	}
	return logging.WithLogger(context.Background(), logger), buf
}

func TestServeHTTP_RequestSummary(t *testing.T) {
	for _, tc := range []struct {
		name, target, outcome, level string
		status                       int
		err                          error
		panicValue                   bool
	}{
		{"success", "/generate?user=listener", "success", "INFO", 200, nil, false},
		{"validation", "/generate?size=2", "invalid_request", "INFO", 400, nil, false},
		{"unknown user", "/generate?user=a", "user_not_found", "INFO", 404, &apiError{404, "user_not_found", "unknown user"}, false},
		{"empty", "/generate?user=a", "no_albums", "INFO", 422, &apiError{422, "no_albums", "no albums"}, false},
		{"upstream", "/generate?user=a", "upstream_error", "ERROR", 502, errors.Join(&apiError{502, "upstream_error", "upstream failed"}, errors.New("network unavailable")), false},
		{"internal", "/generate?user=a", "internal_error", "ERROR", 500, errors.New("disk failure"), false},
		{"panic", "/generate?user=a", "internal_error", "ERROR", 500, nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, buf := loggedContext(t, "info")
			h := newHandler(func(ctx context.Context, _ options) ([]byte, error) {
				if tc.panicValue {
					panic("renderer failure")
				}
				return []byte("png"), tc.err
			}, time.Second)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest("GET", tc.target, nil).WithContext(ctx))
			output := buf.String()
			for _, field := range []string{"attempt_id=", "duration=", "outcome=" + tc.outcome, "level=" + tc.level, fmt.Sprintf("status=%d", tc.status)} {
				if !strings.Contains(output, field) {
					t.Errorf("missing %s: %s", field, output)
				}
			}
			if w.Code != tc.status || strings.Count(output, "Generation completed") != 1 || strings.Contains(output, "!BADKEY") {
				t.Fatal(output)
			}
			if tc.name == "panic" && !strings.Contains(output, "renderer failure") {
				t.Fatal(output)
			}
		})
	}
}

func TestServeHTTP_TimeoutBusyAndCleanupLogs(t *testing.T) {
	ctx, buf := loggedContext(t, "debug")
	started, release := make(chan struct{}), make(chan struct{})
	h := newHandler(func(ctx context.Context, _ options) ([]byte, error) {
		close(started)
		<-release
		return nil, ctx.Err()
	}, 50*time.Millisecond)
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/generate?user=a", nil).WithContext(ctx))
	}()
	<-started
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/generate?user=b", nil).WithContext(ctx))
	<-finished
	output := buf.String()
	for _, field := range []string{"outcome=timeout", "status=504", "outcome=busy", "status=503"} {
		if !strings.Contains(output, field) {
			t.Errorf("missing %s: %s", field, output)
		}
	}
	close(release)
	deadline := time.Now().Add(time.Second)
	for !strings.Contains(buf.String(), "Generation worker stopped") && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	output = buf.String()
	if !strings.Contains(output, "Generation worker stopped") || strings.Count(output, "Generation completed") != 2 {
		t.Fatal(output)
	}
	var timeoutID, cleanupID, busyID string
	idPattern := regexp.MustCompile(`attempt_id=([^ ]+)`)
	for _, line := range strings.Split(output, "\n") {
		match := idPattern.FindStringSubmatch(line)
		if len(match) != 2 {
			continue
		}
		switch {
		case strings.Contains(line, "outcome=timeout"):
			timeoutID = match[1]
		case strings.Contains(line, "outcome=busy"):
			busyID = match[1]
		case strings.Contains(line, "Generation worker stopped"):
			cleanupID = match[1]
		}
	}
	if timeoutID == "" || timeoutID != cleanupID || busyID == timeoutID {
		t.Fatalf("wrong attempt correlation: %s", output)
	}
}

func TestServeHTTP_CancelledSummaryOmitsStatus(t *testing.T) {
	base, buf := loggedContext(t, "info")
	ctx, cancel := context.WithCancel(base)
	h := newHandler(func(ctx context.Context, _ options) ([]byte, error) { cancel(); <-ctx.Done(); return nil, ctx.Err() }, time.Second)
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/generate?user=a", nil).WithContext(ctx))
	output := buf.String()
	if !strings.Contains(output, "outcome=cancelled") || !strings.Contains(output, "level=INFO") || strings.Contains(output, "status=") || strings.Count(output, "Generation completed") != 1 {
		t.Fatal(output)
	}
}

func TestServeHTTP_UpstreamLogRedactsCredentials(t *testing.T) {
	t.Setenv("API_KEY", "private-api-key")
	t.Setenv("SHARED_SECRET", "private-shared-secret")
	t.Setenv("BASE_URL", "https://upstream-user:upstream-password@example.org/?token=private-token")
	ctx, buf := loggedContext(t, "debug")
	client := &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("request %s failed: private-shared-secret", r.URL)
	})}
	h := newHandler(generator(client), time.Second)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/generate?user=listener", nil).WithContext(ctx))
	output := buf.String()
	for _, secret := range []string{"private-api-key", "private-shared-secret", "upstream-password", "private-token", "upstream-user"} {
		if strings.Contains(output, secret) {
			t.Errorf("credential leaked: %s", output)
		}
	}
	if !strings.Contains(output, "example.org") || !strings.Contains(output, "outcome=upstream_error") || w.Code != 502 {
		t.Fatal(output)
	}
}

func TestRun_LifecycleLogsBoundPortAndFiltering(t *testing.T) {
	t.Setenv("API_KEY", "lifecycle-key")
	t.Chdir(t.TempDir())
	for _, level := range []string{"info", "error"} {
		ctx, buf := loggedContext(t, level)
		ctx, cancel := context.WithCancel(ctx)
		cancel()
		if err := Run(ctx, "127.0.0.1:0"); err != nil {
			t.Fatal(err)
		}
		output := buf.String()
		if level == "error" {
			if output != "" {
				t.Fatal(output)
			}
			continue
		}
		match := regexp.MustCompile(`url=http://127\.0\.0\.1:([0-9]+)`).FindStringSubmatch(output)
		if len(match) != 2 || match[1] == "0" || !strings.Contains(output, "Server stopped") {
			t.Fatal(output)
		}
	}
}
