package server

import (
	"encoding/json"
	"net/http"

	"github.com/swaggest/openapi-go"
	"github.com/swaggest/openapi-go/openapi3"
)

//go:generate go run ./internal/specgen

// GenerateSpec builds the deterministic HTTP contract without credentials or a server.
func GenerateSpec() ([]byte, error) {
	r := openapi3.NewReflector()
	r.SpecEns().Info.Title = "Last.fm collage generator"
	r.SpecEns().Info.Version = "1.0.0"
	oc, err := r.NewOperationContext(http.MethodGet, generatePath)
	if err != nil {
		return nil, err
	}
	oc.SetID("generateCollage")
	oc.SetSummary("Generate a PNG collage of a user's top albums")
	oc.SetDescription("Unknown and duplicate query parameters are rejected, as are malformed URL escapes and unescaped semicolons. Defaults apply only to omitted parameters. Only one generation runs at a time; concurrent requests receive 503. Generation has a 60-second deadline; after a timeout the slot remains busy until the worker exits. A disconnected client cancels generation. Documentation endpoints remain available while generation is busy.")
	oc.AddReqStructure(new(generateRequest))
	oc.AddRespStructure(new(responseHeaders), func(cu *openapi.ContentUnit) {
		cu.HTTPStatus = http.StatusOK
		cu.ContentType = "image/png"
		cu.Format = "binary"
		cu.Description = "PNG collage. Each grid cell is 300 × 300 pixels; unavailable artwork uses a black tile and unused cells remain black."
	})
	for _, failure := range []struct {
		status      int
		description string
	}{
		{400, "invalid_request: missing or invalid query parameters"},
		{404, "user_not_found: Last.fm user not found"},
		{405, "method_not_allowed: use GET; the Allow header is GET"},
		{422, "no_albums: no albums for this listening period"},
		{500, "internal_error: collage generation failed"},
		{502, "upstream_error: could not retrieve Last.fm data"},
		{503, "busy: another generation is running"},
		{504, "timeout: generation exceeded 60 seconds"},
	} {
		var response any = new(errorHTTPResponse)
		if failure.status == http.StatusMethodNotAllowed {
			response = new(methodErrorResponse)
		}
		oc.AddRespStructure(response, func(cu *openapi.ContentUnit) {
			cu.HTTPStatus = failure.status
			cu.ContentType = "application/json"
			cu.Description = failure.description
		})
	}
	if err := r.AddOperation(oc); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(r.SpecEns(), "", "  ")
	return append(data, '\n'), err
}
