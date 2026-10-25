//go:build e2e

package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/BoxingOctopus/squirrelbot/e2e/harness"
)

var stack *harness.Stack

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	st, err := harness.Start(ctx, harness.Options{})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	stack = st
	code := m.Run()
	st.Close()
	os.Exit(code)
}

func TestHealthAndSignIn(t *testing.T) {
	status, body := call(t, http.MethodGet, "/healthz", nil, "")
	if status != http.StatusOK || !strings.Contains(string(body), `"ok":true`) {
		t.Fatalf("health %d %s", status, body)
	}
	res := raw(t, http.MethodGet, "/healthz", nil, "")
	if res.Header.Get("X-Content-Type-Options") != "nosniff" || res.Header.Get("X-Frame-Options") != "DENY" {
		t.Fatalf("security headers: %#v", res.Header)
	}
	res.Body.Close()

	status, body = call(t, http.MethodGet, "/readyz", nil, "")
	if status != http.StatusOK {
		t.Fatalf("ready %d %s", status, body)
	}

	status, _ = call(t, http.MethodGet, "/api/me", nil, "")
	if status != http.StatusUnauthorized {
		t.Fatalf("anonymous me %d", status)
	}

	blocked := request(t, http.MethodPost, "/api/guilds/"+harness.GuildID+"/automod", map[string]any{"action": "delete"}, stack.Cookie)
	blocked.Header.Set("Origin", "https://evil.example")
	res = do(t, blocked)
	if res.StatusCode != http.StatusForbidden {
		body, _ = io.ReadAll(res.Body)
		t.Fatalf("cross origin %d %s", res.StatusCode, body)
	}
	res.Body.Close()

	res = noFollow(t, http.MethodGet, "/auth/discord", nil, "")
	if res.StatusCode != http.StatusFound || !strings.Contains(res.Header.Get("Location"), "https://discord.com/oauth2/authorize") {
		t.Fatalf("login redirect %d %s", res.StatusCode, res.Header.Get("Location"))
	}
	var state string
	for _, cookie := range res.Cookies() {
		if cookie.Name == "sb_oauth_state" {
			state = cookie.Value
		}
	}
	res.Body.Close()
	if state == "" {
		t.Fatal("login did not set the state cookie")
	}

	res = noFollow(t, http.MethodGet, "/auth/discord/callback?error=denied", nil, "")
	if res.StatusCode != http.StatusFound || !strings.Contains(res.Header.Get("Location"), "error=denied") {
		t.Fatalf("denied callback %d %s", res.StatusCode, res.Header.Get("Location"))
	}
	res.Body.Close()

	callback := request(t, http.MethodGet, "/auth/discord/callback?code=e2e-code&state="+state, nil, "")
	callback.AddCookie(&http.Cookie{Name: "sb_oauth_state", Value: state})
	client := &http.Client{CheckRedirect: stopRedirect, Timeout: 10 * time.Second}
	res, err := client.Do(callback)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusFound || res.Header.Get("Location") != "/app" {
		t.Fatalf("callback %d %s", res.StatusCode, res.Header.Get("Location"))
	}
	var signedIn string
	for _, cookie := range res.Cookies() {
		if cookie.Name == "sb_session" && cookie.Value != "" {
			signedIn = cookie.Value
		}
	}
	if signedIn == "" {
		t.Fatal("callback did not set a session cookie")
	}
	status, body = call(t, http.MethodGet, "/api/me", nil, signedIn)
	if status != http.StatusOK || !strings.Contains(string(body), harness.UserName) {
		t.Fatalf("signed-in me %d %s", status, body)
	}

	status, _ = call(t, http.MethodPost, "/auth/logout", nil, signedIn)
	if status != http.StatusNoContent {
		t.Fatalf("logout %d", status)
	}
	status, _ = call(t, http.MethodGet, "/api/me", nil, signedIn)
	if status != http.StatusUnauthorized {
		t.Fatalf("me after logout %d", status)
	}
}

func TestGuildAccess(t *testing.T) {
	status, body := call(t, http.MethodGet, "/api/guilds", nil, stack.Cookie)
	if status != http.StatusOK {
		t.Fatalf("guilds %d %s", status, body)
	}
	if !strings.Contains(string(body), harness.GuildName) || strings.Contains(string(body), "Hollow") {
		t.Fatalf("guild list %s", body)
	}

	status, body = call(t, http.MethodGet, "/api/guilds/"+harness.HollowGuildID, nil, stack.Cookie)
	if status != http.StatusNotFound {
		t.Fatalf("missing bot %d %s", status, body)
	}
	status, body = call(t, http.MethodGet, "/api/guilds/"+harness.GuildID, nil, stack.GuestCookie)
	if status != http.StatusForbidden {
		t.Fatalf("guest %d %s", status, body)
	}
	status, body = call(t, http.MethodGet, "/api/guilds/not-a-snowflake", nil, stack.Cookie)
	if status != http.StatusNotFound {
		t.Fatalf("bad id %d %s", status, body)
	}
}

func TestDashboardJourney(t *testing.T) {
	status, body := call(t, http.MethodGet, "/api/me", nil, stack.Cookie)
	must(t, status, body, http.StatusOK)
	if !strings.Contains(string(body), harness.UserName) {
		t.Fatalf("me %s", body)
	}

	status, body = call(t, http.MethodGet, "/api/guilds/"+harness.GuildID, nil, stack.Cookie)
	must(t, status, body, http.StatusOK)
	bundle := asMap(t, body)
	guild := bundle["guild"].(map[string]any)
	if guild["name"] != harness.GuildName || guild["prefix"] != "!" {
		t.Fatalf("guild %v", guild)
	}
	channels, _ := bundle["channels"].([]any)
	roles, _ := bundle["roles"].([]any)
	if len(channels) < 2 || len(roles) < 3 {
		t.Fatalf("metadata channels %d roles %d", len(channels), len(roles))
	}

	status, body = call(t, http.MethodPatch, "/api/guilds/"+harness.GuildID, map[string]any{
		"modRoleId": harness.ModRoleID, "adminRoleId": harness.AdminRoleID,
		"welcomeChannelId": harness.GeneralChannelID, "welcomeMessage": "Hello {user}",
		"prefix": "bad prefix",
	}, stack.Cookie)
	must(t, status, body, http.StatusBadRequest)

	status, body = call(t, http.MethodPatch, "/api/guilds/"+harness.GuildID, map[string]any{
		"modLogChannelId": harness.GeneralChannelID, "modRoleId": harness.ModRoleID,
		"adminRoleId": harness.AdminRoleID, "welcomeChannelId": harness.GeneralChannelID,
		"welcomeMessage": "Hello {user}", "autoroleId": harness.MemberRoleID, "prefix": "?",
	}, stack.Cookie)
	must(t, status, body, http.StatusOK)
	if asMap(t, body)["prefix"] != "?" || asMap(t, body)["welcomeMessage"] != "Hello {user}" {
		t.Fatalf("settings %s", body)
	}

	status, body = call(t, http.MethodPut, "/api/guilds/"+harness.GuildID+"/automod", map[string]any{
		"enabled": true, "action": "ban", "timeoutSeconds": 600,
	}, stack.Cookie)
	must(t, status, body, http.StatusBadRequest)
	status, body = call(t, http.MethodPut, "/api/guilds/"+harness.GuildID+"/automod", map[string]any{
		"enabled": true, "blockInvites": true, "blockSpam": true,
		"mentionLimit": 4, "capsPercent": 80, "capsMinLength": 10,
		"badWords": []string{" Slur ", "nope"}, "action": "timeout", "timeoutSeconds": 120,
		"ignoredChannelIds": []string{harness.GeneralChannelID},
		"ignoredRoleIds":    []string{harness.ModRoleID},
	}, stack.Cookie)
	must(t, status, body, http.StatusOK)
	savedAutomod := asMap(t, body)
	if savedAutomod["action"] != "timeout" || savedAutomod["enabled"] != true {
		t.Fatalf("automod %s", body)
	}

	status, body = call(t, http.MethodPut, "/api/guilds/"+harness.GuildID+"/logging", map[string]any{
		"channelId": harness.GeneralChannelID, "messageDelete": true, "messageEdit": false,
		"memberJoin": true, "memberLeave": true, "memberBan": true,
	}, stack.Cookie)
	must(t, status, body, http.StatusOK)
	if asMap(t, body)["messageEdit"] != false {
		t.Fatalf("logging %s", body)
	}

	status, body = call(t, http.MethodPut, "/api/guilds/"+harness.GuildID+"/starboard", map[string]any{
		"enabled": true, "emoji": "⭐", "threshold": 3, "ignoredChannelIds": []string{},
	}, stack.Cookie)
	must(t, status, body, http.StatusBadRequest)
	status, body = call(t, http.MethodPut, "/api/guilds/"+harness.GuildID+"/starboard", map[string]any{
		"enabled": true, "channelId": harness.AnnounceChannelID, "emoji": "acorn:" + harness.EmojiID,
		"threshold": 4, "selfStar": false, "nsfw": false,
		"ignoredChannelIds": []string{harness.GeneralChannelID},
	}, stack.Cookie)
	must(t, status, body, http.StatusOK)
	if asMap(t, body)["threshold"] != float64(4) {
		t.Fatalf("starboard %s", body)
	}

	status, body = call(t, http.MethodGet, "/api/guilds/"+harness.GuildID+"/profile", nil, stack.Cookie)
	must(t, status, body, http.StatusOK)
	if asMap(t, body)["username"] != "squirrelbot" {
		t.Fatalf("profile %s", body)
	}
	status, body = putProfile(t, "Canopy Guide")
	must(t, status, body, http.StatusOK)
	if asMap(t, body)["nick"] != "Canopy Guide" {
		t.Fatalf("saved profile %s", body)
	}

	status, body = call(t, http.MethodPost, "/api/guilds/"+harness.GuildID+"/commands", map[string]any{
		"name": "hello", "response": "function(", "roleIds": []string{},
	}, stack.Cookie)
	must(t, status, body, http.StatusBadRequest)
	status, body = call(t, http.MethodPost, "/api/guilds/"+harness.GuildID+"/commands", map[string]any{
		"name": "hello", "response": `return "hello"`, "roleIds": []string{harness.MemberRoleID},
	}, stack.Cookie)
	must(t, status, body, http.StatusCreated)
	status, body = call(t, http.MethodGet, "/api/guilds/"+harness.GuildID+"/commands", nil, stack.Cookie)
	must(t, status, body, http.StatusOK)
	if !strings.Contains(string(body), `"hello"`) {
		t.Fatalf("commands %s", body)
	}
	status, _ = call(t, http.MethodDelete, "/api/guilds/"+harness.GuildID+"/commands/hello", nil, stack.Cookie)
	if status != http.StatusNoContent {
		t.Fatalf("delete command %d", status)
	}

	when := time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339)
	status, body = call(t, http.MethodPost, "/api/guilds/"+harness.GuildID+"/announcements", map[string]any{
		"channelId": harness.AnnounceChannelID, "title": "Later", "body": "Not yet.",
		"scheduledAt": when,
	}, stack.Cookie)
	must(t, status, body, http.StatusCreated)
	announcementID := int(asMap(t, body)["id"].(float64))
	if asMap(t, body)["status"] != "scheduled" {
		t.Fatalf("scheduled %s", body)
	}
	status, _ = call(t, http.MethodDelete, "/api/guilds/"+harness.GuildID+"/announcements/"+strconv.Itoa(announcementID), nil, stack.Cookie)
	if status != http.StatusNoContent {
		t.Fatalf("delete announcement %d", status)
	}
	status, body = call(t, http.MethodPost, "/api/guilds/"+harness.GuildID+"/announcements", map[string]any{
		"channelId": harness.AnnounceChannelID, "title": "Canopy night", "body": "The canopy is open.",
	}, stack.Cookie)
	must(t, status, body, http.StatusCreated)
	if asMap(t, body)["status"] != "sent" || asMap(t, body)["messageId"] == "" {
		t.Fatalf("sent announcement %s", body)
	}

	status, body = call(t, http.MethodPost, "/api/guilds/"+harness.GuildID+"/announcement-templates", map[string]any{
		"name": "Outage", "title": "{{ server }} downtime", "body": "{{ server }} pauses at {{ when }}. Posted by {{ author }}.",
	}, stack.Cookie)
	must(t, status, body, http.StatusCreated)
	templateID := int(asMap(t, body)["id"].(float64))
	status, body = call(t, http.MethodPost, "/api/guilds/"+harness.GuildID+"/announcement-templates/"+strconv.Itoa(templateID)+"/render", map[string]any{
		"vars": map[string]string{"when": "Tonight"},
	}, stack.Cookie)
	must(t, status, body, http.StatusOK)
	rendered := asMap(t, body)
	if rendered["title"] != "Canopy downtime" || !strings.Contains(rendered["body"].(string), "Tonight") || !strings.Contains(rendered["body"].(string), harness.UserName) {
		t.Fatalf("render %s", body)
	}
	status, body = call(t, http.MethodPut, "/api/guilds/"+harness.GuildID+"/announcement-templates/"+strconv.Itoa(templateID), map[string]any{
		"name": "Outage", "title": "Updated", "body": "Plain text.",
	}, stack.Cookie)
	must(t, status, body, http.StatusOK)
	status, _ = call(t, http.MethodDelete, "/api/guilds/"+harness.GuildID+"/announcement-templates/"+strconv.Itoa(templateID), nil, stack.Cookie)
	if status != http.StatusNoContent {
		t.Fatalf("delete template %d", status)
	}

	status, body = call(t, http.MethodPut, "/api/guilds/"+harness.GuildID+"/escalation-roles", map[string]any{
		"roles": []any{map[string]any{
			"roleId": harness.ModRoleID, "tier": "mod", "maxDurationSeconds": 86400,
		}},
	}, stack.Cookie)
	must(t, status, body, http.StatusOK)
	status, body = call(t, http.MethodPost, "/api/guilds/"+harness.GuildID+"/escalations", map[string]any{
		"userId": harness.TargetUserID, "roleId": harness.ModRoleID, "duration": "4h", "reason": "Covering the night shift",
	}, stack.Cookie)
	must(t, status, body, http.StatusCreated)
	grant := asMap(t, body)
	if grant["status"] != "active" || grant["userName"] != harness.TargetName {
		t.Fatalf("grant %s", body)
	}
	grantID := int(grant["id"].(float64))
	status, body = call(t, http.MethodGet, "/api/guilds/"+harness.GuildID+"/escalations", nil, stack.Cookie)
	must(t, status, body, http.StatusOK)
	if !strings.Contains(string(body), "Covering the night shift") {
		t.Fatalf("escalations %s", body)
	}
	status, _ = call(t, http.MethodPost, "/api/guilds/"+harness.GuildID+"/escalations/"+strconv.Itoa(grantID)+"/revoke", map[string]any{
		"reason": "Shift ended",
	}, stack.Cookie)
	if status != http.StatusNoContent {
		t.Fatalf("revoke %d", status)
	}
	status, body = call(t, http.MethodGet, "/api/guilds/"+harness.GuildID+"/cases", nil, stack.Cookie)
	must(t, status, body, http.StatusOK)
	if !strings.Contains(string(body), "escalate") || !strings.Contains(string(body), harness.TargetName) {
		t.Fatalf("cases %s", body)
	}

	status, body = call(t, http.MethodPut, "/api/guilds/"+harness.GuildID+"/gate", map[string]any{
		"enabled": true, "deadlineSeconds": 86400, "warnBeforeSeconds": 3600,
		"kickReason":    "Did not finish the server application in time",
		"dmMessage":     "Welcome to {server}. You have {duration}.",
		"questions":     []string{"Where did you find us?"},
		"exemptRoleIds": []string{harness.ModRoleID},
		"logChannelId":  harness.GeneralChannelID,
	}, stack.Cookie)
	must(t, status, body, http.StatusOK)
	if asMap(t, body)["enabled"] != true {
		t.Fatalf("gate %s", body)
	}
	status, body = call(t, http.MethodGet, "/api/guilds/"+harness.GuildID+"/gate/members", nil, stack.Cookie)
	must(t, status, body, http.StatusOK)
	status, _ = call(t, http.MethodPost, "/api/guilds/"+harness.GuildID+"/gate/panel", map[string]any{
		"channelId": harness.GeneralChannelID,
	}, stack.Cookie)
	if status != http.StatusNoContent {
		t.Fatalf("gate panel %d", status)
	}

	status, body = call(t, http.MethodPost, "/api/guilds/"+harness.GuildID+"/channels", map[string]any{
		"name": "Mod Logs", "purpose": "logging",
	}, stack.Cookie)
	must(t, status, body, http.StatusCreated)
	if asMap(t, body)["name"] != "mod-logs" {
		t.Fatalf("channel %s", body)
	}
	status, body = call(t, http.MethodPost, "/api/guilds/"+harness.GuildID+"/roles", map[string]any{
		"name": "Helpers",
	}, stack.Cookie)
	must(t, status, body, http.StatusCreated)
	helperID, _ := asMap(t, body)["id"].(string)
	if asMap(t, body)["name"] != "Helpers" || helperID == "" {
		t.Fatalf("role %s", body)
	}

	status, body = call(t, http.MethodPut, "/api/guilds/"+harness.GuildID+"/reaction-roles/channel", map[string]any{
		"channelId": harness.GeneralChannelID,
	}, stack.Cookie)
	must(t, status, body, http.StatusOK)
	status, body = call(t, http.MethodPost, "/api/guilds/"+harness.GuildID+"/reaction-roles", map[string]any{
		"emoji": "🔥", "roleId": harness.MemberRoleID,
	}, stack.Cookie)
	must(t, status, body, http.StatusCreated)
	reactionID := int(asMap(t, body)["id"].(float64))
	status, body = call(t, http.MethodGet, "/api/guilds/"+harness.GuildID+"/reaction-roles", nil, stack.Cookie)
	must(t, status, body, http.StatusOK)
	if !strings.Contains(string(body), "🔥") {
		t.Fatalf("reactions %s", body)
	}
	status, _ = call(t, http.MethodDelete, "/api/guilds/"+harness.GuildID+"/reaction-roles/"+strconv.Itoa(reactionID), nil, stack.Cookie)
	if status != http.StatusNoContent {
		t.Fatalf("delete reaction %d", status)
	}

	status, body = call(t, http.MethodGet, "/api/guilds/"+harness.GuildID+"/members?q=fern", nil, stack.Cookie)
	must(t, status, body, http.StatusOK)
	if !strings.Contains(string(body), harness.TargetUserID) || strings.Contains(string(body), harness.UserID) {
		t.Fatalf("members %s", body)
	}
}

func putProfile(t *testing.T, nick string) (int, []byte) {
	t.Helper()
	var buf bytes.Buffer
	form := multipart.NewWriter(&buf)
	if err := form.WriteField("nick", nick); err != nil {
		t.Fatal(err)
	}
	if err := form.Close(); err != nil {
		t.Fatal(err)
	}
	req := request(t, http.MethodPut, "/api/guilds/"+harness.GuildID+"/profile", nil, stack.Cookie)
	req.Body = io.NopCloser(&buf)
	req.ContentLength = int64(buf.Len())
	req.Header.Set("Content-Type", form.FormDataContentType())
	res := do(t, req)
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return res.StatusCode, body
}

func call(t *testing.T, method, path string, payload any, cookie string) (int, []byte) {
	t.Helper()
	res := do(t, request(t, method, path, payload, cookie))
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return res.StatusCode, body
}

func raw(t *testing.T, method, path string, payload any, cookie string) *http.Response {
	t.Helper()
	return do(t, request(t, method, path, payload, cookie))
}

func noFollow(t *testing.T, method, path string, payload any, cookie string) *http.Response {
	t.Helper()
	client := &http.Client{CheckRedirect: stopRedirect, Timeout: 10 * time.Second}
	res, err := client.Do(request(t, method, path, payload, cookie))
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func request(t *testing.T, method, path string, payload any, cookie string) *http.Request {
	t.Helper()
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, stack.BaseURL+path, body)
	if err != nil {
		t.Fatal(err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Origin", stack.Origin)
	if cookie != "" {
		req.AddCookie(&http.Cookie{Name: "sb_session", Value: cookie})
	}
	return req
}

func do(t *testing.T, req *http.Request) *http.Response {
	t.Helper()
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func must(t *testing.T, status int, body []byte, want int) {
	t.Helper()
	if status != want {
		t.Fatalf("status %d, want %d, body %s", status, want, body)
	}
}

func asMap(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("json: %v body %s", err, body)
	}
	return out
}

func stopRedirect(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}
