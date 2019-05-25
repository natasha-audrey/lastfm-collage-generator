package handler

import (
	"fmt"
	"html"
	"net/http"
)

// Index handles the index of the API
func Index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
}
