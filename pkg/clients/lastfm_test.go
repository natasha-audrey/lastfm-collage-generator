package clients

import (
	"errors"
	"io"
	"natasha-audrey/lastfm-collage-generator/pkg/config/timeframe"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

type errRoundTripper struct {
	err error
}

func (rt errRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, rt.err
}

func TestNewLastFmClientFromHTTP_ReadsConfigFromEnv(t *testing.T) {
	t.Setenv("API_KEY", "test-api-key")
	t.Setenv("BASE_URL", "https://lastfm.test/2.0")

	httpClient := &http.Client{}
	client := NewLastFmClientFromHTTP(httpClient)

	if client == nil {
		t.Fatal("NewLastFmClientFromHTTP() returned nil")
	}
	if client.http != httpClient {
		t.Fatal("NewLastFmClientFromHTTP() did not store the provided HTTP client")
	}
	if client.config.APIKey != "test-api-key" {
		t.Errorf("APIKey = %q, want %q", client.config.APIKey, "test-api-key")
	}
	if client.config.BaseURL != "https://lastfm.test/2.0" {
		t.Errorf("BaseURL = %q, want %q", client.config.BaseURL, "https://lastfm.test/2.0")
	}
}

func TestGetTopAlbums_BuildsRequestAndReturnsResponse(t *testing.T) {
	var gotURL *url.URL
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"topalbums":{"album":[]}}`)
	}))
	t.Cleanup(server.Close)

	t.Setenv("API_KEY", "secret-key")
	t.Setenv("BASE_URL", server.URL)

	client := NewLastFmClientFromHTTP(server.Client())
	res, err := client.GetTopAlbums(timeframe.Month, "natasha")
	if err != nil {
		t.Fatalf("GetTopAlbums() error = %v", err)
	}
	t.Cleanup(func() { _ = res.Body.Close() })

	if res.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", res.StatusCode, http.StatusOK)
	}
	if gotURL == nil {
		t.Fatal("handler was not called")
	}

	q := gotURL.Query()
	want := map[string]string{
		"api_key": "secret-key",
		"user":    "natasha",
		"period":  "1month",
		"format":  "json",
		"method":  "user.gettopalbums",
		"limit":   "100",
	}
	for key, value := range want {
		if got := q.Get(key); got != value {
			t.Errorf("query %s = %q, want %q", key, got, value)
		}
	}
}

func TestGetTopAlbums_ReturnsHTTPError(t *testing.T) {
	t.Setenv("API_KEY", "secret-key")
	t.Setenv("BASE_URL", "https://lastfm.test/2.0")

	wantErr := errors.New("network down")
	client := NewLastFmClientFromHTTP(&http.Client{Transport: errRoundTripper{err: wantErr}})

	res, err := client.GetTopAlbums(timeframe.Week, "natasha")
	if err == nil {
		if res != nil {
			_ = res.Body.Close()
		}
		t.Fatal("GetTopAlbums() error = nil, want network error")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("GetTopAlbums() error = %v, want %v", err, wantErr)
	}
	if res != nil {
		t.Errorf("GetTopAlbums() response = %v, want nil", res)
	}
}

func TestGetTopAlbums_ReturnsErrorWhenRequestCannotBeCreated(t *testing.T) {
	t.Setenv("API_KEY", "secret-key")
	t.Setenv("BASE_URL", "://not-a-valid-url")

	client := NewLastFmClientFromHTTP(&http.Client{})
	res, err := client.GetTopAlbums(timeframe.Overall, "natasha")
	if err == nil {
		if res != nil {
			_ = res.Body.Close()
		}
		t.Fatal("GetTopAlbums() error = nil, want request creation error")
	}
	if res != nil {
		t.Errorf("GetTopAlbums() response = %v, want nil", res)
	}
}
