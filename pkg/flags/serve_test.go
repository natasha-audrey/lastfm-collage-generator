package flags

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"natasha-audrey/lastfm-collage-generator/pkg/logging"
)

func TestServeCommand(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"serve"}, "127.0.0.1:8080"},
		{[]string{"serve", "--listen", "127.0.0.1:9000"}, "127.0.0.1:9000"},
		{[]string{"serve", "--help"}, ""},
	} {
		called := ""
		root := NewCommand("test", func(*Flags) error { t.Fatal("CLI generation invoked"); return nil })
		root.AddCommand(NewServeCommand(func(address string) error { called = address; return nil }))
		root.SetOut(&bytes.Buffer{})
		root.SetArgs(tc.args)
		if err := root.Execute(); err != nil {
			t.Fatal(err)
		}
		if called != tc.want {
			t.Fatalf("address %q want %q", called, tc.want)
		}
	}
}

func TestNewServeCommandContext_InheritsLogLevel(t *testing.T) {
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
