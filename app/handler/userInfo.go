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
	albums := model.GetAlbums(responseBodyBytes)
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
