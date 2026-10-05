// Package main.
//
// Command line entry point for last-fm-collage-generator.
package main

import (
	"context"
	"errors"
	"fmt"
	"natasha-audrey/lastfm-collage-generator/pkg/clients"
	"natasha-audrey/lastfm-collage-generator/pkg/flags"
	"natasha-audrey/lastfm-collage-generator/pkg/logging"
	"natasha-audrey/lastfm-collage-generator/pkg/server"
	"natasha-audrey/lastfm-collage-generator/pkg/workers"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
)

// version is the CLI release version, maintained by release-please.
const version = "v1.1.0" // x-release-please-version

func generateCollage(f *flags.Flags) error {
	return generateCollageContext(context.Background(), f)
}

func generateCollageContext(ctx context.Context, f *flags.Flags) error {
	client := clients.NewLastFmClientFromHTTP(&http.Client{})
	done := logging.Stage(ctx, "lastfm_fetch")
	res, err := client.GetTopAlbumsContext(ctx, f.Time, f.User)
	done()
	if err != nil {
		return logging.Failure("upstream_error", err)
	}
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusBadRequest && res.StatusCode != http.StatusNotFound {
		res.Body.Close()
		return logging.Failure("upstream_error", fmt.Errorf("Last.fm HTTP status %d", res.StatusCode))
	}
	done = logging.Stage(ctx, "album_parsing")
	albums, err := workers.Albums{}.Parse(res)
	done()
	if err != nil {
		var api *workers.APIError
		if errors.As(err, &api) && api.Code == 6 && strings.EqualFold(strings.TrimSpace(api.Message), "User not found") {
			return logging.Failure("user_not_found", err)
		}
		return logging.Failure("upstream_error", err)
	}
	if res.StatusCode != http.StatusOK {
		return logging.Failure("upstream_error", fmt.Errorf("Last.fm HTTP status %d", res.StatusCode))
	}
	if len(albums) == 0 {
		return logging.Failure("no_albums", errors.New("no albums for this listening period"))
	}
	_, err = (workers.Collage{}).MakeCollageContext(ctx, albums, f.Size, f.Path)
	return err
}

func main() {
	cmd := flags.NewCommandContext(version, func(ctx context.Context, options *flags.Flags) error {
		if err := os.MkdirAll(filepath.Join(".", "generated"), os.ModePerm); err != nil {
			return err
		}
		logging.FromContext(ctx).Info("starting collage generation")
		return generateCollageContext(ctx, options)
	})
	cmd.AddCommand(flags.NewServeCommandContext(func(parent context.Context, address string) error {
		ctx, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
		defer stop()
		return server.Run(ctx, address)
	}))
	if err := cmd.Execute(); err != nil {
		if !logging.IsReported(err) {
			fmt.Fprintln(os.Stderr, logging.Sanitize(err.Error()))
		}
		os.Exit(1)
	}
}
