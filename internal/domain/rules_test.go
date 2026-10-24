package domain

import (
	"testing"
	"time"
)

func TestDetectAutomod(t *testing.T) {
	cfg := AutomodConfig{
		Enabled:       true,
		BlockInvites:  true,
		MentionLimit:  3,
		CapsPercent:   70,
		CapsMinLength: 8,
		BadWords:      []string{"slur"},
	}
	if _, ok := DetectAutomod("join discord.gg/acorns", 0, 0, false, cfg); !ok {
		t.Fatal("expected invite hit")
	}
	if _, ok := DetectAutomod("see discord.com/channels/1/2", 0, 0, false, cfg); ok {
		t.Fatal("server links are not invites")
	}
	if _, ok := DetectAutomod("hello @a @b @c @d", 4, 0, false, cfg); !ok {
		t.Fatal("expected mention hit")
	}
	if _, ok := DetectAutomod("THIS IS ALL SHOUTING NOW", 0, 0, false, cfg); !ok {
		t.Fatal("expected caps hit")
	}
	if _, ok := DetectAutomod("a slur in the middle", 0, 0, false, cfg); !ok {
		t.Fatal("expected word hit")
	}
	if _, ok := DetectAutomod("slurring is a different word", 0, 0, false, cfg); ok {
		t.Fatal("blocked words should match whole words")
	}
	cfg.Enabled = false
	if _, ok := DetectAutomod("discord.gg/acorns", 0, 0, false, cfg); ok {
		t.Fatal("disabled automod should not hit")
	}
}

func TestValidateGate(t *testing.T) {
	if err := ValidateGate(true, 24*time.Hour, time.Hour); err != nil {
		t.Fatal(err)
	}
	if err := ValidateGate(true, 24*time.Hour, 24*time.Hour); err == nil {
		t.Fatal("warning must be shorter than the deadline")
	}
	if err := ValidateGate(true, time.Minute, 0); err == nil {
		t.Fatal("deadline is too short")
	}
}

func TestValidateEscalation(t *testing.T) {
	if err := ValidateEscalation(2*time.Hour, 24*time.Hour); err != nil {
		t.Fatal(err)
	}
	if err := ValidateEscalation(48*time.Hour, 24*time.Hour); err == nil {
		t.Fatal("expected the role cap to apply")
	}
	if err := ValidateEscalation(time.Minute, 0); err == nil {
		t.Fatal("expected the minimum duration")
	}
}

func TestTopRolePosition(t *testing.T) {
	positions := map[string]int{"a": 1, "b": 8, "c": 3}
	if got := TopRolePosition(positions, []string{"a", "c"}); got != 3 {
		t.Fatalf("got %d", got)
	}
	if got := TopRolePosition(positions, []string{"b"}); got != 8 {
		t.Fatalf("got %d", got)
	}
}

func TestPermissions(t *testing.T) {
	if !CanManageGuild(PermAdministrator) {
		t.Fatal("administrators can manage the server")
	}
	if CanManageGuild(PermKickMembers) {
		t.Fatal("kick alone is not manage server")
	}
	if !HasPermission(PermManageMessages, PermManageMessages) {
		t.Fatal("expected exact permission")
	}
}
