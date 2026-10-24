package domain

import (
	"strings"
	"testing"
	"time"
)

func TestGateTiming(t *testing.T) {
	joined := time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)
	deadline := GateDeadline(joined, 24*time.Hour)
	if !deadline.Equal(joined.Add(24 * time.Hour)) {
		t.Fatalf("deadline %s", deadline)
	}
	warnBefore := time.Hour
	early := joined.Add(22 * time.Hour)
	if GateShouldWarn(early, deadline, warnBefore, false) {
		t.Fatal("too early to warn")
	}
	inside := joined.Add(23*time.Hour + 5*time.Minute)
	if !GateShouldWarn(inside, deadline, warnBefore, false) {
		t.Fatal("expected a warning")
	}
	if GateShouldWarn(inside, deadline, warnBefore, true) {
		t.Fatal("already warned")
	}
	if GateShouldKick(inside, deadline) {
		t.Fatal("not due for a kick")
	}
	if !GateShouldKick(deadline, deadline) {
		t.Fatal("kick at the deadline")
	}
}

func TestGatePanelText(t *testing.T) {
	text := GatePanelText()
	if !strings.Contains(text, "18 or older") {
		t.Fatalf("age rule missing: %s", text)
	}
}

func TestCleanGateQuestions(t *testing.T) {
	got, err := CleanGateQuestions([]string{"  Where did you join from?  "})
	if err != nil || len(got) != 1 || got[0] != "Where did you join from?" {
		t.Fatalf("got %#v err %v", got, err)
	}
	if _, err := CleanGateQuestions([]string{"", "ok"}); err == nil {
		t.Fatal("blank question")
	}
	if _, err := CleanGateQuestions([]string{"one", "two", "three", "four", "five"}); err == nil {
		t.Fatal("too many questions")
	}
}

func TestClassifyAge(t *testing.T) {
	under, ok := ClassifyAge(" 17 ")
	if !ok || !under {
		t.Fatal("17 should be underage")
	}
	under, ok = ClassifyAge("18")
	if !ok || under {
		t.Fatal("18 should pass")
	}
	under, ok = ClassifyAge("0")
	if !ok || !under {
		t.Fatal("0 should be underage")
	}
	if _, ok = ClassifyAge("18 years"); ok {
		t.Fatal("words are not a whole number")
	}
	if _, ok = ClassifyAge(""); ok {
		t.Fatal("blank is not a whole number")
	}
}
