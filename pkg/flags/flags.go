// Package flags parses and validates command-line options for collage generation.
package flags

import (
	"flag"
	"fmt"
	"natasha-audrey/lastfm-collage-generator/pkg/config/timeframe"
)

// Flags holds the parsed collage options.
type Flags struct {
	// Time selects the listening period.
	Time timeframe.TimeFrame
	// Size is the number of rows and columns, from 3 to 10.
	Size int
	// Path is the output image filename.
	Path string
	// User is the Last.fm username to query.
	User string
}

// Parse registers and parses -t, -s, -p, and -u on flag.CommandLine.
// Defaults are 7day, 5, ./collage.png, and tashayasha, respectively.
// It prints usage on validation failure and returns the parsed options with an error.
// Output path validation may temporarily create and remove a file.
func Parse() (*Flags, error) {
	t := timeOption.Option()
	s := sizeOption.Option()
	p := pathOption.Option()
	u := userOption.Option()
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

	user, _ := userOption.Parse(*u)

	if errors != nil {
		flag.Usage()
	}

	return &Flags{
		time,
		size,
		path,
		user,
	}, errors
}
