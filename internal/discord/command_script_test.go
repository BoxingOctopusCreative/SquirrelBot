package discord

import "testing"

func TestSplitCommand(t *testing.T) {
	name, args, text, ok := splitCommand("!join hello there", "!")
	if !ok || name != "join" || text != "hello there" || len(args) != 2 || args[0] != "hello" || args[1] != "there" {
		t.Fatalf("name=%s args=%v text=%s ok=%v", name, args, text, ok)
	}
	name, args, text, ok = splitCommand("?Ping", "?")
	if !ok || name != "ping" || text != "" || len(args) != 0 {
		t.Fatalf("name=%s args=%v text=%s ok=%v", name, args, text, ok)
	}
}

func TestPublicEffectErrorHidesHTTP(t *testing.T) {
	if got := publicEffectError(errString("HTTP 403 Forbidden, {\"message\":\"Missing Permissions\"}")); got != "Discord refused that action." {
		t.Fatalf("%s", got)
	}
	if got := publicEffectError(errString("That role is managed by an integration.")); got != "That role is managed by an integration." {
		t.Fatalf("%s", got)
	}
}

type errString string

func (e errString) Error() string { return string(e) }
