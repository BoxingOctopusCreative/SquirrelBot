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
	MaxTicketSubject = 100
	MaxTicketBody    = 1000
	MaxTicketReason  = 200
	MaxTicketTitle   = 80
	MaxTicketPanel   = 800

	TicketOpen    = "open"
	TicketClosed  = "closed"
	TicketDeleted = "deleted"

	DefaultTicketTitle = "Support"
	DefaultTicketBody  = "Open a ticket and a private channel is created for you and the support team."
)

var ticketBroadcast = regexp.MustCompile(`(?i)@(everyone|here)`)

// CleanTicketField trims text, drops control characters, and blocks server-wide pings.
// singleLine collapses line breaks. emptyMsg is the error when the result is blank.
func CleanTicketField(text string, max int, emptyMsg string, singleLine bool) (string, error) {
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
	if singleLine {
		out = strings.Join(strings.Fields(strings.ReplaceAll(out, "\n", " ")), " ")
	}
	out = ticketBroadcast.ReplaceAllString(out, "@ $1")
	if out == "" {
		if emptyMsg == "" {
			return "", nil
		}
		return "", fmt.Errorf("%s", emptyMsg)
	}
	if utf8.RuneCountInString(out) > max {
		return "", fmt.Errorf("keep this to %d characters", max)
	}
	return out, nil
}

// PrepareTicketPanel fills the panel title and message, and checks their length.
func PrepareTicketPanel(title, body string) (string, string, error) {
	title, err := CleanTicketField(title, MaxTicketTitle, "", true)
	if err != nil {
		return "", "", fmt.Errorf("the panel title is too long. Keep it to %d characters", MaxTicketTitle)
	}
	body, err = CleanTicketField(body, MaxTicketPanel, "", false)
	if err != nil {
		return "", "", fmt.Errorf("the panel message is too long. Keep it to %d characters", MaxTicketPanel)
	}
	if title == "" {
		title = DefaultTicketTitle
	}
	if body == "" {
		body = DefaultTicketBody
	}
	return title, body, nil
}

// ValidateTicketChannels checks the ids saved for a ticket setup.
func ValidateTicketChannels(panelID, categoryID, supportRoleID, logID string) error {
	if !OptionalSnowflake(panelID) {
		return fmt.Errorf("the panel channel is not a valid id")
	}
	if !OptionalSnowflake(categoryID) {
		return fmt.Errorf("the category is not a valid id")
	}
	if !OptionalSnowflake(supportRoleID) {
		return fmt.Errorf("the support role is not a valid id")
	}
	if !OptionalSnowflake(logID) {
		return fmt.Errorf("the transcript channel is not a valid id")
	}
	if panelID != "" && panelID == logID {
		return fmt.Errorf("use a different channel for the panel and the transcripts")
	}
	return nil
}

// TicketReady reports whether a server can open tickets and post the panel.
func TicketReady(panelID, categoryID, supportRoleID string) error {
	if !ValidSnowflake(panelID) {
		return fmt.Errorf("Choose a panel channel.")
	}
	if !ValidSnowflake(categoryID) {
		return fmt.Errorf("Choose a category for ticket channels.")
	}
	if !ValidSnowflake(supportRoleID) {
		return fmt.Errorf("Choose the support role.")
	}
	return nil
}

// TicketPanelText is the message posted above the Open a ticket button.
func TicketPanelText(title, body string) string {
	if strings.TrimSpace(title) == "" {
		title = DefaultTicketTitle
	}
	if strings.TrimSpace(body) == "" {
		body = DefaultTicketBody
	}
	return "**" + title + "**\n" + body
}

// TicketView is the state shown in the ticket channel.
type TicketView struct {
	Number        int
	OpenerID      string
	SupportRoleID string
	Subject       string
	Body          string
	Status        string
	ClaimerID     string
	ClosedByName  string
	CloseReason   string
}

// TicketChannelText is the control message in the ticket channel.
func TicketChannelText(v TicketView) string {
	var b strings.Builder
	if v.Status == TicketClosed {
		fmt.Fprintf(&b, "Ticket %d is closed.\n", v.Number)
		name := strings.TrimSpace(v.ClosedByName)
		if name == "" {
			name = "someone"
		}
		fmt.Fprintf(&b, "Closed by %s.\n", name)
		reason := strings.TrimSpace(v.CloseReason)
		if reason == "" {
			reason = "No reason given."
		}
		fmt.Fprintf(&b, "Reason: %s\n\n", reason)
		b.WriteString("The transcript is saved. Reopen puts this ticket back. Delete asks for the ticket number, then removes the channel.\n\n")
	} else {
		fmt.Fprintf(&b, "Ticket %d is open.\n", v.Number)
		if v.OpenerID != "" {
			fmt.Fprintf(&b, "<@%s> asked for help. ", v.OpenerID)
		}
		if v.SupportRoleID != "" {
			fmt.Fprintf(&b, "<@&%s> can claim it, then close it with a reason.\n\n", v.SupportRoleID)
		} else {
			b.WriteString("\n\n")
		}
	}
	fmt.Fprintf(&b, "**%s**\n%s", v.Subject, v.Body)
	if v.Status != TicketClosed {
		b.WriteString("\n\n")
		if v.ClaimerID != "" {
			fmt.Fprintf(&b, "Claimed by <@%s>.", v.ClaimerID)
		} else {
			b.WriteString("Nobody has claimed this ticket.")
		}
	}
	return b.String()
}

// TicketTranscript is the saved copy of a closed ticket.
type TicketTranscript struct {
	Number     int
	OpenerName string
	Subject    string
	Body       string
	ClosedBy   string
	Reason     string
	Note       string
	Lines      []string
}

// FormatTicketTranscript builds the text file posted to the transcript channel.
func FormatTicketTranscript(item TicketTranscript) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Ticket %d\n", item.Number)
	fmt.Fprintf(&b, "Opened by %s\n", item.OpenerName)
	fmt.Fprintf(&b, "Subject: %s\n", item.Subject)
	fmt.Fprintf(&b, "Request: %s\n", item.Body)
	if strings.TrimSpace(item.ClosedBy) != "" {
		fmt.Fprintf(&b, "Closed by %s\n", item.ClosedBy)
	}
	if strings.TrimSpace(item.Reason) != "" {
		fmt.Fprintf(&b, "Reason: %s\n", item.Reason)
	}
	if strings.TrimSpace(item.Note) != "" {
		fmt.Fprintf(&b, "%s\n", item.Note)
	}
	b.WriteString("\n")
	if len(item.Lines) == 0 {
		b.WriteString("No messages were copied.\n")
		return b.String()
	}
	for _, line := range item.Lines {
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

// TicketCustomID is a button or form id for one ticket action.
func TicketCustomID(action string, id int64) string {
	if id <= 0 {
		return "sb:ticket:" + action
	}
	return fmt.Sprintf("sb:ticket:%s:%d", action, id)
}

// ParseTicketAction reads a ticket button or form id.
func ParseTicketAction(customID string) (action string, id int64, ok bool) {
	const prefix = "sb:ticket:"
	if !strings.HasPrefix(customID, prefix) {
		return "", 0, false
	}
	rest := customID[len(prefix):]
	action, raw, found := strings.Cut(rest, ":")
	switch action {
	case "open", "form":
		return action, 0, !found
	case "claim", "release", "close", "close-form", "reopen", "delete", "delete-form":
		if !found || strings.Contains(raw, ":") {
			return "", 0, false
		}
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || n <= 0 {
			return "", 0, false
		}
		return action, n, true
	default:
		return "", 0, false
	}
}

// TicketNumberMatches reports whether typed text is the ticket number.
// Accepts 12, #12, and ticket 12.
func TicketNumberMatches(typed string, number int) bool {
	if number <= 0 {
		return false
	}
	typed = strings.ToLower(strings.TrimSpace(typed))
	typed = strings.TrimPrefix(typed, "#")
	typed = strings.TrimSpace(typed)
	typed = strings.TrimPrefix(typed, "ticket")
	typed = strings.Trim(typed, "-# ")
	return typed == strconv.Itoa(number)
}
