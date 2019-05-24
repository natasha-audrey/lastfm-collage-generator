package main

import (
	"fmt"
	"html"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	var config LastFmConfig
	config.Init()
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", index)
	fmt.Println("Listening on http://localhost:5000")
	log.Fatal(http.ListenAndServe(":"+config.Port, router))
}

func index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
}
