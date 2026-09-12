package workers

import (
	"encoding/json"
	"io"
	"natasha-audrey/lastfm-collage-generator/pkg/model"
	"net/http"
	"path"
	"regexp"
)

// Worker to parse API responses
type Albums struct{}

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

	var albums []model.Album
	for _, value := range result.TopAlbums["album"] {
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
