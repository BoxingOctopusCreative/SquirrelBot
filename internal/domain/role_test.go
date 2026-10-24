package domain

import (
	"strings"
	"testing"
)

func TestCleanRoleName(t *testing.T) {
	got, err := CleanRoleName("  Subscribers  ")
	if err != nil || got != "Subscribers" {
		t.Fatalf("got %q %v", got, err)
	}
	if _, err := CleanRoleName("  "); err == nil {
		t.Fatal("blank name")
	}
	if _, err := CleanRoleName("@everyone"); err == nil {
		t.Fatal("reserved name")
	}
	if _, err := CleanRoleName("line\nbreak"); err == nil {
		t.Fatal("line break")
	}
	if _, err := CleanRoleName(strings.Repeat("a", 101)); err == nil {
		t.Fatal("long name")
	}
}
