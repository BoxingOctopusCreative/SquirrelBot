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
}
