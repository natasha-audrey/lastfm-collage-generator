// Package clients includes clients for making API requests.
package clients

import (
	"natasha-audrey/lastfm-collage-generator/pkg/config"
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

//go:generate stringer -type=TimeFrame -linecomment
type TimeFrame int

const (
	Week       TimeFrame = iota // 7day
	Month                       // 1month
	ThreeMonth                  // 3month
	SixMonth                    // 6month
	Year                        // 12month
	Overall                     // overall
)

func (c LastFmClient) GetTopAlbums(tf TimeFrame, user string) (*http.Response, error) {
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
