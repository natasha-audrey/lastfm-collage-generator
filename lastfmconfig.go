package main

import "os"

// LastFmConfig Struct of the configuration variables
// Initialized in its own file.
//
type LastFmConfig struct {
	ApplicationName string
	APIKey          string
	SharedSecret    string
	RegisteredTo    string
	Port            string
}

// Init Initialize the config
func (c *LastFmConfig) Init() {
	c.ApplicationName = os.Getenv("APPLICATION_NAME")
	c.APIKey = os.Getenv("API_KEY")
	c.SharedSecret = os.Getenv("SHARED_SECRET")
	c.RegisteredTo = os.Getenv("REGISTERED_TO")
	c.Port = os.Getenv("PORT")
}
