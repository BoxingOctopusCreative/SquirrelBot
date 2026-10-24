package api

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"
)

func TestDiscordAuthorizeInvitesManagedServers(t *testing.T) {
	raw := discordAuthorizeURL("app", "http://localhost:5173/auth/discord/callback", "state")
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	if query.Get("scope") != oauthScopes {
		t.Fatalf("scope %q", query.Get("scope"))
	}
	if !strings.Contains(raw, "scope=identify%20guilds%20bot%20applications.commands") {
		t.Fatalf("scope encoding %s", raw)
	}
	if strings.Contains(raw, "identify+") {
		t.Fatalf("scope used plus encoding %s", raw)
	}
	if query.Get("permissions") != botInvitePermissions {
		t.Fatalf("permissions %q", query.Get("permissions"))
	}
	if query.Get("integration_type") != "0" {
		t.Fatalf("integration_type %q", query.Get("integration_type"))
	}
	if query.Get("guild_id") != "" {
		t.Fatal("a server must not be preselected")
	}
	if query.Get("prompt") != "consent" {
		t.Fatalf("prompt %q", query.Get("prompt"))
	}
	if query.Get("response_type") != "code" {
		t.Fatalf("response_type %q", query.Get("response_type"))
	}
}

func TestDiscordGuildPermissionsAcceptStringOrNumber(t *testing.T) {
	raw := []byte(`[
		{"id":"100000000000000001","name":"Canopy","icon":null,"owner":true,"permissions":2147483648},
		{"id":"100000000000000002","name":"Den","icon":null,"owner":false,"permissions":"8"}
	]`)
	var guilds []discordGuild
	if err := json.Unmarshal(raw, &guilds); err != nil {
		t.Fatal(err)
	}
	if string(guilds[0].Permissions) != "2147483648" {
		t.Fatalf("number permissions = %q", guilds[0].Permissions)
	}
	if string(guilds[1].Permissions) != "8" {
		t.Fatalf("string permissions = %q", guilds[1].Permissions)
	}
}
