package handler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/nathanyocum/lastfm-collage-generator/app/model"
	"github.com/nathanyocum/lastfm-collage-generator/configs"
)

// GetAlbums maps the album data to the album object
func GetAlbums(albumData []byte) []model.Album {
	var result map[string]map[string][]map[string]interface{}
	json.Unmarshal(albumData, &result)
	var albums []model.Album
	for _, value := range result["topalbums"]["album"] {
		var album model.Album
		album.Name = value["name"].(string)
		album.Listens = value["playcount"].(string)
		album.Artist = value["artist"].(map[string]interface{})["name"].(string)
		album.Image = value["image"].([]interface{})[2].(map[string]interface{})["#text"].(string)

		albums = append(albums, album)
	}
	return albums
}

// GetWeeklyTopAlbums Returns the weekly tracks
func GetWeeklyTopAlbums(w http.ResponseWriter, r *http.Request) {
	var config configs.LastFmConfig
	vars := mux.Vars(r)
	config.Init()
	URL := "http://ws.audioscrobbler.com/2.0/?method=user.gettopalbums&user=" + vars["user"] + "&api_key=" + config.APIKey + "&period=7day&format=json"
	response, err := http.Get(URL)
	if err != nil {
		log.Print(err)
	}
	defer response.Body.Close()
	responseBodyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Println(err)
		return
	}
	// fmt.Fprintf(w, "%s", string(responseBodyBytes))
	albums := GetAlbums(responseBodyBytes)
	if albums == nil {
		fmt.Fprintf(w, "500: Error getting albums!\n")
		return
	}
	js, err := json.Marshal(albums)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}
