package domain

import "testing"

func TestStarEmoji(t *testing.T) {
	emoji, err := CleanStarEmoji("  ")
	if err != nil || emoji != StarEmoji {
		t.Fatalf("blank: %q %v", emoji, err)
	}
	emoji, err = CleanStarEmoji("⭐\uFE0F")
	if err != nil || emoji != StarEmoji {
		t.Fatalf("variation: %q %v", emoji, err)
	}
	emoji, err = CleanStarEmoji("canopy:123456789012345678")
	if err != nil || emoji != "canopy:123456789012345678" {
		t.Fatalf("custom: %q %v", emoji, err)
	}
	if _, err := CleanStarEmoji("not an emoji"); err == nil {
		t.Fatal("words should be rejected")
	}
	if _, err := CleanStarEmoji("bad:name"); err == nil {
		t.Fatal("custom emoji needs a snowflake")
	}
	if !MatchEmoji(StarEmoji, "⭐\uFE0F") {
		t.Fatal("star should match with a variation selector")
	}
	if !MatchEmoji("canopy:123456789012345678", "canopy:123456789012345678") {
		t.Fatal("custom emoji should match")
	}
	if MatchEmoji(StarEmoji, "🔥") {
		t.Fatal("other emoji")
	}
}

func TestCountStars(t *testing.T) {
	reactors := []StarReactor{
		{ID: "author"},
		{ID: "author"},
		{ID: "one"},
		{ID: "two"},
		{ID: "bot", Bot: true},
	}
	if got := CountStars("author", reactors, false); got != 2 {
		t.Fatalf("without self star: %d", got)
	}
	if got := CountStars("author", reactors, true); got != 3 {
		t.Fatalf("with self star: %d", got)
	}
}

func TestValidateStarboard(t *testing.T) {
	channel := "123456789012345678"
	if err := ValidateStarboard(true, channel, StarEmoji, 3, nil); err != nil {
		t.Fatal(err)
	}
	if err := ValidateStarboard(true, "", StarEmoji, 3, nil); err == nil {
		t.Fatal("enabled board needs a channel")
	}
	if err := ValidateStarboard(true, channel, StarEmoji, 0, nil); err == nil {
		t.Fatal("limit too small")
	}
	if err := ValidateStarboard(true, channel, "nope", 3, nil); err == nil {
		t.Fatal("emoji should be rejected")
	}
	if err := ValidateStarboard(true, channel, StarEmoji, 3, []string{channel}); err == nil {
		t.Fatal("cannot ignore the starboard channel")
	}
}
