// Package timeframe defines the listening periods accepted by Last.fm.
package timeframe

import "fmt"

// TimeFrame identifies a listening period. Its String method returns the
// corresponding Last.fm API value, such as "7day". The zero value is Week.
//
//go:generate stringer -type=TimeFrame -linecomment
type TimeFrame int

const (
	// Week covers the last seven days.
	Week TimeFrame = iota // 7day
	// Month covers the last month.
	Month // 1month
	// ThreeMonth covers the last three months.
	ThreeMonth // 3month
	// SixMonth covers the last six months.
	SixMonth // 6month
	// Year covers the last twelve months.
	Year // 12month
	// Overall covers all recorded listening history.
	Overall // overall
)

var timeFrameMap = func() map[string]TimeFrame {
	periods := make(map[string]TimeFrame)
	for _, period := range Values() {
		periods[period.String()] = period
	}
	return periods
}()

// ParseString parses a case-sensitive Last.fm period: 7day, 1month, 3month,
// 6month, 12month, or overall. It returns an error for any other value.
func ParseString(str string) (TimeFrame, error) {
	t, ok := timeFrameMap[str]
	if !ok {
		return t, fmt.Errorf("%s invaild string", str)
	}
	return t, nil
}

// Values returns the supported listening periods in declaration order.
func Values() []TimeFrame {
	return []TimeFrame{Week, Month, ThreeMonth, SixMonth, Year, Overall}
}
