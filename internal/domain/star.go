package domain

import (
	"fmt"
	"strings"
)

const (
	StarEmoji      = "⭐"
	MinStarLimit   = 1
	MaxStarLimit   = 20
	MaxStarIgnores = 50
)

// StarReactor is one member who added the star emoji.
type StarReactor struct {
	ID  string
	Bot bool
}

// IsStarEmoji reports whether a reaction is the starboard star.
func IsStarEmoji(emoji string) bool {
	emoji = strings.TrimSpace(emoji)
	emoji = strings.TrimSuffix(emoji, "\uFE0F")
	return emoji == StarEmoji
}

// CountStars counts unique member stars. Bots never count.
// The author's star counts only when selfStar is true.
func CountStars(authorID string, reactors []StarReactor, selfStar bool) int {
	seen := make(map[string]struct{}, len(reactors))
	count := 0
	for _, reactor := range reactors {
		if reactor.ID == "" {
			continue
		}
		if _, ok := seen[reactor.ID]; ok {
			continue
		}
		seen[reactor.ID] = struct{}{}
		if reactor.Bot {
			continue
		}
		if !selfStar && reactor.ID == authorID {
			continue
		}
		count++
	}
	return count
}

// ValidateStarboard checks the dashboard settings.
func ValidateStarboard(enabled bool, channelID string, limit int, ignored []string) error {
	if limit < MinStarLimit || limit > MaxStarLimit {
		return fmt.Errorf("the star limit must be between 1 and 20")
	}
	if enabled && !ValidSnowflake(channelID) {
		return fmt.Errorf("choose the starboard channel")
	}
	if channelID != "" && !ValidSnowflake(channelID) {
		return fmt.Errorf("starboard channel is not a valid id")
	}
	if len(ignored) > MaxStarIgnores {
		return fmt.Errorf("you can ignore up to 50 channels")
	}
	for _, id := range ignored {
		if !ValidSnowflake(id) {
			return fmt.Errorf("an ignored channel id is not valid")
		}
		if channelID != "" && id == channelID {
			return fmt.Errorf("the starboard channel cannot be ignored")
		}
	}
	return nil
}
