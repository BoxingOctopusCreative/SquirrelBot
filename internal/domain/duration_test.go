package domain

import (
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	cases := []struct {
		in   string
		want time.Duration
		ok   bool
	}{
		{"30m", 30 * time.Minute, true},
		{"12h", 12 * time.Hour, true},
		{"2d", 48 * time.Hour, true},
		{"1w", 7 * 24 * time.Hour, true},
		{"1d12h", 36 * time.Hour, true},
		{"1d 2h 15m", 26*time.Hour + 15*time.Minute, true},
		{"", 0, false},
		{"0s", 0, false},
		{"1x", 0, false},
		{"999d", 0, false},
		{"-1h", 0, false},
	}
	for _, tc := range cases {
		got, err := ParseDuration(tc.in)
		if tc.ok && err != nil {
			t.Fatalf("%q: %v", tc.in, err)
		}
		if !tc.ok && err == nil {
			t.Fatalf("%q: expected an error", tc.in)
		}
		if tc.ok && got != tc.want {
			t.Fatalf("%q: got %s want %s", tc.in, got, tc.want)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	if got := FormatDuration(90 * time.Minute); got != "1h30m" {
		t.Fatalf("got %s", got)
	}
	if got := FormatDuration(26 * time.Hour); got != "1d2h" {
		t.Fatalf("got %s", got)
	}
}
