package model

type LastFMTopAlbums struct {
	TopAlbums map[string][]lastfmAlbum `json:"topalbums"`
}

type lastfmAlbum struct {
	Name      string              `json:"name"`
	Playcount string              `json:"playcount"`
	Artist    map[string]string   `json:"artist"`
	Image     []map[string]string `json:"image"`
}
