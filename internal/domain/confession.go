package domain

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	MaxConfession         = 1500
	MaxPendingConfessions = 3
	ConfessionPending     = "pending"
	ConfessionApproved    = "approved"
	ConfessionRejected    = "rejected"
)

var confessionBroadcast = regexp.MustCompile(`(?i)@(everyone|here)`)

// CleanConfession trims a confession and blocks pings that would highlight the whole server.
func CleanConfession(text string) (string, error) {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	var b strings.Builder
	for _, r := range text {
		switch r {
		case '\n', '\t':
			b.WriteRune(r)
		case '\u200b', '\u200c', '\u200d', '\u2060', '\ufeff':
			continue
		default:
			if unicode.IsControl(r) {
				continue
			}
			b.WriteRune(r)
		}
	}
	out := strings.TrimSpace(b.String())
	out = confessionBroadcast.ReplaceAllString(out, "@ $1")
	if out == "" {
		return "", fmt.Errorf("write a confession")
	}
	if utf8.RuneCountInString(out) > MaxConfession {
		return "", fmt.Errorf("a confession can be %d characters", MaxConfession)
	}
	return out, nil
}

// ValidateConfessionChannels checks the review channel and the public confessions channel.
func ValidateConfessionChannels(reviewID, publicID string) error {
	if !OptionalSnowflake(reviewID) {
		return fmt.Errorf("the review channel is not a valid id")
	}
	if !OptionalSnowflake(publicID) {
		return fmt.Errorf("the confessions channel is not a valid id")
	}
	if reviewID != "" && reviewID == publicID {
		return fmt.Errorf("use a different channel for review and for posted confessions")
	}
	return nil
}

// ConfessionCustomID is the button id for one confession decision.
func ConfessionCustomID(action string, id int64) string {
	return fmt.Sprintf("sb:confess:%s:%d", action, id)
}

// ParseConfessionAction reads an approve or reject button id.
func ParseConfessionAction(customID string) (action string, id int64, ok bool) {
	const prefix = "sb:confess:"
	if !strings.HasPrefix(customID, prefix) {
		return "", 0, false
	}
	action, raw, found := strings.Cut(customID[len(prefix):], ":")
	if !found || (action != "approve" && action != "reject") {
		return "", 0, false
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n < 1 || strings.TrimSpace(raw) != raw {
		return "", 0, false
	}
	return action, n, true
}
