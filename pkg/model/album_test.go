package model

import (
	"testing"
)

// Just make sure we can init properly.
func TestAlbum(t *testing.T) {
	var album Album
	album.Artist = "Artist"
	album.Name = "Album Name"
	album.Listens = "32"
	album.Image = "image"
	album.LocalImage = "local_image"
	album.Ext = "jpeg"

	if album.Artist != "Artist" {
		t.Errorf("got %q, want Artist", album.Artist)
	}
	if album.Name != "Album Name" {
		t.Errorf("got %q, want Album Name", album.Name)
	}
	if album.Listens != "32" {
		t.Errorf("got %q, want 32", album.Listens)
	}
	if album.Image != "image" {
		t.Errorf("got %q, want image", album.Image)
	}
	if album.LocalImage != "local_image" {
		t.Errorf("got %q, want local_image", album.LocalImage)
	}
	if album.Ext != "jpeg" {
		t.Errorf("got %q, want jpeg", album.Ext)
	}
}
