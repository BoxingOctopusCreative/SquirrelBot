package domain

import (
	"strings"
	"testing"
)

func TestApplyReaction(t *testing.T) {
	bindings := []ReactionBinding{
		{Emoji: "🍎", RoleID: "red"},
		{Emoji: "🍋", RoleID: "yellow"},
	}

	role, grant, ok := ApplyReaction(bindings, "🍎", true)
	if !ok || !grant || role != "red" {
		t.Fatalf("add got %s grant=%v ok=%v", role, grant, ok)
	}
	role, grant, ok = ApplyReaction(bindings, "🍎", false)
	if !ok || grant || role != "red" {
		t.Fatalf("remove got %s grant=%v ok=%v", role, grant, ok)
	}
	if _, _, ok := ApplyReaction(bindings, "🍌", true); ok {
		t.Fatal("unknown emoji should not match")
	}
}

func TestReactionPanelText(t *testing.T) {
	text := ReactionPanelText([]ReactionLine{
		{Emoji: "🍎", RoleName: "Member"},
		{Emoji: "canopy:123456789012345678", RoleName: "@everyone"},
	})
	if !strings.Contains(text, "🍎 Member") {
		t.Fatalf("unicode line missing: %s", text)
	}
	if !strings.Contains(text, "<:canopy:123456789012345678> @\u200beveryone") {
		t.Fatalf("custom emoji line: %s", text)
	}
}
