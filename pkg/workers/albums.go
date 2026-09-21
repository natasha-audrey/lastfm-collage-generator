// Package workers parses album responses and renders album collages.
package workers

import (
	"encoding/json"
	"errors"
	"io"
	"natasha-audrey/lastfm-collage-generator/pkg/model"
	"net/http"
	"path"
	"regexp"
)

// Albums converts Last.fm top-album responses into collage metadata.
type Albums struct{}

// Parse reads a Last.fm JSON response and assigns sanitized local artwork paths
// under ./generated. It returns read, JSON, or Last.fm API errors.
// The response must have a non-nil body and each album must have an image entry.
// Parse closes the body after a successful read, including when decoding fails.
func (a Albums) Parse(res *http.Response) ([]model.Album, error) {
	responseBodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var result model.LastFMTopAlbums
	err = json.Unmarshal(responseBodyBytes, &result)
	if err != nil {
		return nil, err
	}
	if result.Error != 0 {
		return nil, errors.New(result.Message)
	}

	var albums []model.Album
	for _, value := range result.TopAlbums.Album {
		var album model.Album
		album.Name = value.Name
		album.Listens = value.Playcount
		album.Artist = value.Artist["name"]
		album.Image = value.Image[len(value.Image)-1]["#text"]
		fileReg := regexp.MustCompile(`[^0-9A-Za-z_\-]`)
		artist := fileReg.ReplaceAllString(album.Artist, "_")
		name := fileReg.ReplaceAllString(album.Name, "_")
		ext := path.Ext(album.Image)
		if ext == "" {
			ext = ".png"
		}
		album.Ext = ext
		album.LocalImage = "./generated/" + artist + "_" + name

		albums = append(albums, album)
	}
	return albums, nil
}
