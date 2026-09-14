package workers

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/iotest"
)

func TestAlbumsParse_LastFMError(t *testing.T) {
	response := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"error": 10, "message": "Invalid API Key"}`)),
	}

	albums, err := (Albums{}).Parse(response)
	if err == nil || err.Error() != "Invalid API Key" {
		t.Fatalf("Parse() error = %v, want Invalid API Key", err)
	}
	if albums != nil {
		t.Fatalf("Parse() albums = %v, want nil", albums)
	}
}

func TestAlbumsParse_ReadError(t *testing.T) {
	wantErr := errors.New("response body read failed")
	body := io.NopCloser(iotest.ErrReader(wantErr))
	t.Cleanup(func() { body.Close() })
	response := &http.Response{Body: body}

	albums, err := (Albums{}).Parse(response)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Parse() error = %v, want %v", err, wantErr)
	}
	if albums != nil {
		t.Fatalf("Parse() albums = %v, want nil", albums)
	}
}

func TestAlbumsParse_JSONUnmarshallErr(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{`))
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
	if err == nil {
		t.Fatal("Expected error not thrown")
	}
}

func TestAlbumsParse_Success(t *testing.T) {
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
	if err != nil {
		t.Fatalf("Error unmarshalling: %q", err)
	}
	if len(albums) != 2 {
		t.Fatalf("len(albums) = %v, want %v", len(albums), 2)
	}
}
