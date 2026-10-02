// Package logging provides credential-safe application logs and attempt correlation.
package logging

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Match quoted URLs first: Go errors quote invalid URLs containing spaces or quotes.
var urls = regexp.MustCompile(`(?i)"[a-z][a-z0-9+.-]*://(?:\\.|[^"\\])*"|[a-z][a-z0-9+.-]*://[^\s"<>]+`)
var credentials = regexp.MustCompile(`(?i)\b(api[_-]?key|shared[_-]?secret|password|access[_-]?token|token)\b["']?\s*[:=]\s*["']?[^\s&,;"']+`)
var bearer = regexp.MustCompile(`(?i)\b(Bearer|Basic)\s+[^\s"',;]+`)

func credentialKey(key string) bool {
	switch strings.ToLower(strings.NewReplacer("_", "", "-", "").Replace(key)) {
	case "apikey", "sharedsecret", "password", "accesstoken", "token", "authorization":
		return true
	}
	return false
}

// Sanitize removes configured secrets and credential-bearing URL components.
func Sanitize(value string) string {
	value = urls.ReplaceAllStringFunc(value, func(raw string) string {
		quoted := strings.HasPrefix(raw, `"`)
		if quoted {
			decoded, err := strconv.Unquote(raw)
			if err != nil {
				return `"[redacted URL]"`
			}
			raw = decoded
		}
		u, err := url.Parse(raw)
		if err != nil {
			if quoted {
				return `"[redacted URL]"`
			}
			return "[redacted URL]"
		}
		u.User = nil
		u.RawQuery = ""
		u.ForceQuery = false
		u.Fragment = ""
		if quoted {
			return strconv.Quote(u.String())
		}
		return u.String()
	})
	value = credentials.ReplaceAllString(value, "${1}=[redacted]")
	value = bearer.ReplaceAllString(value, "${1} [redacted]")
	for _, key := range []string{"API_KEY", "SHARED_SECRET"} {
		if secret := os.Getenv(key); secret != "" {
			for _, form := range []string{secret, url.QueryEscape(secret), url.PathEscape(secret)} {
				value = strings.ReplaceAll(value, form, "[redacted]")
			}
		}
	}
	return value
}

// New creates a text logger that filters levels and sanitizes every attribute.
func New(w io.Writer, level string) (*slog.Logger, error) {
	levels := map[string]slog.Level{"debug": slog.LevelDebug, "info": slog.LevelInfo, "warn": slog.LevelWarn, "error": slog.LevelError}
	threshold, ok := levels[level]
	if !ok {
		return nil, errors.New("log-level must be debug, info, warn, or error")
	}
	return slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: threshold, ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
		if credentialKey(a.Key) {
			return slog.String(a.Key, "[redacted]")
		}
		a.Key = Sanitize(a.Key)
		if a.Value.Kind() == slog.KindString {
			a.Value = slog.StringValue(Sanitize(a.Value.String()))
		}
		if a.Value.Kind() == slog.KindAny {
			a.Value = slog.StringValue(Sanitize(fmt.Sprint(a.Value.Any())))
		}
		return a
	}})), nil
}

type loggerKey struct{}

// WithLogger carries a logger through generation and its workers.
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

// FromContext returns the attempt logger, or a safe stderr logger outside an attempt.
func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok {
		return logger
	}
	logger, _ := New(os.Stderr, "info")
	return logger
}

// Start attaches a new attempt ID and returns its completion recorder.
func Start(ctx context.Context, source string) (context.Context, func(string, error, ...any)) {
	logger := FromContext(ctx).With("attempt_id", rand.Text(), "source", source)
	ctx = WithLogger(ctx, logger)
	started := time.Now()
	return ctx, func(outcome string, err error, attrs ...any) {
		level := slog.LevelInfo
		switch outcome {
		case "timeout", "upstream_error", "internal_error":
			level = slog.LevelError
		}
		args := []any{"outcome", outcome, "duration", time.Since(started)}
		if err != nil {
			args = append(args, "error", err)
		}
		args = append(args, attrs...)
		logger.Log(ctx, level, "Generation completed", args...)
	}
}

// Stage records debug progress and returns a stage completion recorder.
func Stage(ctx context.Context, name string) func() {
	logger := FromContext(ctx)
	started := time.Now()
	logger.DebugContext(ctx, "Stage started", "stage", name)
	return func() { logger.DebugContext(ctx, "Stage finished", "stage", name, "duration", time.Since(started)) }
}

type outcomeError struct {
	outcome string
	err     error
}

func (e *outcomeError) Error() string { return e.err.Error() }
func (e *outcomeError) Unwrap() error { return e.err }

// Failure classifies an error for a generation summary.
func Failure(outcome string, err error) error { return &outcomeError{outcome, err} }

// Outcome returns an error's classification, defaulting to an internal failure.
func Outcome(err error) string {
	if err == nil {
		return "success"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	if errors.Is(err, context.Canceled) {
		return "cancelled"
	}
	var classified *outcomeError
	if errors.As(err, &classified) {
		return classified.outcome
	}
	return "internal_error"
}

type reportedError struct{ error }

func (e reportedError) Unwrap() error { return e.error }

// Reported marks an error already included in a completion or lifecycle log.
func Reported(err error) error {
	if err == nil {
		return nil
	}
	return reportedError{err}
}

// IsReported reports whether an error has already been logged.
func IsReported(err error) bool { var reported reportedError; return errors.As(err, &reported) }
