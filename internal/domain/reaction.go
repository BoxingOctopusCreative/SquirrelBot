package domain

import (
	"strings"
	"unicode/utf8"
)

const MaxReactionRoles = 20

// ReactionBinding is one emoji on a reaction-role message.
type ReactionBinding struct {
	Emoji  string
	RoleID string
}

// ReactionLine is one emoji and the role name shown in the roles channel.
type ReactionLine struct {
	Emoji    string
	RoleName string
}

// ApplyReaction returns the role for an emoji.
// grant is true when the member added the reaction.
// ok is false when the emoji is not bound.
func ApplyReaction(bindings []ReactionBinding, emoji string, added bool) (roleID string, grant bool, ok bool) {
	for _, binding := range bindings {
		if binding.Emoji == emoji {
			return binding.RoleID, added, true
		}
	}
	return "", false, false
}

// ReactionPanelText is the message SquirrelBot posts in the roles channel.
func ReactionPanelText(lines []ReactionLine) string {
	var b strings.Builder
	b.WriteString("Click a reaction to get the role. Remove the reaction to drop it.")
	for _, line := range lines {
		name := strings.TrimSpace(strings.ReplaceAll(line.RoleName, "\n", " "))
		name = strings.ReplaceAll(name, "@", "@\u200b")
		if name == "" {
			name = "role"
		}
		b.WriteString("\n")
		b.WriteString(DisplayEmoji(line.Emoji))
		b.WriteString(" ")
		b.WriteString(name)
	}
	text := b.String()
	if utf8.RuneCountInString(text) > 2000 {
		text = string([]rune(text)[:2000])
	}
	return text
}

// DisplayEmoji turns a stored custom emoji (name:id) into Discord message text.
func DisplayEmoji(emoji string) string {
	name, id, ok := strings.Cut(emoji, ":")
	if !ok || name == "" || id == "" {
		return emoji
	}
	return "<:" + name + ":" + id + ">"
}
