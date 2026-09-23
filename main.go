// Package main.
//
// Command line entry point for last-fm-collage-generator.
package main

import (
	"fmt"
	"natasha-audrey/lastfm-collage-generator/pkg/clients"
	"natasha-audrey/lastfm-collage-generator/pkg/flags"
	"natasha-audrey/lastfm-collage-generator/pkg/workers"
	"net/http"
	"os"
	"path/filepath"
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
	workers.Collage{}.MakeCollage(albums, f.Size, f.Path)
	return nil
}

func main() {
	cmd := flags.NewCommand(version, func(options *flags.Flags) error {
		if err := os.MkdirAll(filepath.Join(".", "generated"), os.ModePerm); err != nil {
			return err
		}
		return generateCollage(options)
	})
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
