package discord

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestCommandPayloadOmitsNulls(t *testing.T) {
	body, err := commandPayload()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(body, []byte(`null`)) {
		t.Fatalf("payload still contains null: %s", body)
	}
	var commands []map[string]any
	if err := json.Unmarshal(body, &commands); err != nil {
		t.Fatal(err)
	}
	if len(commands) < 10 {
		t.Fatalf("expected the moderation command set, got %d", len(commands))
	}
	names := map[string]bool{}
	for _, command := range commands {
		name, _ := command["name"].(string)
		names[name] = true
	}
	for _, name := range []string{"confession", "confession-review", "confession-channel"} {
		if !names[name] {
			t.Fatalf("missing %s", name)
		}
	}
}
