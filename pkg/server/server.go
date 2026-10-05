// Package server exposes local collage generation over HTTP.
package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"natasha-audrey/lastfm-collage-generator/pkg/clients"
	"natasha-audrey/lastfm-collage-generator/pkg/config/timeframe"
	"natasha-audrey/lastfm-collage-generator/pkg/logging"
	"natasha-audrey/lastfm-collage-generator/pkg/workers"
)

type options struct {
	user   string
	period timeframe.TimeFrame
	size   int
}
type apiError struct {
	status        int
	code, message string
}

func (e *apiError) Error() string { return e.message }

type generateFunc func(context.Context, options) ([]byte, error)

type handler struct {
	generate generateFunc
	busy     chan struct{}
	timeout  time.Duration
}

func newHandler(generate generateFunc, timeout time.Duration) http.Handler {
	return &handler{generate: generate, busy: make(chan struct{}, 1), timeout: timeout}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{"code": code, "message": message},
	})
}

func parseOptions(raw string) (options, error) {
	result := options{period: timeframe.Week, size: 5}
	q, err := url.ParseQuery(raw)
	if err != nil {
		return result, errors.New("invalid query string")
	}
	for key, values := range q {
		if key != "user" && key != "timeframe" && key != "size" {
			return result, fmt.Errorf("unknown parameter %q", key)
		}
		if len(values) != 1 {
			return result, fmt.Errorf("duplicate parameter %q", key)
		}
	}
	result.user = strings.TrimSpace(q.Get("user"))
	if result.user == "" {
		return result, errors.New("user is required")
	}
	if q.Has("timeframe") {
		result.period, err = timeframe.ParseString(q.Get("timeframe"))
		if err != nil {
			return result, errors.New("invalid timeframe")
		}
	}
	if q.Has("size") {
		result.size, err = strconv.Atoi(q.Get("size"))
		if err != nil || result.size < 3 || result.size > 10 {
			return result, errors.New("size must be an integer between 3 and 10")
		}
	}
	return result, nil
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, finish := logging.Start(r.Context(), "server")
	r = r.WithContext(ctx)
	outcome, status := "cancelled", 0
	var cause error
	var opts options
	defer func() {
		attrs := []any{"method", r.Method}
		if status != 0 {
			attrs = append(attrs, "status", status)
		}
		if opts.user != "" {
			attrs = append(attrs, "user", opts.user, "listening_period", opts.period.String(), "grid_size", opts.size)
		}
		finish(outcome, cause, attrs...)
	}()
	fail := func(code int, name, message string) {
		outcome, status = name, code
		writeError(w, code, name, message)
	}
	w.Header().Set("Cache-Control", "no-store")
	if r.URL.Path != "/v1/generate" {
		fail(404, "not_found", "endpoint not found")
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		fail(405, "method_not_allowed", "use GET")
		return
	}
	var err error
	opts, err = parseOptions(r.URL.RawQuery)
	if err != nil {
		cause = err
		fail(400, "invalid_request", err.Error())
		return
	}
	select {
	case h.busy <- struct{}{}:
	default:
		fail(503, "busy", "another generation is running")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()
	type result struct {
		data []byte
		err  error
	}
	done := make(chan result, 1)
	go func() {
		// Keep the slot until generation exits, even if the HTTP deadline expires.
		defer func() {
			<-h.busy
			logging.FromContext(ctx).DebugContext(ctx, "Generation worker stopped")
		}()
		defer func() {
			if recovered := recover(); recovered != nil {
				done <- result{err: fmt.Errorf("generation panic: %v", recovered)}
			}
		}()
		data, err := h.generate(ctx, opts)
		done <- result{data, err}
	}()
	var res result
	select {
	case <-ctx.Done():
		cause = ctx.Err()
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			fail(504, "timeout", "generation exceeded 60 seconds")
		}
		return
	case res = <-done:
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		cause = ctx.Err()
		fail(504, "timeout", "generation exceeded 60 seconds")
		return
	}
	if ctx.Err() != nil {
		cause = ctx.Err()
		return
	}
	if res.err != nil {
		cause = res.err
		var failure *apiError
		if errors.As(res.err, &failure) {
			fail(failure.status, failure.code, failure.message)
		} else {
			fail(500, "internal_error", "collage generation failed")
		}
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.WriteHeader(http.StatusOK)
	status, outcome = http.StatusOK, "success"
	if _, err := w.Write(res.data); err != nil {
		cause = err
		outcome = "internal_error"
		if r.Context().Err() != nil {
			outcome = "cancelled"
		}
	}
}

func generator(client *http.Client) generateFunc {
	lastfm := clients.NewLastFmClientFromHTTP(client)
	return func(ctx context.Context, opts options) ([]byte, error) {
		upstream := &apiError{502, "upstream_error", "could not retrieve Last.fm data"}
		done := logging.Stage(ctx, "lastfm_fetch")
		response, err := lastfm.GetTopAlbumsContext(ctx, opts.period, opts.user)
		done()
		if err != nil {
			return nil, errors.Join(upstream, err)
		}
		if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusBadRequest && response.StatusCode != http.StatusNotFound {
			response.Body.Close()
			return nil, errors.Join(upstream, fmt.Errorf("Last.fm HTTP status %d", response.StatusCode))
		}
		done = logging.Stage(ctx, "album_parsing")
		albums, err := (workers.Albums{}).Parse(response)
		done()
		if err != nil {
			var api *workers.APIError
			if errors.As(err, &api) && api.Code == 6 && strings.EqualFold(strings.TrimSpace(api.Message), "User not found") {
				return nil, &apiError{404, "user_not_found", "Last.fm user not found"}
			}
			return nil, errors.Join(upstream, err)
		}
		if response.StatusCode != http.StatusOK {
			return nil, errors.Join(upstream, fmt.Errorf("Last.fm HTTP status %d", response.StatusCode))
		}
		if len(albums) == 0 {
			return nil, &apiError{422, "no_albums", "no albums for this listening period"}
		}
		img, err := workers.Render(ctx, client, albums, opts.size)
		if err != nil {
			return nil, err
		}
		var buf bytes.Buffer
		done = logging.Stage(ctx, "png_writing")
		err = png.Encode(&buf, img)
		done()
		if err != nil {
			return nil, err
		}
		return buf.Bytes(), ctx.Err()
	}
}

// Run serves until ctx is cancelled, immediately closing active connections.
// API_KEY must be configured; artwork shares the CLI's ./generated cache.
func Run(ctx context.Context, address string) (err error) {
	logger := logging.FromContext(ctx)
	defer func() {
		if err != nil {
			logger.ErrorContext(ctx, "Server failed", "error", err)
			err = logging.Reported(err)
		}
	}()
	if strings.TrimSpace(os.Getenv("API_KEY")) == "" {
		return errors.New("API_KEY is required")
	}
	if err := os.MkdirAll("generated", 0755); err != nil {
		return err
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	defer listener.Close()
	logger.InfoContext(ctx, "Server listening", "url", "http://"+listener.Addr().String())
	defer logger.InfoContext(ctx, "Server stopped")
	srv := &http.Server{
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
		Handler:           newHandler(generator(&http.Client{Timeout: 60 * time.Second}), 60*time.Second),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
		WriteTimeout:      65 * time.Second,
		BaseContext:       func(net.Listener) context.Context { return ctx },
	}
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			_ = srv.Close()
		case <-done:
		}
	}()
	err = srv.Serve(listener)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
