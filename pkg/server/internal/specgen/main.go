// Command specgen writes the server's checked-in OpenAPI document.
package main

import (
	"log"
	"os"

	"natasha-audrey/lastfm-collage-generator/pkg/server"
)

func main() {
	data, err := server.GenerateSpec()
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile("openapi.json", data, 0644); err != nil {
		log.Fatal(err)
	}
}
