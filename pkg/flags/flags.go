package flags

import (
	"flag"
	"fmt"
	"natasha-audrey/lastfm-collage-generator/pkg/config/timeframe"
)

type Flags struct {
	Time timeframe.TimeFrame
	Size int
	Path string
}

// Parse parses command line flags.
func Parse() (*Flags, error) {
	t := timeOption.Option()
	s := sizeOption.Option()
	p := pathOption.Option()
	flag.Parse()

	var errors error = nil

	time, err := timeOption.Parse(*t)
	if err != nil {
		errors = fmt.Errorf("%w", err)
	}

	size, err := sizeOption.Parse(*s)
	if err != nil {
		errors = fmt.Errorf("%w", err)
	}

	path, err := pathOption.Parse(*p)
	if err != nil {
		errors = fmt.Errorf("%w", err)
	}

	if errors != nil {
		flag.Usage()
	}

	return &Flags{
		time,
		size,
		path,
	}, errors
}
