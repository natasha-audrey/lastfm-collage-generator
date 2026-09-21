// Package static embeds the fonts used to label album artwork.
package static

import "embed"

// Fonts contains the bundled fonts and their licenses.
//
//go:embed IBMPlexMono-Text.ttf fonts/*.ttf fonts/*.otf fonts/*.txt
var Fonts embed.FS
