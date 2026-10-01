package flags

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"natasha-audrey/lastfm-collage-generator/pkg/logging"
)

func TestLoggingFlagAndSummary(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, tc := range []struct {
		name           string
		args           []string
		fail           bool
		outcome        string
		summary, debug bool
	}{
		{"default", nil, false, "success", true, false},
		{"debug", []string{"--log-level", "debug"}, false, "success", true, true},
		{"quiet", []string{"--log-level", "error"}, false, "success", false, false},
		{"failure", nil, true, "internal_error", true, false},
		{"validation", []string{"--size", "2"}, false, "invalid_request", true, false},
		{"flag parsing", []string{"--size", "abc"}, false, "invalid_request", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var stderr, stdout bytes.Buffer
			cmd := NewCommandContext("v1", func(ctx context.Context, _ *Flags) error {
				logging.FromContext(ctx).DebugContext(ctx, "Progress")
				if tc.fail {
					return errors.New("failed")
				}
				return nil
			})
			cmd.SetErr(&stderr)
			cmd.SetOut(&stdout)
			cmd.SetArgs(tc.args)
			err := cmd.Execute()
			if (err != nil) != (tc.fail || tc.outcome == "invalid_request") {
				t.Fatalf("error: %v", err)
			}
			if stdout.Len() != 0 {
				t.Fatalf("logging polluted stdout: %s", &stdout)
			}
			wantCount := 0
			if tc.summary {
				wantCount = 1
			}
			if strings.Count(stderr.String(), "Generation completed") != wantCount {
				t.Fatal(stderr.String())
			}
			if tc.summary && !strings.Contains(stderr.String(), "outcome="+tc.outcome) {
				t.Fatal(stderr.String())
			}
			if strings.Contains(stderr.String(), "Progress") != tc.debug {
				t.Fatal(stderr.String())
			}
			if err != nil && !logging.IsReported(err) {
				t.Fatal("error could be printed twice")
			}
		})
	}
}

func TestServeInheritsLogLevel(t *testing.T) {
	var output bytes.Buffer
	cmd := NewCommand("v1", func(*Flags) error { t.Fatal("root called"); return nil })
	cmd.AddCommand(NewServeCommandContext(func(ctx context.Context, _ string) error {
		logging.FromContext(ctx).DebugContext(ctx, "Server progress")
		return nil
	}))
	cmd.SetErr(&output)
	cmd.SetArgs([]string{"serve", "--log-level", "debug"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Server progress") {
		t.Fatal(output.String())
	}
}

func TestInvalidLogLevelSkipsGeneration(t *testing.T) {
	cmd := NewCommand("v1", func(*Flags) error { t.Fatal("generation ran"); return nil })
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--log-level", "verbose"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "log-level") {
		t.Fatalf("error: %v", err)
	}
}
