package domain

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode"
)

const (
	PermKickMembers     int64 = 1 << 1
	PermBanMembers      int64 = 1 << 2
	PermAdministrator   int64 = 1 << 3
	PermManageChannels  int64 = 1 << 4
	PermManageGuild     int64 = 1 << 5
	PermManageMessages  int64 = 1 << 13
	PermManageRoles     int64 = 1 << 28
	PermModerateMembers int64 = 1 << 40
)

const (
	MinEscalation     = 5 * time.Minute
	MaxEscalation     = 30 * 24 * time.Hour
	MaxDiscordTimeout = 28 * 24 * time.Hour
	MinGateDeadline   = 5 * time.Minute
	MaxGateDeadline   = 30 * 24 * time.Hour
	MaxAnnounceLead   = 30 * 24 * time.Hour
	MaxAnnounceTitle  = 256
	MaxAnnounceBody   = 4000
	MaxReason         = 400
	MaxCustomResponse = 1500
	MaxCommandSource  = 8000
	MaxCommandRoles   = 10
	MaxWelcome        = 1500
	MaxBadWords       = 100
	MaxBadWordLen     = 40
)

var commandName = regexp.MustCompile(`^[a-z0-9_-]{1,32}$`)
var invitePattern = regexp.MustCompile(`(?i)(?:discord\.gg/|discord(?:app)?\.com/invite/)[a-z0-9-]+`)

func HasPermission(bits, perm int64) bool {
	if bits&PermAdministrator == PermAdministrator {
		return true
	}
	return perm != 0 && bits&perm == perm
}

func CanManageGuild(bits int64) bool {
	return HasPermission(bits, PermManageGuild)
}

func ValidSnowflake(id string) bool {
	if len(id) < 17 || len(id) > 20 {
		return false
	}
	for _, c := range id {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func OptionalSnowflake(id string) bool {
	return id == "" || ValidSnowflake(id)
}

func ValidPrefix(prefix string) bool {
	if len(prefix) < 1 || len(prefix) > 5 {
		return false
	}
	return !strings.ContainsAny(prefix, " \n\t/")
}

func ValidCommandName(name string) bool {
	return commandName.MatchString(name)
}

func CleanReason(reason string) (string, error) {
	reason = strings.TrimSpace(reason)
	reason = strings.ReplaceAll(reason, "\n", " ")
	if len(reason) > MaxReason {
		return "", fmt.Errorf("reason must be %d characters or fewer", MaxReason)
	}
	return reason, nil
}

// AuditReason prepares a reason for Discord's audit log header.
func AuditReason(reason string) string {
	reason = strings.TrimSpace(strings.ReplaceAll(reason, "\n", " "))
	if len(reason) > 500 {
		reason = reason[:500]
	}
	return url.PathEscape(reason)
}

func ValidateAnnouncement(title, body string, scheduledAt *time.Time, now time.Time) error {
	title = strings.TrimSpace(title)
	body = strings.TrimSpace(body)
	if title == "" || len(title) > MaxAnnounceTitle {
		return fmt.Errorf("title must be 1 to %d characters", MaxAnnounceTitle)
	}
	if body == "" || len(body) > MaxAnnounceBody {
		return fmt.Errorf("body must be 1 to %d characters", MaxAnnounceBody)
	}
	if scheduledAt != nil && scheduledAt.After(now.Add(MaxAnnounceLead)) {
		return fmt.Errorf("schedule announcements within 30 days")
	}
	return nil
}

func ValidateEscalation(length, maxAllowed time.Duration) error {
	if length < MinEscalation {
		return fmt.Errorf("admin firecall access must last at least 5 minutes")
	}
	if length > MaxEscalation {
		return fmt.Errorf("admin firecall access cannot last longer than 30 days")
	}
	if maxAllowed > 0 && length > maxAllowed {
		return fmt.Errorf("that role can be granted for %s at most", FormatDuration(maxAllowed))
	}
	return nil
}

func ValidateGate(enabled bool, deadline, warnBefore time.Duration) error {
	if !enabled && deadline == 0 {
		return nil
	}
	if deadline < MinGateDeadline || deadline > MaxGateDeadline {
		return fmt.Errorf("the deadline must be between 5 minutes and 30 days")
	}
	if warnBefore < 0 || warnBefore >= deadline {
		return fmt.Errorf("the warning must be shorter than the deadline")
	}
	return nil
}

func ValidateTimeout(length time.Duration) error {
	if length < time.Minute {
		return fmt.Errorf("timeouts must be at least 1 minute")
	}
	if length > MaxDiscordTimeout {
		return fmt.Errorf("Discord timeouts cannot be longer than 28 days")
	}
	return nil
}

// TopRolePosition returns the highest role position held by any of the given role ids.
func TopRolePosition(positions map[string]int, roleIDs []string) int {
	top := 0
	for _, id := range roleIDs {
		if positions[id] > top {
			top = positions[id]
		}
	}
	return top
}

// ApplyTemplate replaces the small set of placeholders used in welcome and gate messages.
func ApplyTemplate(template string, values map[string]string) string {
	pairs := make([]string, 0, len(values)*2)
	for key, value := range values {
		pairs = append(pairs, "{"+key+"}", value)
	}
	return strings.NewReplacer(pairs...).Replace(template)
}

type AutomodConfig struct {
	Enabled           bool
	BlockInvites      bool
	BlockSpam         bool
	MentionLimit      int
	CapsPercent       int
	CapsMinLength     int
	BadWords          []string
	IgnoredChannelIDs []string
	IgnoredRoleIDs    []string
}

type AutomodHit struct {
	Rule   string
	Detail string
}

// DetectAutomod reports the first rule a message breaks. Mods and ignored rooms are skipped by the caller.
func DetectAutomod(content string, userMentions, roleMentions int, everyone bool, cfg AutomodConfig) (AutomodHit, bool) {
	if !cfg.Enabled {
		return AutomodHit{}, false
	}
	if cfg.BlockInvites && invitePattern.MatchString(content) {
		return AutomodHit{Rule: "invites", Detail: "posted a Discord invite"}, true
	}
	if everyone || (cfg.MentionLimit > 0 && userMentions+roleMentions > cfg.MentionLimit) {
		return AutomodHit{Rule: "mentions", Detail: "mentioned too many people"}, true
	}
	if hit, ok := capsHit(content, cfg.CapsPercent, cfg.CapsMinLength); ok {
		return hit, true
	}
	if word, ok := badWord(content, cfg.BadWords); ok {
		return AutomodHit{Rule: "words", Detail: "matched a blocked word (" + word + ")"}, true
	}
	return AutomodHit{}, false
}

func capsHit(content string, percent, minLength int) (AutomodHit, bool) {
	if percent <= 0 || minLength <= 0 {
		return AutomodHit{}, false
	}
	letters := 0
	upper := 0
	for _, r := range content {
		if unicode.IsLetter(r) {
			letters++
			if unicode.IsUpper(r) {
				upper++
			}
		}
	}
	if letters < minLength {
		return AutomodHit{}, false
	}
	if upper*100/letters >= percent {
		return AutomodHit{Rule: "caps", Detail: "was mostly capital letters"}, true
	}
	return AutomodHit{}, false
}

func badWord(content string, words []string) (string, bool) {
	if len(words) == 0 {
		return "", false
	}
	lower := strings.ToLower(content)
	count := 0
	for _, word := range words {
		word = strings.ToLower(strings.TrimSpace(word))
		if word == "" || len(word) > MaxBadWordLen {
			continue
		}
		count++
		if count > MaxBadWords {
			break
		}
		pattern := `(?:^|[^\p{L}\p{N}])` + regexp.QuoteMeta(word) + `(?:$|[^\p{L}\p{N}])`
		if regexp.MustCompile(pattern).MatchString(lower) {
			return word, true
		}
	}
	return "", false
}

func ContainsID(ids []string, id string) bool {
	for _, candidate := range ids {
		if candidate == id {
			return true
		}
	}
	return false
}
