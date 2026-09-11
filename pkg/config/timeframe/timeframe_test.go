package timeframe

import (
	"testing"
)

func TestParseString_ValidValues(t *testing.T) {
	tests := []struct {
		input string
		want  TimeFrame
	}{
		{input: "7day", want: Week},
		{input: "1month", want: Month},
		{input: "3month", want: ThreeMonth},
		{input: "6month", want: SixMonth},
		{input: "12month", want: Year},
		{input: "overall", want: Overall},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseString(tt.input)
			if err != nil {
				t.Fatalf("ParseString(%q) error = %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("ParseString(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseString_InvalidValues(t *testing.T) {
	tests := []string{
		"",
		"week",
		"7DAY",
		"1Month",
		"overall ",
		" 7day",
		"unknown",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			got, err := ParseString(input)
			if err == nil {
				t.Fatalf("ParseString(%q) error = nil, want error", input)
			}
			if got != 0 {
				t.Errorf("ParseString(%q) = %v, want zero TimeFrame", input, got)
			}
		})
	}
}

func TestTimeFrameString_KnownValues(t *testing.T) {
	tests := []struct {
		tf   TimeFrame
		want string
	}{
		{tf: Week, want: "7day"},
		{tf: Month, want: "1month"},
		{tf: ThreeMonth, want: "3month"},
		{tf: SixMonth, want: "6month"},
		{tf: Year, want: "12month"},
		{tf: Overall, want: "overall"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.tf.String()
			if got != tt.want {
				t.Fatalf("TimeFrame(%d).String() = %q, want %q", tt.tf, got, tt.want)
			}
		})
	}
}

func TestTimeFrameString_UnknownValues(t *testing.T) {
	tests := []struct {
		tf   TimeFrame
		want string
	}{
		{tf: TimeFrame(-1), want: "TimeFrame(-1)"},
		{tf: TimeFrame(6), want: "TimeFrame(6)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.tf.String()
			if got != tt.want {
				t.Fatalf("TimeFrame(%d).String() = %q, want %q", tt.tf, got, tt.want)
			}
		})
	}
}

func TestParseString_RoundTripWithString(t *testing.T) {
	frames := []TimeFrame{Week, Month, ThreeMonth, SixMonth, Year, Overall}

	for _, tf := range frames {
		t.Run(tf.String(), func(t *testing.T) {
			got, err := ParseString(tf.String())
			if err != nil {
				t.Fatalf("ParseString(%q) error = %v", tf.String(), err)
			}
			if got != tf {
				t.Fatalf("ParseString(%q) = %v, want %v", tf.String(), got, tf)
			}
		})
	}
}
