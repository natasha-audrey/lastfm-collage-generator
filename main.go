// Package main.
//
// Command line entry point for last-fm-collage-generator.
package main

import (
	"log"
	"natasha-audrey/lastfm-collage-generator/pkg/clients"
	"natasha-audrey/lastfm-collage-generator/pkg/flags"
	"natasha-audrey/lastfm-collage-generator/pkg/workers"
	"net/http"
	"os"
	"path/filepath"
)

func generateCollage(f *flags.Flags) {
	client := clients.NewLastFmClientFromHTTP(&http.Client{})
	res, err := client.GetTopAlbums(f.Time, "n8yo")
	if err != nil {
		panic(err)
	}
	albums, err := workers.Albums{}.Parse(res)
	if err != nil {
		panic(err)
	}
	workers.Collage{}.MakeCollage(albums, f.Size, f.Path)
}

func main() {
	flags, err := flags.Parse()
	if err != nil {
		log.Fatal(err)
	}
	newpath := filepath.Join(".", "generated")
	os.MkdirAll(newpath, os.ModePerm)
	generateCollage(flags)
}
