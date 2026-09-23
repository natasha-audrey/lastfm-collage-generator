// Package main.
//
// Command line entry point for last-fm-collage-generator.
package main

import (
	"fmt"
	"log"
	"natasha-audrey/lastfm-collage-generator/pkg/clients"
	"natasha-audrey/lastfm-collage-generator/pkg/flags"
	"natasha-audrey/lastfm-collage-generator/pkg/workers"
	"net/http"
	"os"
	"path/filepath"
)

// version is the CLI release version, maintained by release-please.
const version = "v0.7.0" // x-release-please-version

func generateCollage(f *flags.Flags) {
	client := clients.NewLastFmClientFromHTTP(&http.Client{})
	res, err := client.GetTopAlbums(f.Time, f.User)
	if err != nil {
		log.Fatal(err)
	}
	albums, err := workers.Albums{}.Parse(res)
	if err != nil {
		log.Fatal(err)
	}
	workers.Collage{}.MakeCollage(albums, f.Size, f.Path)
}

func main() {
	flags, err := flags.Parse()
	if err != nil {
		log.Fatal(err)
	}
	if flags.Version {
		fmt.Println(version)
		return
	}
	newpath := filepath.Join(".", "generated")
	os.MkdirAll(newpath, os.ModePerm)
	generateCollage(flags)
}
