package model

type LastFMTopAlbums struct {
	TopAlbums lastFmAlbums `json:"topalbums"`
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
