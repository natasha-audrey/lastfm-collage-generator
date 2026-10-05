package server

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"github.com/swaggest/jsonschema-go"
	"github.com/swaggest/openapi-go/openapi3"
	"github.com/swaggest/rest"
	validation "github.com/swaggest/rest/jsonschema"
	"github.com/swaggest/rest/nethttp"
	restopenapi "github.com/swaggest/rest/openapi"
	restrequest "github.com/swaggest/rest/request"

	"natasha-audrey/lastfm-collage-generator/pkg/config/timeframe"
)

const generatePath = "/v1/generate"
const cacheControl = "no-store"

// generateRequest declares the HTTP parameters shared by decoding and OpenAPI.
type generateRequest struct {
	User   string          `query:"user" required:"true" minLength:"1" description:"Last.fm username. Leading and trailing whitespace is trimmed; a blank username is rejected."`
	Period listeningPeriod `query:"timeframe" description:"Case-sensitive Last.fm listening period. An explicitly empty value is rejected."`
	Size   gridSize        `query:"size" description:"Square grid side length. Parsed as a base-10 integer (a leading sign and zeros are accepted); whitespace and explicitly empty values are rejected."`
	_      struct{}        `query:"_" additionalProperties:"false"`
}

type listeningPeriod string

func (listeningPeriod) PrepareJSONSchema(s *jsonschema.Schema) error {
	s.WithDefault(timeframe.Week.String())
	for _, period := range timeframe.Values() {
		s.Enum = append(s.Enum, period.String())
	}
	return nil
}

type gridSize int

func (gridSize) PrepareJSONSchema(s *jsonschema.Schema) error {
	s.WithDefault(5).WithMinimum(3).WithMaximum(10)
	return nil
}

// queryName obtains transport names from the declaration instead of duplicating them.
func queryName(field string) string {
	f, ok := reflect.TypeFor[generateRequest]().FieldByName(field)
	if !ok {
		panic("unknown contract field: " + field)
	}
	return f.Tag.Get("query")
}

// strictQuery preserves the server's parsing rules before typed decoding.
func strictQuery(r *http.Request) (url.Values, error) {
	q, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return nil, errors.New("invalid query string")
	}
	allowed := map[string]bool{}
	for _, field := range reflect.VisibleFields(reflect.TypeFor[generateRequest]()) {
		if name := field.Tag.Get("query"); name != "" && name != "_" {
			allowed[name] = true
		}
	}
	for key, values := range q {
		if !allowed[key] {
			return nil, fmt.Errorf("unknown parameter %q", key)
		}
		if len(values) != 1 {
			return nil, fmt.Errorf("duplicate parameter %q", key)
		}
	}
	user, period, size := queryName("User"), queryName("Period"), queryName("Size")
	q.Set(user, strings.TrimSpace(q.Get(user)))
	if q.Get(user) == "" {
		return nil, errors.New("user is required")
	}
	if q.Has(period) && q.Get(period) == "" {
		return nil, errors.New("invalid timeframe")
	}
	if q.Has(size) {
		n, err := strconv.Atoi(q.Get(size))
		if err != nil {
			return nil, errors.New("size must be an integer between 3 and 10")
		}
		// Canonicalize the existing Atoi syntax before schema validation.
		q.Set(size, strconv.Itoa(n))
	}
	return q, nil
}

var contractDecoder, contractValidator = newContractDecoder()

func newContractDecoder() (nethttp.RequestDecoder, rest.Validator) {
	reflector := openapi3.NewReflector()
	collector := restopenapi.NewCollector(reflector)
	validator := validation.NewFactory(collector, collector).MakeRequestValidator(http.MethodGet, new(generateRequest), nil)
	factory := restrequest.NewDecoderFactory()
	factory.ApplyDefaults = true
	factory.JSONSchemaReflector = &jsonschema.Reflector{}
	factory.SetDecoderFunc(rest.ParamInQuery, strictQuery)
	return factory.MakeDecoder(http.MethodGet, new(generateRequest), nil), validator
}

func decodeOptions(raw string) (options, error) {
	var input generateRequest
	r := &http.Request{Method: http.MethodGet, URL: &url.URL{RawQuery: raw}, Header: make(http.Header)}
	if err := contractDecoder.Decode(r, &input, contractValidator); err != nil {
		var fields rest.ValidationErrors
		if errors.As(err, &fields) {
			if _, ok := fields["query:"+queryName("Period")]; ok {
				return options{}, errors.New("invalid timeframe")
			}
			if _, ok := fields["query:"+queryName("Size")]; ok {
				return options{}, errors.New("size must be an integer between 3 and 10")
			}
		}
		return options{}, err
	}
	period, err := timeframe.ParseString(string(input.Period))
	if err != nil {
		return options{}, errors.New("invalid timeframe")
	}
	return options{input.User, period, int(input.Size)}, nil
}

type errorResponse struct {
	Error errorDetail `json:"error" required:"true"`
}

type errorDetail struct {
	Code    string `json:"code" required:"true"`
	Message string `json:"message" required:"true"`
}

type responseHeaders struct {
	CacheControl string `header:"Cache-Control" required:"true" enum:"no-store"`
}

// errorHTTPResponse shares the JSON envelope and headers across error statuses.
type errorHTTPResponse struct {
	responseHeaders
	errorResponse
}

type methodErrorResponse struct {
	errorHTTPResponse
	Allow string `header:"Allow" required:"true" enum:"GET"`
}
