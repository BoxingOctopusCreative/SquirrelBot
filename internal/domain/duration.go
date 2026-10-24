package domain

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

var durationPattern = regexp.MustCompile(`(?i)^\s*(?:(\d+)\s*w\s*)?(?:(\d+)\s*d\s*)?(?:(\d+)\s*h\s*)?(?:(\d+)\s*m\s*)?(?:(\d+)\s*s\s*)?$`)

const maxParsedDuration = 366 * 24 * time.Hour

// ParseDuration reads durations such as 30m, 12h, 2d, 1w, and 1d12h.
func ParseDuration(raw string) (time.Duration, error) {
	match := durationPattern.FindStringSubmatch(raw)
	if match == nil {
		return 0, fmt.Errorf("use a duration like 30m, 12h, 2d, or 1w")
	}
	var total time.Duration
	scales := []time.Duration{
		7 * 24 * time.Hour,
		24 * time.Hour,
		time.Hour,
		time.Minute,
		time.Second,
	}
	for i, scale := range scales {
		if match[i+1] == "" {
			continue
		}
		n, err := strconv.Atoi(match[i+1])
		if err != nil {
			return 0, fmt.Errorf("use a duration like 30m, 12h, 2d, or 1w")
		}
		total += time.Duration(n) * scale
	}
	if total <= 0 {
		return 0, fmt.Errorf("duration must be longer than 0")
	}
	if total > maxParsedDuration {
		return 0, fmt.Errorf("duration cannot be longer than 366 days")
	}
	return total, nil
}

// FormatDuration renders a duration as weeks, days, hours, and minutes.
func FormatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	d = d.Round(time.Minute)
	weeks := d / (7 * 24 * time.Hour)
	d -= weeks * 7 * 24 * time.Hour
	days := d / (24 * time.Hour)
	d -= days * 24 * time.Hour
	hours := d / time.Hour
	d -= hours * time.Hour
	mins := d / time.Minute
	var parts []string
	if weeks > 0 {
		parts = append(parts, fmt.Sprintf("%dw", weeks))
	}
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if mins > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%dm", mins))
	}
	out := ""
	for _, p := range parts {
		out += p
	}
	return out
}
