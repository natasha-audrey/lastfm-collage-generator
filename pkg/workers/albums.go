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
		return nil, nil
	}
	defer res.Body.Close()

	var result map[string]map[string][]map[string]interface{}
	json.Unmarshal(responseBodyBytes, &result)

	var albums []model.Album
	for _, value := range result["topalbums"]["album"] {
		var album model.Album
		album.Name = value["name"].(string)
		album.Listens = value["playcount"].(string)
		album.Artist = value["artist"].(map[string]interface{})["name"].(string)
		album.Image = value["image"].([]interface{})[3].(map[string]interface{})["#text"].(string)
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
