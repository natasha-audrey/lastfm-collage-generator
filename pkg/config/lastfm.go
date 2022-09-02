package config

import (
	"log"
	"os"
)

// Utility function to get an environment variable, and if not set
// use a default value
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// LastFmConfig Struct of the configuration variables
// Initialized in its own file.
type LastFmConfig struct {
	ApplicationName string
	APIKey          string
	SharedSecret    string
	RegisteredTo    string
	BaseURL         string
	Logger          log.Logger
}

// Init Initialize the config
func (c *LastFmConfig) Init() {
	c.ApplicationName = os.Getenv("APPLICATION_NAME")
	c.APIKey = os.Getenv("API_KEY")
	c.SharedSecret = os.Getenv("SHARED_SECRET")
	c.RegisteredTo = os.Getenv("REGISTERED_TO")
	c.BaseURL = getEnv("BASE_URL", "http://ws.audioscrobbler.com/2.0")
	c.Logger = *log.Default()
}
