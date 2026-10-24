package domain

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	AgeQuestion       = "What is your age?"
	MinVerifyAge      = 18
	MaxGateQuestions  = 4
	MaxGatePrompt     = 45
	ageKickReasonText = "Under the age requirement"
)

// AgeKickReason is the case reason for an underage answer. It does not include the answer.
func AgeKickReason() string {
	return ageKickReasonText
}

// GateDeadline is the moment a new member must finish the application.
func GateDeadline(joined time.Time, deadline time.Duration) time.Time {
	return joined.Add(deadline)
}

// GateShouldWarn is true once the warning window has opened and no warning has been sent.
func GateShouldWarn(now, deadline time.Time, warnBefore time.Duration, warned bool) bool {
	if warned || warnBefore <= 0 {
		return false
	}
	if !now.Before(deadline) {
		return false
	}
	return !now.Before(deadline.Add(-warnBefore))
}

// GateShouldKick is true after the deadline while the member is still pending.
func GateShouldKick(now, deadline time.Time) bool {
	return !now.Before(deadline)
}

// GatePanelText is the application message posted for new members.
func GatePanelText() string {
	return "Answer the questions to stay in the server. You must be 18 or older. If the time runs out first, you will be removed."
}

// CleanGateQuestions checks the admin-written questions that follow the age question.
func CleanGateQuestions(prompts []string) ([]string, error) {
	if len(prompts) > MaxGateQuestions {
		return nil, fmt.Errorf("you can add up to 4 questions besides age")
	}
	out := make([]string, 0, len(prompts))
	for _, prompt := range prompts {
		prompt = strings.TrimSpace(prompt)
		if prompt == "" {
			return nil, fmt.Errorf("every question needs text")
		}
		if utf8.RuneCountInString(prompt) > MaxGatePrompt {
			return nil, fmt.Errorf("questions can be 45 characters")
		}
		for _, r := range prompt {
			if unicode.IsControl(r) {
				return nil, fmt.Errorf("questions cannot include a line break")
			}
		}
		out = append(out, prompt)
	}
	return out, nil
}

// ClassifyAge reports whether a whole-number answer is under 18.
// ok is false when the answer is not a whole number.
func ClassifyAge(raw string) (under bool, ok bool) {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return false, false
	}
	if n < MinVerifyAge {
		return true, true
	}
	return false, true
}
