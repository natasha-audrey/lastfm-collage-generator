package app

import (
	"net/http"
	"testing"

	"github.com/gorilla/mux"
	"github.com/nathanyocum/lastfm-collage-generator/configs"
)

func TestApp_Init(t *testing.T) {
	type fields struct {
		Router *mux.Router
		Config configs.LastFmConfig
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &App{
				Router: tt.fields.Router,
				Config: tt.fields.Config,
			}
			a.Init()
		})
	}
}

func TestApp_setRouters(t *testing.T) {
	type fields struct {
		Router *mux.Router
		Config configs.LastFmConfig
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &App{
				Router: tt.fields.Router,
				Config: tt.fields.Config,
			}
			a.setRouters()
		})
	}
}

func TestApp_Get(t *testing.T) {
	type fields struct {
		Router *mux.Router
		Config configs.LastFmConfig
	}
	type args struct {
		path string
		f    func(w http.ResponseWriter, r *http.Request)
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &App{
				Router: tt.fields.Router,
				Config: tt.fields.Config,
			}
			a.Get(tt.args.path, tt.args.f)
		})
	}
}

func TestApp_Run(t *testing.T) {
	type fields struct {
		Router *mux.Router
		Config configs.LastFmConfig
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &App{
				Router: tt.fields.Router,
				Config: tt.fields.Config,
			}
			a.Run()
		})
	}
}
