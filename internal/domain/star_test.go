package domain

import "testing"

func TestIsStarEmoji(t *testing.T) {
	if !IsStarEmoji("⭐") || !IsStarEmoji("⭐\uFE0F") {
		t.Fatal("star emoji should match")
	}
	if IsStarEmoji("🔥") {
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
	if err := ValidateStarboard(true, channel, 3, nil); err != nil {
		t.Fatal(err)
	}
	if err := ValidateStarboard(true, "", 3, nil); err == nil {
		t.Fatal("enabled board needs a channel")
	}
	if err := ValidateStarboard(true, channel, 0, nil); err == nil {
		t.Fatal("limit too small")
	}
	if err := ValidateStarboard(true, channel, 3, []string{channel}); err == nil {
		t.Fatal("cannot ignore the starboard channel")
	}
}
