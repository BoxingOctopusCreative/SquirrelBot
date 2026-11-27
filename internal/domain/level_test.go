package domain

import (
	"strings"
	"testing"
	"time"
)

func TestMee6Curve(t *testing.T) {
	sum := 0
	for level := 0; level < 40; level++ {
		if TotalXP(level) != sum {
			t.Fatalf("level %d: got %d want %d", level, TotalXP(level), sum)
		}
		sum += XPToNext(level)
	}
	if TotalXP(1) != 100 || TotalXP(2) != 255 || TotalXP(3) != 475 || TotalXP(5) != 1150 || TotalXP(10) != 4675 {
		t.Fatalf("known totals: 1=%d 2=%d 3=%d 5=%d 10=%d", TotalXP(1), TotalXP(2), TotalXP(3), TotalXP(5), TotalXP(10))
	}
	if LevelForXP(0) != 0 || LevelForXP(99) != 0 || LevelForXP(100) != 1 || LevelForXP(254) != 1 || LevelForXP(255) != 2 {
		t.Fatal("level lookup")
	}
	if LevelForXP(TotalXP(MaxLevel)) != MaxLevel {
		t.Fatal("cap")
	}
	level, into, span := Progress(180)
	if level != 1 || into != 80 || span != 155 {
		t.Fatalf("progress %d %d %d", level, into, span)
	}
	if XPBar(80, 160, 8) != "####----" {
		t.Fatalf("bar %q", XPBar(80, 160, 8))
	}
	if _, into, span = Progress(MaxStoredXP()); into != span {
		t.Fatal("the cap should fill the bar")
	}
}

func TestRollAndRate(t *testing.T) {
	if RollXP(15, 15, func(int) int { return 99 }) != 15 {
		t.Fatal("fixed roll")
	}
	if RollXP(15, 25, func(n int) int {
		if n != 11 {
			t.Fatalf("span %d", n)
		}
		return 10
	}) != 25 {
		t.Fatal("top of range")
	}
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var times []time.Time
	var ok bool
	for i := 0; i < 2; i++ {
		times, ok = AllowGrant(times, now.Add(time.Duration(i)*time.Second), 2, 60)
		if !ok {
			t.Fatal("first grants should fit")
		}
	}
	if _, ok = AllowGrant(times, now.Add(2*time.Second), 2, 60); ok {
		t.Fatal("third grant inside the window")
	}
	if _, ok = AllowGrant(times, now.Add(61*time.Second), 2, 60); !ok {
		t.Fatal("grant after the window")
	}
}

func TestRewardRolesAndAnnounce(t *testing.T) {
	rewards := []LevelReward{{Level: 2, RoleID: "a"}, {Level: 5, RoleID: "b"}, {Level: 9, RoleID: "a"}}
	kept := KeptRewardRoles(rewards, 9, false)
	if strings.Join(kept, ",") != "a,b" {
		t.Fatalf("stack %v", kept)
	}
	kept = KeptRewardRoles(rewards, 9, true)
	if strings.Join(kept, ",") != "a" {
		t.Fatalf("replace %v", kept)
	}
	kept = KeptRewardRoles(rewards, 4, true)
	if strings.Join(kept, ",") != "a" {
		t.Fatalf("early replace %v", kept)
	}
	if ShouldAnnounce(0, 1, 0, 0, false, nil) != true {
		t.Fatal("first level")
	}
	if ShouldAnnounce(4, 5, 15, 0, false, nil) {
		t.Fatal("below the limit")
	}
	if ShouldAnnounce(15, 16, 15, 0, false, nil) != true {
		t.Fatal("above the limit")
	}
	if ShouldAnnounce(3, 4, 0, 5, false, nil) {
		t.Fatal("not a multiple")
	}
	if ShouldAnnounce(4, 6, 0, 5, false, nil) != true {
		t.Fatal("crossed a multiple")
	}
	if ShouldAnnounce(3, 4, 0, 0, true, rewards) {
		t.Fatal("no reward on the way")
	}
	if ShouldAnnounce(1, 3, 0, 0, true, rewards) != true {
		t.Fatal("passed the level 2 reward")
	}
}

func TestValidateLevels(t *testing.T) {
	guild := "111111111111111111"
	role := "222222222222222222"
	channel := "333333333333333333"
	cfg := LevelConfig{
		XPMin: 15, XPMax: 25, RateCount: 1, RateSeconds: 60, Announce: AnnounceCurrent,
		Rewards: []LevelReward{{Level: 2, RoleID: role}},
	}
	cleaned, err := ValidateLevels(guild, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if cleaned.Announce != AnnounceCurrent {
		t.Fatal(cleaned.Announce)
	}
	cfg.XPMin = 10
	if _, err := ValidateLevels(guild, cfg); err == nil {
		t.Fatal("xp below 15")
	}
	cfg.XPMin = 20
	cfg.XPMax = 15
	if _, err := ValidateLevels(guild, cfg); err == nil {
		t.Fatal("min above max")
	}
	cfg.XPMin, cfg.XPMax = 15, 25
	cfg.Announce = AnnounceChannel
	if _, err := ValidateLevels(guild, cfg); err == nil {
		t.Fatal("missing channel")
	}
	cfg.AnnounceChannelID = channel
	cfg.Message = "{{ nope }}"
	if _, err := ValidateLevels(guild, cfg); err == nil {
		t.Fatal("unknown variable")
	}
	cfg.Message = "{{ mention }} is level {{ level }}"
	if _, err := ValidateLevels(guild, cfg); err != nil {
		t.Fatal(err)
	}
	cfg.Rewards = []LevelReward{{Level: 2, RoleID: role}, {Level: 2, RoleID: role}}
	if _, err := ValidateLevels(guild, cfg); err == nil {
		t.Fatal("duplicate level")
	}
	text, err := RenderLevelMessage("", LevelView{Mention: "<@1>", User: "Acorn", Server: "Canopy", Level: 3, XP: 400})
	if err != nil || !strings.Contains(text, "<@1>") || !strings.Contains(text, "3") || !strings.Contains(text, "Canopy") {
		t.Fatalf("default message %q %v", text, err)
	}
}

func TestCleanLevelName(t *testing.T) {
	if CleanLevelName("  Acorn   Fern \n") != "Acorn Fern" {
		t.Fatal(CleanLevelName("  Acorn   Fern \n"))
	}
	long := strings.Repeat("a", 90)
	if utfLen(CleanLevelName(long)) != 80 {
		t.Fatal("trim")
	}
}

func utfLen(s string) int {
	return len([]rune(s))
}
