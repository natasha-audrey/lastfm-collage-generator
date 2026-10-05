package server

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestGenerateSpec_Freshness(t *testing.T) {
	t.Setenv("API_KEY", "")
	first, err := GenerateSpec()
	if err != nil {
		t.Fatal(err)
	}
	second, err := GenerateSpec()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("specification generation is not deterministic")
	}
	if !bytes.Equal(first, specification) {
		t.Fatal("stale specification: run go generate ./pkg/server")
	}
}

func TestServeHTTP_DocumentationWhileBusy(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	h := newHandler(func(context.Context, options) ([]byte, error) {
		close(started)
		<-release
		return nil, nil
	}, time.Minute)
	finished := make(chan struct{})
	go func() { defer close(finished); request(h, "/v1/generate?user=a") }()
	<-started
	defer func() { close(release); <-finished }()
	if w := request(h, "/v1/generate?user=b"); w.Code != 503 {
		t.Fatalf("not busy: %d", w.Code)
	}
	if w := request(h, specificationPath); w.Code != 200 || w.Header().Get("Content-Type") != "application/json" || !bytes.Equal(w.Body.Bytes(), specification) {
		t.Fatal("specification unavailable while busy")
	}
	for _, path := range []string{documentationPath, documentationPath + "/"} {
		w := request(h, path)
		if w.Code != 200 || !strings.Contains(w.Body.String(), specificationPath) {
			t.Fatalf("offline UI: %d %s", w.Code, w.Body.String())
		}
		for _, match := range regexp.MustCompile(`(?:src|href)="([^"]+)"`).FindAllStringSubmatch(w.Body.String(), -1) {
			if !strings.HasPrefix(match[1], documentationPath+"/") {
				t.Fatalf("external UI dependency: %s", match[1])
			}
		}
		if !strings.Contains(w.Body.String(), "validatorUrl: null") {
			t.Fatal("external specification validator enabled")
		}
	}
	// All UI dependencies are served by the embedded local asset handler.
	for _, asset := range []string{"swagger-ui.css", "swagger-ui-bundle.js", "swagger-ui-standalone-preset.js", "favicon-32x32.png"} {
		w := request(h, documentationPath+"/"+asset)
		if w.Code != 200 || w.Body.Len() == 0 {
			t.Fatalf("asset %s unavailable: %d", asset, w.Code)
		}
	}
	for _, path := range []string{"/openapi.json", "/docs", "/v1/docs-other"} {
		if w := request(h, path); w.Code != 404 {
			t.Fatalf("unexpected route %s: %d", path, w.Code)
		}
	}
	for _, path := range []string{specificationPath, documentationPath} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, path, nil))
		if w.Code != 405 || w.Header().Get("Allow") != "GET" {
			t.Fatalf("method response: %v", w)
		}
	}
}
