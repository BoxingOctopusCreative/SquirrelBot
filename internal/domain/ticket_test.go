package domain

import (
	"strings"
	"testing"
)

func TestCleanTicketField(t *testing.T) {
	got, err := CleanTicketField("  hello \n there  ", 20, "write something", false)
	if err != nil || got != "hello \n there" {
		t.Fatalf("clean %q %v", got, err)
	}
	got, err = CleanTicketField("hello\neveryone", 20, "write something", true)
	if err != nil || got != "hello everyone" {
		t.Fatalf("single line %q %v", got, err)
	}
	if _, err := CleanTicketField("   ", 20, "write a subject", true); err == nil {
		t.Fatal("blank should fail")
	}
	if _, err := CleanTicketField("too long", 3, "write a subject", true); err == nil {
		t.Fatal("long text should fail")
	}
	got, err = CleanTicketField("ping @everyone", 40, "", true)
	if err != nil || got != "ping @ everyone" {
		t.Fatalf("ping %q %v", got, err)
	}
}

func TestPrepareTicketPanel(t *testing.T) {
	title, body, err := PrepareTicketPanel("", "")
	if err != nil || title != DefaultTicketTitle || body != DefaultTicketBody {
		t.Fatalf("defaults %q %q %v", title, body, err)
	}
	if _, _, err := PrepareTicketPanel(strings.Repeat("a", MaxTicketTitle+1), "ok"); err == nil {
		t.Fatal("long title should fail")
	}
}

func TestValidateTicketChannels(t *testing.T) {
	if err := ValidateTicketChannels("nope", "", "", ""); err == nil {
		t.Fatal("bad panel id")
	}
	panel := "777777777777777771"
	if err := ValidateTicketChannels(panel, "", "", panel); err == nil {
		t.Fatal("panel and transcript should differ")
	}
	if err := ValidateTicketChannels(panel, "", "888888888888888882", "777777777777777772"); err != nil {
		t.Fatal(err)
	}
}

func TestTicketReady(t *testing.T) {
	if err := TicketReady("", "777777777777777771", "888888888888888882"); err == nil {
		t.Fatal("missing panel")
	}
	if err := TicketReady("777777777777777771", "777777777777777773", "888888888888888882"); err != nil {
		t.Fatal(err)
	}
}

func TestTicketChannelText(t *testing.T) {
	open := TicketChannelText(TicketView{
		Number: 4, OpenerID: "222222222222222222", SupportRoleID: "888888888888888882",
		Subject: "Map pin", Body: "The pin is missing.", Status: TicketOpen,
	})
	if !hasAll(open, "Ticket 4 is open.", "Map pin", "Nobody has claimed this ticket.") {
		t.Fatalf("open text %q", open)
	}
	claimed := TicketChannelText(TicketView{
		Number: 4, Status: TicketOpen, Subject: "Map pin", Body: "The pin is missing.",
		ClaimerID: "333333333333333333",
	})
	if !hasAll(claimed, "Claimed by <@333333333333333333>.") {
		t.Fatalf("claimed text %q", claimed)
	}
	closed := TicketChannelText(TicketView{
		Number: 4, Status: TicketClosed, Subject: "Map pin", Body: "The pin is missing.",
		ClosedByName: "Ada", CloseReason: "Fixed the pin.",
	})
	if !hasAll(closed, "Ticket 4 is closed.", "Closed by Ada.", "Fixed the pin.", "Reopen puts this ticket back.") {
		t.Fatalf("closed text %q", closed)
	}
}

func TestFormatTicketTranscript(t *testing.T) {
	text := FormatTicketTranscript(TicketTranscript{
		Number: 4, OpenerName: "Ada", Subject: "Map pin", Body: "The pin is missing.",
		ClosedBy: "Bea", Reason: "Fixed the pin.", Lines: []string{"2026-11-13 12:00 Ada: hello"},
	})
	if !hasAll(text, "Ticket 4", "Opened by Ada", "Fixed the pin.", "Ada: hello") {
		t.Fatalf("transcript %q", text)
	}
	empty := FormatTicketTranscript(TicketTranscript{Number: 1, OpenerName: "Ada", Subject: "Hi", Body: "There"})
	if !hasAll(empty, "No messages were copied.") {
		t.Fatalf("empty transcript %q", empty)
	}
}

func TestParseTicketAction(t *testing.T) {
	action, id, ok := ParseTicketAction("sb:ticket:open")
	if !ok || action != "open" || id != 0 {
		t.Fatalf("open %s %d %v", action, id, ok)
	}
	action, id, ok = ParseTicketAction("sb:ticket:close-form:12")
	if !ok || action != "close-form" || id != 12 {
		t.Fatalf("close form %s %d %v", action, id, ok)
	}
	if _, _, ok := ParseTicketAction("sb:ticket:open:1"); ok {
		t.Fatal("open should not take an id")
	}
	if _, _, ok := ParseTicketAction("sb:confess:approve:1"); ok {
		t.Fatal("other buttons are not tickets")
	}
}

func TestTicketNumberMatches(t *testing.T) {
	if !TicketNumberMatches("12", 12) || !TicketNumberMatches("#12", 12) || !TicketNumberMatches("ticket 12", 12) || !TicketNumberMatches("ticket-12", 12) {
		t.Fatal("expected the ticket number to match")
	}
	if TicketNumberMatches("13", 12) || TicketNumberMatches("", 12) || TicketNumberMatches("12", 0) {
		t.Fatal("wrong number should not match")
	}
}

func hasAll(text string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(text, part) {
			return false
		}
	}
	return true
}
