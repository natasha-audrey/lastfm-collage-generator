package logging

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/url"
	"strings"
	"testing"
)

func TestNew_CredentialSanitization(t *testing.T) {
	t.Setenv("API_KEY", "key +/&value")
	t.Setenv("SHARED_SECRET", "shared-secret-value")
	var buf bytes.Buffer
	logger, _ := New(&buf, "debug")
	logger = logger.With("api_key", "unknown-structured-key")
	logger.Debug("Failure key +/&value", "error", errors.New(`Get "https://login:password-value@example.org/art?signature=unknown-query-secret#fragment-secret": network failure`), "detail", `{"password":"json-password"} Authorization: Bearer bearer-value`, "encoded", url.QueryEscape("key +/&value"), "shared", "shared-secret-value", "nested", slog.GroupValue(slog.String("token", "group-secret")), "artist", "Some Artist", "path", "/local/art.png")
	output := buf.String()
	for _, secret := range []string{"key +/&value", "shared-secret-value", "unknown-structured-key", "password-value", "unknown-query-secret", "fragment-secret", "json-password", "bearer-value", "group-secret", url.QueryEscape("key +/&value")} {
		if strings.Contains(output, secret) {
			t.Errorf("credential %q leaked: %s", secret, output)
		}
	}
	for _, detail := range []string{"https://example.org/art", "network failure", "Some Artist", "/local/art.png"} {
		if !strings.Contains(output, detail) {
			t.Errorf("missing useful detail %q: %s", detail, output)
		}
	}
	if strings.Contains(output, "!BADKEY") {
		t.Fatal(output)
	}
}

func TestStart_LevelsAndCorrelation(t *testing.T) {
	for _, level := range []string{"debug", "info", "warn", "error"} {
		t.Run(level, func(t *testing.T) {
			var buf bytes.Buffer
			logger, _ := New(&buf, level)
			ctx, finish := Start(WithLogger(context.Background(), logger), "cli")
			done := Stage(ctx, "artwork_preparation")
			done()
			FromContext(ctx).WarnContext(ctx, "Artwork fallback", "album", "Album")
			finish("success", nil)
			output := buf.String()
			if strings.Contains(output, "level=DEBUG") != (level == "debug") {
				t.Fatal(output)
			}
			if strings.Contains(output, "Generation completed") != (level == "debug" || level == "info") {
				t.Fatal(output)
			}
			if strings.Contains(output, "Artwork fallback") != (level != "error") {
				t.Fatal(output)
			}
			var id string
			for _, field := range strings.Fields(output) {
				if strings.HasPrefix(field, "attempt_id=") {
					if id == "" {
						id = field
					}
					if id != field {
						t.Fatalf("attempt IDs differ: %s", output)
					}
				}
			}
			if level != "error" && id == "" {
				t.Fatal("missing attempt ID")
			}
			buf.Reset()
			finish("upstream_error", errors.New("network failure"))
			if !strings.Contains(buf.String(), "level=ERROR") {
				t.Fatal(buf.String())
			}
		})
	}
}

func TestNew_InvalidLevel(t *testing.T) {
	for _, level := range []string{"", "verbose", "INFO"} {
		if _, err := New(&bytes.Buffer{}, level); err == nil {
			t.Fatalf("accepted %q", level)
		}
	}
}
