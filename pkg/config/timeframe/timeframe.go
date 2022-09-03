package timeframe

import "fmt"

//go:generate stringer -type=TimeFrame -linecomment
type TimeFrame int

const (
	Week       TimeFrame = iota // 7day
	Month                       // 1month
	ThreeMonth                  // 3month
	SixMonth                    // 6month
	Year                        // 12month
	Overall                     // overall
)

var timeFrameMap = map[string]TimeFrame{
	"7day":    Week,
	"1month":  Month,
	"3month":  ThreeMonth,
	"6month":  SixMonth,
	"12month": Year,
	"overall": Overall,
}

func ParseString(str string) (TimeFrame, error) {
	t, ok := timeFrameMap[str]
	if !ok {
		return t, fmt.Errorf("%s invaild string", str)
	}
	return t, nil
}
