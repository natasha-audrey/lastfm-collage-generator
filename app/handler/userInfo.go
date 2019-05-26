package handler

import (
	"encoding/json"
	"fmt"
	"image/png"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"

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
		album.Image = value["image"].([]interface{})[3].(map[string]interface{})["#text"].(string)
		fileReg := regexp.MustCompile(`[^0-9A-Za-z_\-]`)
		artist := fileReg.ReplaceAllString(album.Artist, "_")
		name := fileReg.ReplaceAllString(album.Name, "_")
		album.LocalImage = "./web/images/" + artist + "_" + name + ".png"

		albums = append(albums, album)
	}
	downloadImages(albums)
	return albums
}

func downloadImages(albums []model.Album) {
	for _, album := range albums {
		if album.Image != "" {

			// If image exists don't bother making a new one
			if _, err := os.Stat(album.LocalImage); os.IsNotExist(err) {
				response, err := http.Get(album.Image)
				if err != nil {
					fmt.Println("Error getting images")
					log.Fatal(err)
					return
				}
				defer response.Body.Close()
				AddText(
					album.LocalImage,
					0,
					0,
					[]string{album.Artist, album.Name},
					response.Body)
			}
		} else {
			AddTextNoImage(album.LocalImage, 0, 0, []string{album.Artist, album.Name})
		}
	}
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
	albums := GetAlbums(responseBodyBytes)
	if albums == nil {
		http.Error(w, "Error getting albums", http.StatusInternalServerError)
		return
	}

	size, err := strconv.Atoi(vars["size"])
	if err != nil {
		http.Error(w, "Size not a number", http.StatusUnprocessableEntity)
	}
	if size < 8 && size > 0 {
		image, err := MakeCollage(albums, size)
		if err != nil || image == nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		png.Encode(w, image)
	} else {
		http.Error(w, vars["size"]+" invalid, needs to be between 0 and 7", http.StatusUnprocessableEntity)
		return
	}
}
