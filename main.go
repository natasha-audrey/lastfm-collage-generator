// Package main.
//
// Command line entry point for last-fm-collage-generator.
package main

import (
	"context"
	"fmt"
	"natasha-audrey/lastfm-collage-generator/pkg/clients"
	"natasha-audrey/lastfm-collage-generator/pkg/flags"
	"natasha-audrey/lastfm-collage-generator/pkg/server"
	"natasha-audrey/lastfm-collage-generator/pkg/workers"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

// version is the CLI release version, maintained by release-please.
const version = "v0.8.0" // x-release-please-version

func generateCollage(f *flags.Flags) error {
	client := clients.NewLastFmClientFromHTTP(&http.Client{})
	res, err := client.GetTopAlbums(f.Time, f.User)
	if err != nil {
		return err
	}
	albums, err := workers.Albums{}.Parse(res)
	if err != nil {
		return err
	}
	_, err = (workers.Collage{}).MakeCollage(albums, f.Size, f.Path)
	return err
}

func main() {
	cmd := flags.NewCommand(version, func(options *flags.Flags) error {
		if err := os.MkdirAll(filepath.Join(".", "generated"), os.ModePerm); err != nil {
			return err
		}
		return generateCollage(options)
	})
	cmd.AddCommand(flags.NewServeCommand(func(address string) error {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		return server.Run(ctx, address)
	}))
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
