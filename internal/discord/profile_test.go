package discord

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGuildProfileBody(t *testing.T) {
	if GuildProfileBody("Canopy", false, nil, "", false) != nil {
		t.Fatal("unchanged profile should not call Discord")
	}
	body := GuildProfileBody("", true, nil, "image/png", true)
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"nick":null`) || !strings.Contains(string(raw), `"avatar":null`) {
		t.Fatalf("clear payload %s", raw)
	}
	body = GuildProfileBody("Canopy", true, []byte{1, 2, 3}, "image/png", true)
	raw, err = json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if !strings.Contains(text, `"nick":"Canopy"`) || !strings.Contains(text, `data:image/png;base64,`) {
		t.Fatalf("set payload %s", text)
	}
	body = GuildProfileBody("Canopy", false, []byte{1}, "image/gif", true)
	if _, ok := body["nick"]; ok {
		t.Fatal("avatar-only update should omit the nickname")
	}
}
