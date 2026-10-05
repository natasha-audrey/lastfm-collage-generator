package server

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers/legacy"

	"natasha-audrey/lastfm-collage-generator/pkg/config/timeframe"
)

func TestDecodeOptions_Parsing(t *testing.T) {
	for _, tc := range []struct {
		query string
		want  options
	}{
		{"user=listener", options{"listener", timeframe.Week, 5}},
		{"user=%20listener%20&size=%2B05", options{"listener", timeframe.Week, 5}},
		{"user=a&size=03&timeframe=overall", options{"a", timeframe.Overall, 3}},
	} {
		got, err := decodeOptions(tc.query)
		if err != nil || got != tc.want {
			t.Fatalf("%s: got %+v (%v), want %+v", tc.query, got, err, tc.want)
		}
	}
	for _, query := range []string{
		"", "user=", "user=%20", "user=a&size=2", "user=a&size=11", "user=a&size=x",
		"user=a&size=", "user=a&size=%205", "user=a&timeframe=", "user=a&timeframe=no",
		"user=a&timeframe=Overall", "user=a&user=b", "user=a&size=5&size=5", "user=a&path=foo",
		"user=%zz", "user=a;size=5", "user[a]=x", "user=a&timeframe=7day&timeframe=overall",
	} {
		t.Run(query, func(t *testing.T) {
			if _, err := decodeOptions(query); err == nil {
				t.Fatal("accepted invalid request")
			}
		})
	}
}

func TestServeHTTP_GenerationContract(t *testing.T) {
	doc, err := openapi3.NewLoader().LoadFromData(specification)
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.Validate(context.Background()); err != nil {
		t.Fatal(err)
	}
	router, err := legacy.NewRouter(doc)
	if err != nil {
		t.Fatal(err)
	}
	openapi3filter.RegisterBodyDecoder("image/png", openapi3filter.FileBodyDecoder)
	defer openapi3filter.UnregisterBodyDecoder("image/png")
	for _, tc := range []struct {
		query string
		want  options
	}{
		{"user=listener", options{"listener", timeframe.Week, 5}},
		{"user=listener&timeframe=overall&size=10", options{"listener", timeframe.Overall, 10}},
		{"user=listener&timeframe=7day&size=3", options{"listener", timeframe.Week, 3}},
		{"user=listener&timeframe=1month&size=5", options{"listener", timeframe.Month, 5}},
		{"user=listener&timeframe=3month&size=6", options{"listener", timeframe.ThreeMonth, 6}},
		{"user=listener&timeframe=6month&size=7", options{"listener", timeframe.SixMonth, 7}},
		{"user=listener&timeframe=12month&size=8", options{"listener", timeframe.Year, 8}},
	} {
		t.Run(tc.query, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, generatePath+"?"+tc.query, nil)
			route, params, err := router.FindRoute(req)
			if err != nil {
				t.Fatal(err)
			}
			input := &openapi3filter.RequestValidationInput{Request: req, PathParams: params, Route: route, Options: &openapi3filter.Options{SkipSettingDefaults: true}}
			if err := openapi3filter.ValidateRequest(context.Background(), input); err != nil {
				t.Fatal(err)
			}
			h := newHandler(func(_ context.Context, got options) ([]byte, error) {
				if got != tc.want {
					t.Errorf("got %+v, want %+v", got, tc.want)
				}
				var buf bytes.Buffer
				err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 1, 1)))
				return buf.Bytes(), err
			}, time.Second)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
			response := &openapi3filter.ResponseValidationInput{RequestValidationInput: input, Status: w.Code, Header: w.Header(), Options: &openapi3filter.Options{IncludeResponseStatus: true}}
			response.SetBodyBytes(w.Body.Bytes())
			if err := openapi3filter.ValidateResponse(context.Background(), response); err != nil {
				t.Fatal(err)
			}
			if w.Code != 200 || w.Header().Get("Content-Type") != "image/png" || w.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("response: %v", w)
			}
			if _, err := png.Decode(bytes.NewReader(w.Body.Bytes())); err != nil {
				t.Fatal(err)
			}
			response.Header.Set("Cache-Control", "public")
			response.SetBodyBytes(w.Body.Bytes())
			if err := openapi3filter.ValidateResponse(context.Background(), response); err == nil {
				t.Fatal("accepted invalid documented header")
			}
			response.Header.Set("Cache-Control", "no-store")
			response.Status = 201
			if err := openapi3filter.ValidateResponse(context.Background(), response); err == nil {
				t.Fatal("accepted undocumented status")
			}
		})
	}
}
