package domain

import (
	"strings"
	"testing"
)

func TestCleanConfession(t *testing.T) {
	got, err := CleanConfession("  I ate the last acorn.  ")
	if err != nil || got != "I ate the last acorn." {
		t.Fatalf("trim: %q %v", got, err)
	}
	got, err = CleanConfession("hello @everyone and @HERE")
	if err != nil || strings.Contains(got, "@everyone") || strings.Contains(got, "@HERE") {
		t.Fatalf("broadcast: %q %v", got, err)
	}
	if _, err := CleanConfession("  \n  "); err == nil {
		t.Fatal("blank confession should fail")
	}
	if _, err := CleanConfession(strings.Repeat("a", MaxConfession+1)); err == nil {
		t.Fatal("long confession should fail")
	}
	got, err = CleanConfession("ok\u200b")
	if err != nil || got != "ok" {
		t.Fatalf("invisible: %q %v", got, err)
	}
}

func TestValidateConfessionChannels(t *testing.T) {
	if err := ValidateConfessionChannels("", ""); err != nil {
		t.Fatal(err)
	}
	review := "777777777777777771"
	public := "777777777777777772"
	if err := ValidateConfessionChannels(review, public); err != nil {
		t.Fatal(err)
	}
	if err := ValidateConfessionChannels(review, review); err == nil {
		t.Fatal("the same channel should fail")
	}
	if err := ValidateConfessionChannels("nope", ""); err == nil {
		t.Fatal("bad review id should fail")
	}
}

func TestParseConfessionAction(t *testing.T) {
	action, id, ok := ParseConfessionAction(ConfessionCustomID("approve", 42))
	if !ok || action != "approve" || id != 42 {
		t.Fatalf("approve: %s %d %v", action, id, ok)
	}
	action, id, ok = ParseConfessionAction(ConfessionCustomID("reject", 7))
	if !ok || action != "reject" || id != 7 {
		t.Fatalf("reject: %s %d %v", action, id, ok)
	}
	if _, _, ok := ParseConfessionAction("sb:verify"); ok {
		t.Fatal("other buttons are not confessions")
	}
	if _, _, ok := ParseConfessionAction("sb:confess:approve:0"); ok {
		t.Fatal("id 0 should fail")
	}
	if _, _, ok := ParseConfessionAction("sb:confess:drop:4"); ok {
		t.Fatal("unknown action should fail")
	}
}
