// Package config loads Last.fm configuration from environment variables.
package config

import (
	"os"
)

// getEnv returns the environment variable value, or fallback if it is unset.
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// LastFmConfig holds application credentials and the Last.fm API endpoint.
type LastFmConfig struct {
	ApplicationName string
	APIKey          string
	SharedSecret    string
	RegisteredTo    string
	BaseURL         string
}

// Init loads APPLICATION_NAME, API_KEY, SHARED_SECRET, REGISTERED_TO, and
// BASE_URL from the environment. An unset BASE_URL defaults to
// http://ws.audioscrobbler.com/2.0; an explicitly empty value is preserved.
func (c *LastFmConfig) Init() {
	c.ApplicationName = os.Getenv("APPLICATION_NAME")
	c.APIKey = os.Getenv("API_KEY")
	c.SharedSecret = os.Getenv("SHARED_SECRET")
	c.RegisteredTo = os.Getenv("REGISTERED_TO")
	c.BaseURL = getEnv("BASE_URL", "http://ws.audioscrobbler.com/2.0")
}
