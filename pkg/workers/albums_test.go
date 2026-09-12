package workers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAlbumsParse_err(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	t.Cleanup(server.Close)

	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatal("Error talking to server")
	}
	response, err := server.Client().Do(req)
	if err != nil {
		t.Fatal("Error talking to server")
	}

	_, err = Albums{}.Parse(response)
	if err != nil {
		t.Fatal("Expected error not thrown")
	}
}

func TestAlbumsParse_success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`
{
	"topalbums": {
		"album": [
			{
				"name": "album_name",
				"playcount": "32",
				"artist": {
					"name": "artist_name"
				},
				"image": [
					{"size": "small", "#text": "url"},
					{"size": "medium", "#text": "url"},
					{"size": "large", "#text": "url"},
					{"size": "extralarge", "#text": "url"}
				]
			},
			{
				"name": "album_name",
				"playcount": "31",
				"artist": {
					"name": "artist_name"
				},
				"image": [
					{"size": "small", "#text": "url.png"},
					{"size": "medium", "#text": "url.png"},
					{"size": "large", "#text": "url.png"},
					{"size": "extralarge", "#text": "url.png"}
				]
			}
		]
	}
}
		`))
	}))
	defer server.Close()

	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatal("Error talking to server")
	}
	response, err := server.Client().Do(req)
	if err != nil {
		t.Fatal("Error talking to server")
	}

	albums, err := Albums{}.Parse(response)
	if len(albums) != 2 {
		t.Fatal("Not expected number of albums")
	}
}
