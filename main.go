// Package main.
//
// Command line entry point for last-fm-collage-generator.
package main

import (
	"natasha-audrey/lastfm-collage-generator/pkg/clients"
	"natasha-audrey/lastfm-collage-generator/pkg/workers"
	"net/http"
	"os"
	"path/filepath"
)

func generateCollage(tf clients.TimeFrame, size int) {
	client := clients.NewLastFmClientFromHTTP(&http.Client{})
	res, err := client.GetTopAlbums(tf, "n8yo")
	if err != nil {
		panic(err)
	}
	albums, err := workers.Albums{}.Parse(res)
	if err != nil {
		panic(err)
	}
	pngPath := "./" + tf.String() + ".png"
	workers.Collage{}.MakeCollage(albums, size, pngPath)
}

func main() {
	newpath := filepath.Join(".", "generated")
	os.MkdirAll(newpath, os.ModePerm)
	generateCollage(clients.Month, 5)
	generateCollage(clients.ThreeMonth, 6)
	generateCollage(clients.SixMonth, 7)
	generateCollage(clients.Year, 10)
}
