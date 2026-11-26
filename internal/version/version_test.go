package version

import "testing"

func TestStringUsesStampedRelease(t *testing.T) {
	previous := Version
	t.Cleanup(func() { Version = previous })

	Version = "v0.7.1"
	if got := String(); got != "0.7.1" {
		t.Fatalf("String() = %q", got)
	}

	Version = "1.2.3-dirty"
	if got := String(); got != "1.2.3-dirty" {
		t.Fatalf("String() = %q", got)
	}
}

func TestStringFallsBackWhenUnstamped(t *testing.T) {
	previous := Version
	t.Cleanup(func() { Version = previous })
	Version = "dev"
	if got := String(); got == "" {
		t.Fatal("String() is empty")
	}
}
