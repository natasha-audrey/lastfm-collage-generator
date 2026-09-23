package flags

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"natasha-audrey/lastfm-collage-generator/pkg/config/timeframe"
)

func TestCommandOptions(t *testing.T) {
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

func TestCommandDefaults(t *testing.T) {
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

func TestCommandHelpAndVersion(t *testing.T) {
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

func TestCommandRejectsInvalidOptions(t *testing.T) {
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

func TestCommandPreservesExistingFile(t *testing.T) {
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
