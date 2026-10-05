package server

import (
	_ "embed"
	"net/http"
	"strings"

	"github.com/swaggest/swgui"
	swaggerui "github.com/swaggest/swgui/v5"
)

//go:embed openapi.json
var specification []byte

const specificationPath = "/v1/openapi.json"
const documentationPath = "/v1/docs"

var documentation = swaggerui.NewHandlerWithConfig(swgui.Config{
	Title:       "Last.fm collage generator",
	SwaggerJSON: specificationPath,
	BasePath:    documentationPath,
	SettingsUI:  map[string]string{"validatorUrl": "null"},
})

// serveDocumentation bypasses generation admission for the contract and offline UI.
func serveDocumentation(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path != specificationPath && r.URL.Path != documentationPath && !strings.HasPrefix(r.URL.Path, documentationPath+"/") {
		return false
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return true
	}
	if r.URL.Path == specificationPath {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(specification)
		return true
	}
	documentation.ServeHTTP(w, r)
	return true
}
