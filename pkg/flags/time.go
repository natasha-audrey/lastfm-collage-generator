package flags

import (
	"flag"
	"fmt"
	"natasha-audrey/lastfm-collage-generator/pkg/config/timeframe"
)

var timeOption = Option[string, timeframe.TimeFrame]{
	func() *string {
		time := flag.String(
			"t",
			"7day",
			fmt.Sprintf(
				`The time frame to generate the collage for.

Available options:
  %s %s %s %s %s %s`,
				timeframe.Week.String(),
				timeframe.Month.String(),
				timeframe.ThreeMonth.String(),
				timeframe.SixMonth.String(),
				timeframe.Year.String(),
				timeframe.Overall.String(),
			),
		)
		return time
	},

	func(str string) (timeframe.TimeFrame, error) {
		t, err := timeframe.ParseString(str)
		return t, err
	},
}
