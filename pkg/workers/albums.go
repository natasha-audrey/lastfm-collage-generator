// Package workers parses album responses and renders album collages.
package workers

import (
	"encoding/json"
	"io"
	"natasha-audrey/lastfm-collage-generator/pkg/model"
	"net/http"
	"path"
	"regexp"
)

// APIError is an error reported by Last.fm.
type APIError struct {
	Code    int
	Message string
}

func (e *APIError) Error() string { return e.Message }

// Albums converts Last.fm top-album responses into collage metadata.
type Albums struct{}

// Parse reads a Last.fm JSON response and assigns sanitized local artwork paths
// under ./generated. It returns read, JSON, or Last.fm API errors.
// The response must have a non-nil body. Parse always closes it.
func (a Albums) Parse(res *http.Response) ([]model.Album, error) {
	defer res.Body.Close()
	responseBodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var result model.LastFMTopAlbums
	err = json.Unmarshal(responseBodyBytes, &result)
	if err != nil {
		return nil, err
	}
	if result.Error != 0 {
		return nil, &APIError{Code: result.Error, Message: result.Message}
	}

	var albums []model.Album
	for _, value := range result.TopAlbums.Album {
		var album model.Album
		album.Name = value.Name
		album.Listens = value.Playcount
		album.Artist = value.Artist["name"]
		if len(value.Image) > 0 {
			album.Image = value.Image[len(value.Image)-1]["#text"]
		}
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
