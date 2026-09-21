// Package model defines album data and Last.fm API response types.
package model

// Album holds listening metadata and artwork locations for a collage tile.
type Album struct {
	Artist string
	Name   string
	// Listens is the play count as returned by Last.fm.
	Listens string
	// Image is the remote artwork URL, or empty when no artwork is available.
	Image string
	// LocalImage is the local artwork path without its file extension.
	LocalImage string
	// Ext is the source artwork extension, including the leading dot.
	Ext string
}
