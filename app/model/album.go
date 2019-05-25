package model

import "encoding/json"

// Album is the data object containing album data
type Album struct {
	Artist  string
	Name    string
	Listens string
	Image   string
}

// GetAlbums maps the album data to the album object
func GetAlbums(albumData []byte) []Album {
	var result map[string]map[string][]map[string]interface{}
	json.Unmarshal(albumData, &result)
	var albums []Album
	for _, value := range result["topalbums"]["album"] {
		var album Album
		album.Name = value["name"].(string)
		album.Listens = value["playcount"].(string)
		album.Artist = value["artist"].(map[string]interface{})["name"].(string)
		album.Image = value["image"].([]interface{})[2].(map[string]interface{})["#text"].(string)

		albums = append(albums, album)
	}
	return albums
}
