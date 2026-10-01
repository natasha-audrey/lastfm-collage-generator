package flags

import (
	"context"
	"github.com/spf13/cobra"
)

// NewServeCommand builds the local HTTP server command.
func NewServeCommand(run func(string) error) *cobra.Command {
	return NewServeCommandContext(func(_ context.Context, address string) error { return run(address) })
}

// NewServeCommandContext carries logging configuration into the server.
func NewServeCommandContext(run func(context.Context, string) error) *cobra.Command {
	var address string
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve the collage generation API locally",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			return run(cmd.Context(), address)
		},
	}
	cmd.Flags().StringVar(&address, "listen", "127.0.0.1:8080", "HTTP listen address")
	return cmd
}
