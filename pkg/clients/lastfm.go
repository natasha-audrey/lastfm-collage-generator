// Package clients includes clients for making API requests.
package clients

import (
	"natasha-audrey/lastfm-collage-generator/pkg/config"
	"natasha-audrey/lastfm-collage-generator/pkg/config/timeframe"
	"net/http"
)

type LastFmClient struct {
	http   *http.Client
	config config.LastFmConfig
}

func NewLastFmClientFromHTTP(httpClient *http.Client) *LastFmClient {
	config := &config.LastFmConfig{}
	config.Init()
	client := &LastFmClient{http: httpClient, config: *config}
	return client
}

func (c LastFmClient) GetTopAlbums(tf timeframe.TimeFrame, user string) (*http.Response, error) {
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

	c.config.Logger.Println(req.URL)

	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}

	return res, nil
}
