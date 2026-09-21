// Package clients includes clients for making API requests.
package clients

import (
	"errors"
	"log/slog"
	"natasha-audrey/lastfm-collage-generator/pkg/config"
	"natasha-audrey/lastfm-collage-generator/pkg/config/timeframe"
	"net/http"
)

// LastFmClient requests album listening data from Last.fm.
type LastFmClient struct {
	http   *http.Client
	config config.LastFmConfig
}

// NewLastFmClientFromHTTP creates a client using a non-nil HTTP client and
// configuration loaded from environment variables.
func NewLastFmClientFromHTTP(httpClient *http.Client) *LastFmClient {
	config := &config.LastFmConfig{}
	config.Init()
	client := &LastFmClient{http: httpClient, config: *config}
	return client
}

func validateTopAlbumsInput(c LastFmClient, user string) error {
	if user == "" {
		return errors.New("User cannot be blank")
	}
	if c.config.APIKey == "" {
		return errors.New("Missing API Key")
	}
	return nil
}

// GetTopAlbums requests up to 100 top albums for user over tf.
// It returns the raw response without checking its status or decoding API errors.
// The caller is responsible for closing the response body.
func (c LastFmClient) GetTopAlbums(tf timeframe.TimeFrame, user string) (*http.Response, error) {
	err := validateTopAlbumsInput(c, user)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("GET", c.config.BaseURL, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("api_key", c.config.APIKey)
	q.Add("user", user)
	q.Add("period", tf.String())
	q.Add("format", "json")
	q.Add("method", "user.gettopalbums")
	q.Add("limit", "100")
	req.URL.RawQuery = q.Encode()

	slog.Debug(req.URL.String())

	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}

	return res, nil
}
