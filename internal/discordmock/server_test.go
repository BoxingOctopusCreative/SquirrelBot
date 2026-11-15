package discordmock

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
)

func TestSessionTalksToTheMock(t *testing.T) {
	const (
		guildID   = "111111111111111111"
		channelID = "777777777777777771"
		userID    = "444444444444444444"
		roleID    = "888888888888888881"
	)
	mock := New()
	mock.SetOAuth("access-token", "refresh-token")
	mock.AddToken("access-token", User{ID: userID, Username: "acorn", GlobalName: "Acorn"}, []UserGuild{
		{ID: guildID, Name: "Canopy", Owner: true, Permissions: "32"},
	})
	mock.AddGuild(guildID, "Canopy", "666666666666666666")
	mock.AddRole(guildID, guildID, "@everyone", 0, false)
	mock.AddRole(guildID, roleID, "Members", 10, false)
	mock.AddChannel(guildID, channelID, "general", int(discordgo.ChannelTypeGuildText))
	mock.AddMember(guildID, Member{ID: userID, Username: "acorn", GlobalName: "Acorn", Roles: []string{roleID}})

	server := httptest.NewServer(mock)
	t.Cleanup(server.Close)
	dest, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	session, err := discordgo.New("Bot test-token")
	if err != nil {
		t.Fatal(err)
	}
	session.Client = &http.Client{Transport: NewTransport(nil, dest)}

	sent, err := session.ChannelMessageSend(channelID, "hello from the mock")
	if err != nil {
		t.Fatal(err)
	}
	if sent.ID == "" || sent.Content != "hello from the mock" {
		t.Fatalf("sent %#v", sent)
	}
	listed, err := session.ChannelMessages(channelID, 10, "", "", "")
	if err != nil || len(listed) != 1 || listed[0].ID != sent.ID {
		t.Fatalf("listed %v %v", err, listed)
	}

	seconds := 15
	edited, err := session.ChannelEdit(channelID, &discordgo.ChannelEdit{RateLimitPerUser: &seconds})
	if err != nil || edited.RateLimitPerUser != 15 {
		t.Fatalf("slowmode %v %#v", err, edited)
	}
	channel, ok := mock.Channel(channelID)
	if !ok || channel.RateLimitPerUser != 15 {
		t.Fatalf("stored channel %#v %v", channel, ok)
	}

	if err := session.GuildMemberRoleAdd(guildID, userID, roleID); err != nil {
		t.Fatal(err)
	}
	interaction := &discordgo.Interaction{ID: "900000000000000001", Token: "callback-token", AppID: "555555555555555555"}
	err = session.InteractionRespond(interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		t.Fatal(err)
	}
	content := "Case #1 · kick"
	if _, err := session.InteractionResponseEdit(interaction, &discordgo.WebhookEdit{Content: &content}); err != nil {
		t.Fatal(err)
	}
	got, ok := mock.InteractionContent(interaction.ID)
	if !ok || got != content {
		t.Fatalf("reply %q %v", got, ok)
	}

	if err := session.GuildMemberDeleteWithReason(guildID, userID, "Broke the rules"); err != nil {
		t.Fatal(err)
	}
	if _, ok := mock.Member(guildID, userID); ok {
		t.Fatal("kicked member is still in the guild")
	}

	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/oauth2/token", strings.NewReader("grant_type=authorization_code"))
	if err != nil {
		t.Fatal(err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("token status %d", res.StatusCode)
	}
	me, err := http.NewRequest(http.MethodGet, server.URL+"/api/users/@me", nil)
	if err != nil {
		t.Fatal(err)
	}
	me.Header.Set("Authorization", "Bearer access-token")
	res, err = http.DefaultClient.Do(me)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("me status %d", res.StatusCode)
	}

	until := time.Now().Add(10 * time.Minute)
	mock.AddMember(guildID, Member{ID: userID, Username: "acorn", Roles: []string{roleID}})
	if err := session.GuildMemberTimeout(guildID, userID, &until); err != nil {
		t.Fatal(err)
	}
	member, ok := mock.Member(guildID, userID)
	if !ok || member.Timeout == nil || member.Timeout.Before(time.Now()) {
		t.Fatalf("timeout %#v", member)
	}
}
