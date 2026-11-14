package domain

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// SetupChannel is a text channel the dashboard asks the bot to create.
type SetupChannel struct {
	Name    string
	Topic   string
	Private bool
}

// NormalizeChannelName turns a typed name into a Discord text channel name.
func NormalizeChannelName(name string) (string, error) {
	var b strings.Builder
	hyphen := false
	for _, r := range strings.TrimSpace(name) {
		switch {
		case unicode.IsLetter(r) || unicode.IsNumber(r):
			b.WriteRune(unicode.ToLower(r))
			hyphen = false
		case r == '_' || r == '-' || r == ' ':
			if b.Len() == 0 || hyphen {
				continue
			}
			if r == '_' {
				b.WriteByte('_')
			} else {
				b.WriteByte('-')
			}
			hyphen = true
		}
	}
	out := strings.Trim(b.String(), "-_")
	count := utf8.RuneCountInString(out)
	if count < 2 || count > 100 {
		return "", fmt.Errorf("channel names are 2 to 100 characters")
	}
	return out, nil
}

// SetupPurpose returns the topic and privacy for a dashboard channel.
func SetupPurpose(purpose string) (topic string, private bool, ok bool) {
	switch strings.ToLower(strings.TrimSpace(purpose)) {
	case "logging":
		return "Deleted messages, edits, joins, leaves, and bans.", true, true
	case "announcements":
		return "Posts from the mod team.", false, true
	case "roles":
		return "Click a reaction to get a role.", false, true
	case "starboard":
		return "Messages that earned enough stars.", false, true
	case "confession-review":
		return "Confessions waiting for a moderator.", true, true
	case "confessions":
		return "Anonymous confessions.", false, true
	case "ticket-panel":
		return "Open a support ticket here.", false, true
	case "ticket-log":
		return "Transcripts from closed support tickets.", true, true
	default:
		return "", false, false
	}
}

// ClassifySetup describes a channel or category the dashboard can create.
func ClassifySetup(purpose string) (kind, topic string, private, ok bool) {
	if strings.EqualFold(strings.TrimSpace(purpose), "ticket-category") {
		return "category", "", false, true
	}
	topic, private, ok = SetupPurpose(purpose)
	if !ok {
		return "", "", false, false
	}
	return "text", topic, private, true
}
