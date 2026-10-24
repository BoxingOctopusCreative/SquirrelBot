package secretbox

import "testing"

func TestRoundTrip(t *testing.T) {
	key := Key("a-session-secret-that-is-long-enough")
	sealed, err := Seal(key, "discord-access-token")
	if err != nil {
		t.Fatal(err)
	}
	if sealed == "discord-access-token" {
		t.Fatal("token was stored in the clear")
	}
	got, err := Open(key, sealed)
	if err != nil {
		t.Fatal(err)
	}
	if got != "discord-access-token" {
		t.Fatalf("got %q", got)
	}
	if _, err := Open(Key("another-session-secret-that-is-long-enough"), sealed); err == nil {
		t.Fatal("expected decrypt to fail with a different key")
	}
}
