package model

// LastFMTopAlbums represents a Last.fm user.gettopalbums JSON response.
type LastFMTopAlbums struct {
	TopAlbums lastFmAlbums `json:"topalbums"`
	// Error is the API error code; zero indicates no reported API error.
	Error int `json:"error"`
	// Message describes an API error.
	Message string `json:"message"`
}

type lastFmAlbums struct {
	Album []lastfmAlbum `json:"album"`
}

type lastfmAlbum struct {
	Name      string              `json:"name"`
	Playcount string              `json:"playcount"`
	Artist    map[string]string   `json:"artist"`
	Image     []map[string]string `json:"image"`
}
