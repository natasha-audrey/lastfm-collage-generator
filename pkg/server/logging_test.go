package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"natasha-audrey/lastfm-collage-generator/pkg/logging"
)

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

func TestRequestSummary(t *testing.T) {
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

func TestTimeoutBusyAndCleanupLogs(t *testing.T) {
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

func TestCancelledSummaryOmitsStatus(t *testing.T) {
	base, buf := loggedContext(t, "info")
	ctx, cancel := context.WithCancel(base)
	h := newHandler(func(ctx context.Context, _ options) ([]byte, error) { cancel(); <-ctx.Done(); return nil, ctx.Err() }, time.Second)
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/generate?user=a", nil).WithContext(ctx))
	output := buf.String()
	if !strings.Contains(output, "outcome=cancelled") || !strings.Contains(output, "level=INFO") || strings.Contains(output, "status=") || strings.Count(output, "Generation completed") != 1 {
		t.Fatal(output)
	}
}

func TestUpstreamLogRedactsCredentials(t *testing.T) {
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

func TestLifecycleLogsBoundPortAndFiltering(t *testing.T) {
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
