package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	MaxLevel          = 100
	MinMessageXP      = 15
	MaxMessageXP      = 100
	DefaultXPMin      = 15
	DefaultXPMax      = 25
	MinRateCount      = 1
	MaxRateCount      = 20
	MinRateSeconds    = 10
	MaxRateSeconds    = 3600
	DefaultRateCount  = 1
	DefaultRateWindow = 60
	MaxLevelMessage   = 500
	MaxLevelRewards   = 40
	MaxLevelIgnores   = 50
	LevelPageSize     = 10
	MaxLevelPage      = 20

	AnnounceOff     = "off"
	AnnounceCurrent = "current"
	AnnounceChannel = "channel"
	AnnounceDM      = "dm"

	// DefaultLevelMessage is used when a server has not written its own.
	// mention is a Discord user mention. user is the display name.
	DefaultLevelMessage = "{{ mention }} reached level {{ level }} in {{ server }}."
)

// LevelReward is a role granted at a level.
type LevelReward struct {
	Level  int
	RoleID string
}

// LevelConfig is the dashboard and command settings for one server.
type LevelConfig struct {
	Enabled           bool
	XPMin             int
	XPMax             int
	RateCount         int
	RateSeconds       int
	VoiceXP           bool
	Announce          string
	AnnounceChannelID string
	Message           string
	AnnounceMinLevel  int
	AnnounceMultiple  int
	RewardOnly        bool
	ReplaceRewards    bool
	IgnoredChannelIDs []string
	IgnoredRoleIDs    []string
	Rewards           []LevelReward
}

// LevelView fills a level-up message.
type LevelView struct {
	Mention string
	User    string
	Server  string
	Level   int
	XP      int
}

// XPToNext is the XP required to advance from level to level+1.
// Carl-bot uses the Mee6 curve: 5n^2 + 50n + 100, where n is the current level.
func XPToNext(level int) int {
	if level < 0 {
		level = 0
	}
	return 5*level*level + 50*level + 100
}

// TotalXP is the XP required to reach level. Level 0 is 0. Level 1 is 100.
func TotalXP(level int) int {
	if level <= 0 {
		return 0
	}
	if level > MaxLevel {
		level = MaxLevel
	}
	sumSquares := (level - 1) * level * (2*level - 1) / 6
	return 5*sumSquares + 25*level*(level-1) + 100*level
}

// MaxStoredXP is the highest XP a member can hold. That fills the bar at level 100.
func MaxStoredXP() int {
	return TotalXP(MaxLevel) + XPToNext(MaxLevel) - 1
}

// ClampXP limits a score to the stored range.
func ClampXP(xp int) int {
	if xp < 0 {
		return 0
	}
	if xp > MaxStoredXP() {
		return MaxStoredXP()
	}
	return xp
}

// LevelForXP returns the level for an XP total, from 0 through MaxLevel.
func LevelForXP(xp int) int {
	if xp <= 0 {
		return 0
	}
	level := 0
	for next := 1; next <= MaxLevel; next++ {
		if xp < TotalXP(next) {
			break
		}
		level = next
	}
	return level
}

// Progress reports the current level and the XP into the next level.
func Progress(xp int) (level, into, span int) {
	xp = ClampXP(xp)
	level = LevelForXP(xp)
	into = xp - TotalXP(level)
	if into < 0 {
		into = 0
	}
	span = XPToNext(level)
	if level >= MaxLevel && xp >= MaxStoredXP() {
		into = span
	}
	return level, into, span
}

// XPBar draws a text bar. into and span are the current level's progress.
func XPBar(into, span, width int) string {
	if width < 1 {
		width = 12
	}
	if span < 1 {
		span = 1
	}
	if into < 0 {
		into = 0
	}
	if into > span {
		into = span
	}
	filled := into * width / span
	if into > 0 && filled == 0 {
		filled = 1
	}
	if filled > width {
		filled = width
	}
	return strings.Repeat("#", filled) + strings.Repeat("-", width-filled)
}

// RollXP picks a gain from min through max. roll returns a value in [0, n).
func RollXP(min, max int, roll func(n int) int) int {
	if max < min {
		max = min
	}
	if max == min || roll == nil {
		return min
	}
	span := max - min + 1
	n := roll(span)
	if n < 0 {
		n = 0
	}
	if n >= span {
		n = span - 1
	}
	return min + n
}

// AllowGrant reports whether another XP grant fits in the window.
// times are earlier grants. On success the returned slice includes now.
// On failure the returned slice is the grants that are still inside the window.
func AllowGrant(times []time.Time, now time.Time, count, seconds int) ([]time.Time, bool) {
	if count < 1 {
		count = 1
	}
	if seconds < 1 {
		return nil, true
	}
	cutoff := now.Add(-time.Duration(seconds) * time.Second)
	kept := make([]time.Time, 0, len(times)+1)
	for _, at := range times {
		if at.After(cutoff) {
			kept = append(kept, at)
		}
	}
	if len(kept) >= count {
		return kept, false
	}
	return append(kept, now), true
}

// KeptRewardRoles returns the reward roles a member should hold at level.
// When replace is false they keep every reward they have reached.
// When replace is true they keep only the highest one.
func KeptRewardRoles(rewards []LevelReward, level int, replace bool) []string {
	earned := make([]LevelReward, 0, len(rewards))
	for _, reward := range rewards {
		if reward.Level > 0 && reward.Level <= level && reward.RoleID != "" {
			earned = append(earned, reward)
		}
	}
	sort.Slice(earned, func(i, j int) bool {
		if earned[i].Level == earned[j].Level {
			return earned[i].RoleID < earned[j].RoleID
		}
		return earned[i].Level < earned[j].Level
	})
	if len(earned) == 0 {
		return nil
	}
	if replace {
		return []string{earned[len(earned)-1].RoleID}
	}
	seen := make(map[string]struct{}, len(earned))
	out := make([]string, 0, len(earned))
	for _, reward := range earned {
		if _, ok := seen[reward.RoleID]; ok {
			continue
		}
		seen[reward.RoleID] = struct{}{}
		out = append(out, reward.RoleID)
	}
	return out
}

// AllRewardRoles returns each reward role once, in first-seen order.
func AllRewardRoles(rewards []LevelReward) []string {
	seen := make(map[string]struct{}, len(rewards))
	out := make([]string, 0, len(rewards))
	for _, reward := range rewards {
		if reward.RoleID == "" {
			continue
		}
		if _, ok := seen[reward.RoleID]; ok {
			continue
		}
		seen[reward.RoleID] = struct{}{}
		out = append(out, reward.RoleID)
	}
	return out
}

// ShouldAnnounce reports whether a level change should be posted.
// minLevel announces only levels above that number. 0 announces every level.
// multiple announces a level only when it divides evenly, when multiple is 2 or more.
// rewardOnly announces when a crossed level has a role reward.
func ShouldAnnounce(oldLevel, newLevel, minLevel, multiple int, rewardOnly bool, rewards []LevelReward) bool {
	if newLevel <= oldLevel {
		return false
	}
	for level := oldLevel + 1; level <= newLevel; level++ {
		if level <= minLevel {
			continue
		}
		if multiple >= 2 && level%multiple != 0 {
			continue
		}
		if rewardOnly && !hasRewardAt(rewards, level) {
			continue
		}
		return true
	}
	return false
}

func hasRewardAt(rewards []LevelReward, level int) bool {
	for _, reward := range rewards {
		if reward.Level == level {
			return true
		}
	}
	return false
}

// CleanLevelName trims a display name stored on the leaderboard.
func CleanLevelName(name string) string {
	name = strings.Join(strings.Fields(name), " ")
	if utf8.RuneCountInString(name) <= 80 {
		return name
	}
	return string([]rune(name)[:80])
}

// ValidateLevels checks dashboard settings and returns the normalized config.
func ValidateLevels(guildID string, cfg LevelConfig) (LevelConfig, error) {
	if cfg.XPMin < MinMessageXP || cfg.XPMin > MaxMessageXP || cfg.XPMax < MinMessageXP || cfg.XPMax > MaxMessageXP {
		return LevelConfig{}, fmt.Errorf("message XP must be from %d to %d", MinMessageXP, MaxMessageXP)
	}
	if cfg.XPMin > cfg.XPMax {
		return LevelConfig{}, fmt.Errorf("the minimum XP cannot be higher than the maximum")
	}
	if cfg.RateCount < MinRateCount || cfg.RateCount > MaxRateCount {
		return LevelConfig{}, fmt.Errorf("the XP rate must be %d to %d messages", MinRateCount, MaxRateCount)
	}
	if cfg.RateSeconds < MinRateSeconds || cfg.RateSeconds > MaxRateSeconds {
		return LevelConfig{}, fmt.Errorf("the XP window must be %d to %d seconds", MinRateSeconds, MaxRateSeconds)
	}
	cfg.Announce = strings.ToLower(strings.TrimSpace(cfg.Announce))
	if cfg.Announce == "" {
		cfg.Announce = AnnounceCurrent
	}
	switch cfg.Announce {
	case AnnounceOff, AnnounceCurrent, AnnounceChannel, AnnounceDM:
	default:
		return LevelConfig{}, fmt.Errorf("choose where level ups are announced")
	}
	cfg.AnnounceChannelID = strings.TrimSpace(cfg.AnnounceChannelID)
	if cfg.AnnounceChannelID != "" && !ValidSnowflake(cfg.AnnounceChannelID) {
		return LevelConfig{}, fmt.Errorf("level-up channel is not a valid id")
	}
	if cfg.Announce == AnnounceChannel && cfg.AnnounceChannelID == "" {
		return LevelConfig{}, fmt.Errorf("choose the level-up channel")
	}
	message, err := cleanLevelMessage(cfg.Message)
	if err != nil {
		return LevelConfig{}, err
	}
	cfg.Message = message
	if cfg.AnnounceMinLevel < 0 || cfg.AnnounceMinLevel > MaxLevel {
		return LevelConfig{}, fmt.Errorf("the announce limit must be from 0 to %d", MaxLevel)
	}
	if cfg.AnnounceMultiple < 0 || cfg.AnnounceMultiple > MaxLevel {
		return LevelConfig{}, fmt.Errorf("the announce multiple must be from 0 to %d", MaxLevel)
	}
	if len(cfg.IgnoredChannelIDs) > MaxLevelIgnores || len(cfg.IgnoredRoleIDs) > MaxLevelIgnores {
		return LevelConfig{}, fmt.Errorf("keep ignored channels and roles to %d each", MaxLevelIgnores)
	}
	for _, id := range cfg.IgnoredChannelIDs {
		if !ValidSnowflake(id) {
			return LevelConfig{}, fmt.Errorf("an ignored channel is not a valid id")
		}
	}
	for _, id := range cfg.IgnoredRoleIDs {
		if !ValidSnowflake(id) {
			return LevelConfig{}, fmt.Errorf("an ignored role is not a valid id")
		}
	}
	if len(cfg.Rewards) > MaxLevelRewards {
		return LevelConfig{}, fmt.Errorf("keep role rewards to %d", MaxLevelRewards)
	}
	seen := make(map[int]struct{}, len(cfg.Rewards))
	for _, reward := range cfg.Rewards {
		if reward.Level < 1 || reward.Level > MaxLevel {
			return LevelConfig{}, fmt.Errorf("a reward level must be from 1 to %d", MaxLevel)
		}
		if _, ok := seen[reward.Level]; ok {
			return LevelConfig{}, fmt.Errorf("level %d already has a reward", reward.Level)
		}
		seen[reward.Level] = struct{}{}
		if !ValidSnowflake(reward.RoleID) || reward.RoleID == guildID {
			return LevelConfig{}, fmt.Errorf("a reward role is not a valid id")
		}
	}
	return cfg, nil
}

func cleanLevelMessage(message string) (string, error) {
	message = strings.TrimSpace(message)
	if message == "" {
		return "", nil
	}
	if utf8.RuneCountInString(message) > MaxLevelMessage {
		return "", fmt.Errorf("level-up message must be at most %d characters", MaxLevelMessage)
	}
	if _, err := RenderLevelMessage(message, LevelView{
		Mention: "@member", User: "member", Server: "Server", Level: 1, XP: 100,
	}); err != nil {
		return "", err
	}
	return message, nil
}

// RenderLevelMessage fills a level-up message. An empty template uses the default.
func RenderLevelMessage(message string, view LevelView) (string, error) {
	message = strings.TrimSpace(message)
	if message == "" {
		message = DefaultLevelMessage
	}
	out, err := renderTemplate(message, map[string]any{
		"mention": view.Mention,
		"user":    view.User,
		"server":  view.Server,
		"level":   view.Level,
		"xp":      view.XP,
	})
	if err != nil {
		return "", fmt.Errorf("level-up message: %s", liquidMessage(err))
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return "", fmt.Errorf("level-up message is empty")
	}
	if utf8.RuneCountInString(out) > 1500 {
		out = string([]rune(out)[:1500])
	}
	return out, nil
}
