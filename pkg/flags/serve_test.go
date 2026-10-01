package flags

import (
	"bytes"
	"testing"
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
