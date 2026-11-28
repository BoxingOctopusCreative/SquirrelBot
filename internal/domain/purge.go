package domain

import (
	"errors"
	"time"
)

const (
	PurgeMinMinutes = 15
	PurgeMaxMinutes = 43200
	// PurgeMaxAge stays inside Discord's 14 day bulk-delete limit.
	// Manual /purge uses this same window.
	PurgeMaxAge = 13 * 24 * time.Hour
)

// Discord channel types the dashboard already lists.
const (
	purgeGuildText = 0
	purgeGuildNews = 5
)

var purgePresets = []struct {
	Key     string
	Seconds int
}{
	{"1h", 3600},
	{"4h", 4 * 3600},
	{"12h", 12 * 3600},
	{"24h", 24 * 3600},
	{"7d", 7 * 24 * 3600},
}

// PurgeInterval is one saved length. An empty Preset is a custom duration.
type PurgeInterval struct {
	Preset  string
	Seconds int
}

// TooOldToBulkDelete reports whether Discord would reject this message in a bulk delete.
func TooOldToBulkDelete(sent, now time.Time) bool {
	return sent.Before(now.Add(-PurgeMaxAge))
}

// PurgeChannelAllowed is true for a guild text channel or an announcement channel.
func PurgeChannelAllowed(channelType int) bool {
	return channelType == purgeGuildText || channelType == purgeGuildNews
}

// ParsePurgeInterval accepts either a preset key or a custom minute count, not both.
func ParsePurgeInterval(preset string, minutes *int) (PurgeInterval, error) {
	if preset != "" && minutes != nil {
		return PurgeInterval{}, errors.New("Choose a common interval or a custom length.")
	}
	if preset == "" && minutes == nil {
		return PurgeInterval{}, errors.New("Choose a common interval or a custom length.")
	}
	if preset != "" {
		for _, item := range purgePresets {
			if item.Key == preset {
				return PurgeInterval{Preset: item.Key, Seconds: item.Seconds}, nil
			}
		}
		return PurgeInterval{}, errors.New("Choose a common interval or a custom length.")
	}
	if *minutes < PurgeMinMinutes || *minutes > PurgeMaxMinutes {
		return PurgeInterval{}, errors.New("Custom intervals are from 15 minutes to 30 days.")
	}
	return PurgeInterval{Seconds: *minutes * 60}, nil
}
