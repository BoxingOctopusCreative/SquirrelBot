package discord

import "testing"

func TestStarMark(t *testing.T) {
	text, icon := starMark("⭐")
	if text != "⭐" || icon != "" {
		t.Fatalf("unicode %q %q", text, icon)
	}
	text, icon = starMark("canopy:123456789012345678")
	if text != ":canopy:" || icon != "https://cdn.discordapp.com/emojis/123456789012345678.png" {
		t.Fatalf("custom %q %q", text, icon)
	}
	if _, icon = starMark("bad:../x"); icon != "" {
		t.Fatal("icon should stay empty")
	}
}
