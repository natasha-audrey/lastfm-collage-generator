package flags

import (
	"flag"
	"io"
	"os"
	"path/filepath"
	"testing"

	"natasha-audrey/lastfm-collage-generator/pkg/config/timeframe"
)

func TestParse_UsesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "collage.png")
	withCommandLine(t, "-p", path)

	got, err := Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	want := &Flags{Time: timeframe.Week, Size: 5, Path: path, User: "tashayasha"}
	if *got != *want {
		t.Errorf("Parse() = %+v, want %+v", *got, *want)
	}
}

func TestParse_ParsesProvidedValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "collage.png")
	if err := os.Mkdir(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("Mkdir(%q): %v", filepath.Dir(path), err)
	}
	withCommandLine(t, "-t", "overall", "-s", "10", "-p", path, "-u", "someone")

	got, err := Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	want := &Flags{Time: timeframe.Overall, Size: 10, Path: path, User: "someone"}
	if *got != *want {
		t.Errorf("Parse() = %+v, want %+v", *got, *want)
	}
}

func TestParse_AcceptsExistingPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "existing-collage.png")
	const originalContents = "existing collage"
	if err := os.WriteFile(path, []byte(originalContents), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
	withCommandLine(t, "-p", path)

	got, err := Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got.Path != path {
		t.Errorf("Parse().Path = %q, want %q", got.Path, path)
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}
	if string(contents) != originalContents {
		t.Errorf("existing file contents = %q, want %q", contents, originalContents)
	}
}

func TestParse_ParsesDefaultUser(t *testing.T) {
	path := filepath.Join(t.TempDir(), "collage.png")
	withCommandLine(t, "-p", path, "-u", "")

	got, err := Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	want := &Flags{Time: timeframe.Week, Size: 5, Path: path, User: "tashayasha"}
	if *got != *want {
		t.Errorf("Parse() = %+v, want %+v", *got, *want)
	}
}

func TestParse_ReturnsErrorForInvalidTime(t *testing.T) {
	path := filepath.Join(t.TempDir(), "collage.png")
	withCommandLine(t, "-t", "not-a-timeframe", "-p", path)

	got, err := Parse()
	if err == nil {
		t.Fatal("Parse() error = nil, want error for invalid time frame")
	}
	if got.Time != 0 {
		t.Errorf("Parse().Time = %v, want zero TimeFrame", got.Time)
	}
}

func TestParse_ReturnsErrorForInvalidSize(t *testing.T) {
	path := filepath.Join(t.TempDir(), "collage.png")
	withCommandLine(t, "-s", "2", "-p", path)

	got, err := Parse()
	if err == nil {
		t.Fatal("Parse() error = nil, want error for invalid size")
	}
	if got.Size != 2 {
		t.Errorf("Parse().Size = %d, want 2", got.Size)
	}
}

func TestParse_ReturnsErrorForUnwritablePath(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(parent, nil, 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", parent, err)
	}
	withCommandLine(t, "-p", filepath.Join(parent, "collage.png"))

	got, err := Parse()
	if err == nil {
		t.Fatal("Parse() error = nil, want error for unwritable path")
	}
	if got.Path != "" {
		t.Errorf("Parse().Path = %q, want empty string", got.Path)
	}
}

func withCommandLine(t *testing.T, args ...string) {
	t.Helper()

	originalCommandLine := flag.CommandLine
	originalArgs := os.Args
	commandLine := flag.NewFlagSet("test", flag.ContinueOnError)
	commandLine.SetOutput(io.Discard)
	flag.CommandLine = commandLine
	os.Args = append([]string{"test"}, args...)

	t.Cleanup(func() {
		flag.CommandLine = originalCommandLine
		os.Args = originalArgs
	})
}
