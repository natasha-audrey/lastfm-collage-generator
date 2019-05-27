package app

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/nathanyocum/lastfm-collage-generator/app/handler"
	"github.com/nathanyocum/lastfm-collage-generator/configs"
)

// App sets up router and db
type App struct {
	Router *mux.Router
	Config configs.LastFmConfig
}

// Init the app
func (a *App) Init() {
	var config configs.LastFmConfig
	config.Init()
	a.Config = config

	a.Router = mux.NewRouter().StrictSlash(true)
	a.setRouters()
}

func (a *App) setRouters() {
	a.Get("/api/v1/{time}/{size}/{user}", handler.GetTopAlbums)

	a.Router.PathPrefix("/").Handler(http.FileServer(http.Dir("./web/")))
}

// Get wraps the router for GET method
func (a *App) Get(path string, f func(w http.ResponseWriter, r *http.Request)) {
	a.Router.HandleFunc(path, f).Methods("GET")
}

// Run Start the App
func (a *App) Run() {
	if a.Config.Port != "" {
		fmt.Println("Listening on http://localhost:" + a.Config.Port)
		log.Fatal(http.ListenAndServe(":"+a.Config.Port, a.Router))
	} else {
		fmt.Println("Listening on http://localhost:5000/ (no env vars)")
		log.Fatal(http.ListenAndServe(":5000", a.Router))
	}
}
