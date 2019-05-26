package handler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"

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

func downloadImages(albums []model.Album) {
	for _, album := range albums {
		if album.Image != "" {
			response, err := http.Get(album.Image)
			if err != nil {
				fmt.Println("Error getting images")
				log.Fatal(err)
				return
			}
			defer response.Body.Close()
			fileReg := regexp.MustCompile(`[^0-9A-Za-z_\-]`)
			artist := fileReg.ReplaceAllString(album.Artist, "_")
			name := fileReg.ReplaceAllString(album.Name, "_")

			// if _, err := os.Stat("./web/images/" + artist + "_" + name + ".png"); os.IsNotExist(err) {
			AddText("./web/images/"+artist+"_"+name+".png", 0, 0, []string{album.Artist, album.Name}, response.Body)
			// file, e := os.Create("./web/images/" + artist + "_" + name + ".png")
			// if e != nil {
			// 	fmt.Println("Error making image")
			// 	log.Fatal(e)
			// }
			// defer file.Close()

			// // _, err = io.Copy(file, response.Body)
			// // if err != nil {
			// // 	log.Fatal(err)
			// // }
		}
		// }
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
	js, err := json.Marshal(albums)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	downloadImages(albums)
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}
