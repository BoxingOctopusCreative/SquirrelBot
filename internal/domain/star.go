package domain

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	StarEmoji      = "⭐"
	MinStarLimit   = 1
	MaxStarLimit   = 20
	MaxStarIgnores = 50
)

var customEmojiName = regexp.MustCompile(`^[A-Za-z0-9_]{2,32}$`)

// StarReactor is one member who added the star emoji.
type StarReactor struct {
	ID  string
	Bot bool
}

// CleanStarEmoji accepts one unicode emoji or a server emoji as name:id.
// A blank value uses the default star.
func CleanStarEmoji(emoji string) (string, error) {
	emoji = strings.TrimSpace(emoji)
	if emoji == "" {
		return StarEmoji, nil
	}
	if strings.ContainsAny(emoji, " \n\t") || utf8.RuneCountInString(emoji) > 32 {
		return "", fmt.Errorf("use one emoji")
	}
	if name, id, ok := strings.Cut(emoji, ":"); ok {
		if !customEmojiName.MatchString(name) || !ValidSnowflake(id) {
			return "", fmt.Errorf("a server emoji looks like name:id")
		}
		return name + ":" + id, nil
	}
	if !unicodeEmoji(emoji) {
		return "", fmt.Errorf("use one emoji")
	}
	return strings.TrimSuffix(emoji, "\uFE0F"), nil
}

// MatchEmoji reports whether a reaction is the configured starboard emoji.
func MatchEmoji(want, got string) bool {
	want = foldEmoji(want)
	got = foldEmoji(got)
	return want != "" && want == got
}

func foldEmoji(emoji string) string {
	emoji = strings.TrimSpace(emoji)
	if name, id, ok := strings.Cut(emoji, ":"); ok && name != "" && id != "" {
		return name + ":" + id
	}
	return strings.TrimSuffix(emoji, "\uFE0F")
}

func unicodeEmoji(emoji string) bool {
	for _, r := range emoji {
		if r > 0x7f || r == '#' || r == '*' {
			return true
		}
	}
	return false
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
func ValidateStarboard(enabled bool, channelID, emoji string, limit int, ignored []string) error {
	if _, err := CleanStarEmoji(emoji); err != nil {
		return err
	}
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
