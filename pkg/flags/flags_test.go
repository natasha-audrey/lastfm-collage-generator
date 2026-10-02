package flags

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"natasha-audrey/lastfm-collage-generator/pkg/config/timeframe"
	"natasha-audrey/lastfm-collage-generator/pkg/logging"
)

func TestCommand_Options(t *testing.T) {
	for _, long := range []bool{false, true} {
		name := "short"
		if long {
			name = "long"
		}
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "collage.png")
			args := []string{"-u", "someone", "-t", "overall", "-s", "10", "-p", path}
			if long {
				args = []string{"--user=someone", "--timeframe=overall", "--size=10", "--path=" + path}
			}
			var got *Flags
			cmd := NewCommand("v1.0.0", func(f *Flags) error { got = f; return nil })
			cmd.SetArgs(args)
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			want := Flags{Time: timeframe.Overall, Size: 10, Path: path, User: "someone"}
			if got == nil || *got != want {
				t.Fatalf("got %+v, want %+v", got, want)
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatalf("validation left a file: %v", err)
			}
		})
	}
}

func TestCommand_Defaults(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, args := range [][]string{{}, {"--user", ""}} {
		var got *Flags
		cmd := NewCommand("v1.0.0", func(f *Flags) error { got = f; return nil })
		cmd.SetArgs(args)
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		want := Flags{Time: timeframe.Week, Size: 5, Path: "./collage.png", User: "tashayasha"}
		if got == nil || *got != want {
			t.Fatalf("got %+v, want %+v", got, want)
		}
	}
}

func TestCommand_HelpAndVersion(t *testing.T) {
	for _, arg := range []string{"-v", "--version", "-h", "--help"} {
		t.Run(arg, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "collage.png")
			cmd := NewCommand("v1.0.0", func(*Flags) error { t.Fatal("collage callback called"); return nil })
			var output bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetArgs([]string{arg, "--timeframe", "invalid", "--size", "2", "--path", path})
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			if arg == "-v" || arg == "--version" {
				if output.String() != "v1.0.0\n" {
					t.Fatalf("unexpected version: %q", output.String())
				}
			} else {
				for _, flag := range []string{"--user", "--timeframe", "--size", "--path", "--version"} {
					if !strings.Contains(output.String(), flag) {
						t.Errorf("help missing %s", flag)
					}
				}
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatalf("help/version touched output: %v", err)
			}
		})
	}
}

func TestCommand_RejectsInvalidOptions(t *testing.T) {
	for _, args := range [][]string{
		{"--timeframe", "invalid"}, {"--size", "2"}, {"--size", "11"},
		{"--size", "abc"}, {"--path", filepath.Join(t.TempDir(), "missing", "collage.png")},
		{"--unknown"}, {"--user"}, {"unexpected"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			cmd := NewCommand("v1.0.0", func(*Flags) error { t.Fatal("collage callback called"); return nil })
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})
			cmd.SetArgs(args)
			if err := cmd.Execute(); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestCommand_PreservesExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "collage.png")
	const contents = "existing collage"
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := NewCommand("v1.0.0", func(*Flags) error { return nil })
	cmd.SetArgs([]string{"--path", path})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != contents {
		t.Fatalf("file changed: %q", got)
	}
}

func TestNewCommandContext_LoggingFlagAndSummary(t *testing.T) {
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

func TestNewCommandContext_InvalidLogLevelSkipsGeneration(t *testing.T) {
	cmd := NewCommand("v1", func(*Flags) error { t.Fatal("generation ran"); return nil })
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--log-level", "verbose"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "log-level") {
		t.Fatalf("error: %v", err)
	}
}
