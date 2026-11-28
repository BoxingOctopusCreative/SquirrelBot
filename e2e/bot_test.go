//go:build e2e

package e2e_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/BoxingOctopus/squirrelbot/e2e/harness"
	"github.com/BoxingOctopus/squirrelbot/internal/discordmock"
	"github.com/BoxingOctopus/squirrelbot/internal/domain"
	"github.com/bwmarrin/discordgo"
)

var interactionSeq atomic.Int64

func TestBotCommandsUseTheDiscordMock(t *testing.T) {
	t.Run("unknown command", func(t *testing.T) {
		interaction := moderatorCommand(t, "dance", harness.GeneralChannelID)
		stack.Bot.HandleInteraction(interaction)
		assertReply(t, interaction.ID, "I don't know that command.")
	})

	t.Run("missing permission", func(t *testing.T) {
		interaction := commandInteraction(t, "kick", harness.GeneralChannelID, &discordgo.Member{
			User:  &discordgo.User{ID: harness.GuestUserID, Username: "guest"},
			Roles: []string{harness.MemberRoleID},
		})
		stack.Bot.HandleInteraction(interaction)
		assertReply(t, interaction.ID, "You don't have permission to do that.")
	})

	t.Run("kick", func(t *testing.T) {
		const userID = "444444444444444451"
		stack.Discord.AddMember(harness.GuildID, discordmock.Member{
			ID: userID, Username: "pebble", GlobalName: "Pebble", Nick: "Pebble",
			Roles: []string{harness.MemberRoleID},
		})
		interaction := moderatorCommand(t, "kick", harness.GeneralChannelID,
			userOption(userID),
			stringOption("reason", "Broke the canopy rules"),
		)
		stack.Bot.HandleInteraction(interaction)
		reply, ok := stack.Discord.InteractionContent(interaction.ID)
		if !ok || !strings.Contains(reply, "kick") || !strings.Contains(reply, "Broke the canopy rules") {
			t.Fatalf("reply %q", reply)
		}
		if _, still := stack.Discord.Member(harness.GuildID, userID); still {
			t.Fatal("kick left the member in the guild")
		}
		status, body := call(t, http.MethodGet, "/api/guilds/"+harness.GuildID+"/cases", nil, stack.Cookie)
		must(t, status, body, http.StatusOK)
		if !strings.Contains(string(body), "kick") || !strings.Contains(string(body), "Broke the canopy rules") || !strings.Contains(string(body), "Pebble") {
			t.Fatalf("cases %s", body)
		}
	})

	t.Run("timeout", func(t *testing.T) {
		const userID = "444444444444444452"
		stack.Discord.AddMember(harness.GuildID, discordmock.Member{
			ID: userID, Username: "chip", GlobalName: "Chip", Nick: "Chip",
			Roles: []string{harness.MemberRoleID},
		})
		interaction := moderatorCommand(t, "timeout", harness.GeneralChannelID,
			userOption(userID),
			stringOption("reason", "Cooling off"),
			stringOption("duration", "10m"),
		)
		stack.Bot.HandleInteraction(interaction)
		reply, ok := stack.Discord.InteractionContent(interaction.ID)
		if !ok || !strings.Contains(reply, "timeout") || !strings.Contains(reply, "Cooling off") {
			t.Fatalf("reply %q", reply)
		}
		member, still := stack.Discord.Member(harness.GuildID, userID)
		if !still || member.Timeout == nil || !member.Timeout.After(time.Now().Add(5*time.Minute)) {
			t.Fatalf("timeout %#v", member)
		}
	})

	t.Run("slowmode", func(t *testing.T) {
		channelID := "777777777777777781"
		stack.Discord.AddChannel(harness.GuildID, channelID, "slow", int(discordgo.ChannelTypeGuildText))
		interaction := moderatorCommand(t, "slowmode", channelID, intOption("seconds", 20))
		stack.Bot.HandleInteraction(interaction)
		assertReply(t, interaction.ID, "Slowmode is 20 seconds in this channel.")
		channel, ok := stack.Discord.Channel(channelID)
		if !ok || channel.RateLimitPerUser != 20 {
			t.Fatalf("channel %#v", channel)
		}
	})

	t.Run("purge", func(t *testing.T) {
		channelID := "777777777777777782"
		stack.Discord.AddChannel(harness.GuildID, channelID, "sweep", int(discordgo.ChannelTypeGuildText))
		first := stack.Discord.AddMessage(channelID, harness.TargetUserID, "one")
		second := stack.Discord.AddMessage(channelID, harness.TargetUserID, "two")
		third := stack.Discord.AddMessage(channelID, harness.TargetUserID, "three")
		interaction := moderatorCommand(t, "purge", channelID, intOption("count", 3))
		stack.Bot.HandleInteraction(interaction)
		assertReply(t, interaction.ID, "Deleted 3 messages.")
		left := map[string]bool{}
		for _, message := range stack.Discord.Messages(channelID) {
			left[message.ID] = true
		}
		if left[first] || left[second] || left[third] {
			t.Fatalf("purge left messages %#v", left)
		}
	})
}

func TestBotMessagesUseTheDiscordMock(t *testing.T) {
	t.Run("automod deletes an invite", func(t *testing.T) {
		status, body := call(t, http.MethodPut, "/api/guilds/"+harness.GuildID+"/automod", map[string]any{
			"enabled": true, "blockInvites": true, "blockSpam": false,
			"mentionLimit": 5, "capsPercent": 70, "capsMinLength": 12,
			"badWords": []string{}, "action": "delete", "timeoutSeconds": 60,
			"ignoredChannelIds": []string{}, "ignoredRoleIds": []string{},
		}, stack.Cookie)
		must(t, status, body, http.StatusOK)

		messageID := stack.Discord.AddMessage(harness.GeneralChannelID, "444444444444444453", "come to discord.gg/acorns")
		stack.Bot.HandleMessage(&discordgo.MessageCreate{Message: &discordgo.Message{
			ID: messageID, GuildID: harness.GuildID, ChannelID: harness.GeneralChannelID,
			Content: "come to discord.gg/acorns",
			Author:  &discordgo.User{ID: "444444444444444453", Username: "pebble"},
			Member:  &discordgo.Member{Roles: []string{harness.MemberRoleID}},
		}})
		for _, message := range stack.Discord.Messages(harness.GeneralChannelID) {
			if message.ID == messageID {
				t.Fatal("automod left the invite in the channel")
			}
		}
	})

	t.Run("custom command replies", func(t *testing.T) {
		setPrefix(t, "!")
		status, body := call(t, http.MethodPost, "/api/guilds/"+harness.GuildID+"/commands", map[string]any{
			"name": "wave", "response": `reply("nuts")`, "roleIds": []string{},
		}, stack.Cookie)
		must(t, status, body, http.StatusCreated)

		stack.Bot.HandleMessage(&discordgo.MessageCreate{Message: &discordgo.Message{
			ID: "910000000000009999", GuildID: harness.GuildID, ChannelID: harness.GeneralChannelID,
			Content: "!wave",
			Author:  &discordgo.User{ID: harness.TargetUserID, Username: harness.TargetUsername},
			Member:  &discordgo.Member{Roles: []string{harness.MemberRoleID}},
		}})
		for _, message := range stack.Discord.Messages(harness.GeneralChannelID) {
			if message.Content == "nuts" {
				return
			}
		}
		t.Fatal("custom command did not reply")
	})
}

func TestLeaveMessage(t *testing.T) {
	status, body := call(t, http.MethodGet, "/api/guilds/"+harness.GuildID, nil, stack.Cookie)
	must(t, status, body, http.StatusOK)
	guild := asMap(t, body)["guild"].(map[string]any)
	guild["leaveChannelId"] = harness.GeneralChannelID
	guild["leaveMessage"] = "Goodbye {username}"
	status, body = call(t, http.MethodPatch, "/api/guilds/"+harness.GuildID, guild, stack.Cookie)
	must(t, status, body, http.StatusOK)

	const userID = "444444444444444454"
	stack.Bot.HandleMemberRemove(&discordgo.GuildMemberRemove{
		Member: &discordgo.Member{
			GuildID: harness.GuildID,
			User:    &discordgo.User{ID: userID, Username: "hazel"},
		},
	})
	for _, message := range stack.Discord.Messages(harness.GeneralChannelID) {
		if message.Content == "Goodbye hazel" {
			return
		}
	}
	t.Fatal("leave message was not posted")
}

func setPrefix(t *testing.T, prefix string) {
	t.Helper()
	status, body := call(t, http.MethodGet, "/api/guilds/"+harness.GuildID, nil, stack.Cookie)
	must(t, status, body, http.StatusOK)
	guild := asMap(t, body)["guild"].(map[string]any)
	guild["prefix"] = prefix
	status, body = call(t, http.MethodPatch, "/api/guilds/"+harness.GuildID, guild, stack.Cookie)
	must(t, status, body, http.StatusOK)
}

func TestOpenTicketSurvivesMissingPanelChannel(t *testing.T) {
	status, body := call(t, http.MethodPost, "/api/guilds/"+harness.GuildID+"/channels", map[string]any{
		"name": "Panel move", "purpose": "ticket-category",
	}, stack.Cookie)
	must(t, status, body, http.StatusCreated)
	categoryID, _ := asMap(t, body)["id"].(string)
	status, body = call(t, http.MethodPut, "/api/guilds/"+harness.GuildID+"/tickets", map[string]any{
		"panelChannelId": "",
		"categoryId":     categoryID,
		"supportRoleId":  harness.ModRoleID,
		"panelTitle":     "Help",
		"panelBody":      "Tell us what you need.",
	}, stack.Cookie)
	must(t, status, body, http.StatusOK)

	member := &discordgo.Member{
		User:        &discordgo.User{ID: "444444444444444461", Username: "birch"},
		Roles:       []string{harness.AdminRoleID},
		Permissions: domain.PermAdministrator,
	}
	click := componentInteraction(t, domain.TicketCustomID("open", 0), harness.AnnounceChannelID, member)
	click.Message = &discordgo.Message{ID: "777777777777777780", ChannelID: harness.AnnounceChannelID}
	stack.Bot.HandleInteraction(click)
	reply, ok := stack.Discord.InteractionContent(click.ID)
	if !ok || reply != "" {
		t.Fatalf("open form %q", reply)
	}
	status, body = call(t, http.MethodGet, "/api/guilds/"+harness.GuildID, nil, stack.Cookie)
	must(t, status, body, http.StatusOK)
	tickets := asMap(t, body)["tickets"].(map[string]any)
	if tickets["panelChannelId"] != harness.AnnounceChannelID {
		t.Fatalf("panel channel %v", tickets["panelChannelId"])
	}

	moved := componentInteraction(t, domain.TicketCustomID("open", 0), harness.GeneralChannelID, member)
	moved.Message = &discordgo.Message{ID: "777777777777777781", ChannelID: harness.GeneralChannelID}
	stack.Bot.HandleInteraction(moved)
	reply, ok = stack.Discord.InteractionContent(moved.ID)
	if !ok || reply != "" {
		t.Fatalf("moved form %q", reply)
	}
	status, body = call(t, http.MethodGet, "/api/guilds/"+harness.GuildID, nil, stack.Cookie)
	must(t, status, body, http.StatusOK)
	tickets = asMap(t, body)["tickets"].(map[string]any)
	if tickets["panelChannelId"] != harness.GeneralChannelID {
		t.Fatalf("moved panel channel %v", tickets["panelChannelId"])
	}
}

func TestDeleteButtonRemovesClosedTicketChannel(t *testing.T) {
	status, body := call(t, http.MethodPost, "/api/guilds/"+harness.GuildID+"/channels", map[string]any{
		"name": "Delete test", "purpose": "ticket-category",
	}, stack.Cookie)
	must(t, status, body, http.StatusCreated)
	categoryID, _ := asMap(t, body)["id"].(string)
	status, body = call(t, http.MethodPut, "/api/guilds/"+harness.GuildID+"/tickets", map[string]any{
		"panelChannelId": harness.GeneralChannelID,
		"categoryId":     categoryID,
		"supportRoleId":  harness.ModRoleID,
		"logChannelId":   harness.AnnounceChannelID,
		"panelTitle":     "Help",
		"panelBody":      "Tell us what you need.",
	}, stack.Cookie)
	must(t, status, body, http.StatusOK)

	member := &discordgo.Member{
		User:        &discordgo.User{ID: harness.UserID, Username: "acorn", GlobalName: harness.UserName},
		Roles:       []string{harness.AdminRoleID},
		Permissions: domain.PermAdministrator,
	}
	openedID, opened := labelModalEvent(t, domain.TicketCustomID("form", 0), harness.GeneralChannelID, member, map[string]string{
		"subject": "Map pin",
		"details": "The pin is missing.",
	})
	stack.Bot.HandleGatewayEvent(opened)
	reply, ok := stack.Discord.InteractionContent(openedID)
	if !ok || !strings.Contains(reply, "Opened ticket") {
		t.Fatalf("open reply %q", reply)
	}

	status, body = call(t, http.MethodGet, "/api/guilds/"+harness.GuildID+"/tickets", nil, stack.Cookie)
	must(t, status, body, http.StatusOK)
	openItems, _ := asMap(t, body)["open"].([]any)
	if len(openItems) != 1 {
		t.Fatalf("open tickets %s", body)
	}
	item := openItems[0].(map[string]any)
	ticketID := int64(item["id"].(float64))
	channelID, _ := item["channelId"].(string)
	if channelID == "" {
		t.Fatal("ticket has no channel")
	}
	messages := stack.Discord.Messages(channelID)
	if len(messages) == 0 {
		t.Fatal("ticket channel has no control message")
	}
	claim := componentInteraction(t, domain.TicketCustomID("claim", ticketID), channelID, member)
	claim.Message = &discordgo.Message{ID: messages[0].ID, ChannelID: channelID}
	stack.Bot.HandleInteraction(claim)
	reply, ok = stack.Discord.InteractionContent(claim.ID)
	if !ok || !strings.Contains(reply, "Claimed by <@"+harness.UserID+">") {
		t.Fatalf("claim message %q", reply)
	}
	status, body = call(t, http.MethodGet, "/api/guilds/"+harness.GuildID+"/tickets", nil, stack.Cookie)
	must(t, status, body, http.StatusOK)
	openItems, _ = asMap(t, body)["open"].([]any)
	if len(openItems) != 1 || openItems[0].(map[string]any)["claimerName"] == "" {
		t.Fatalf("dashboard claim %s", body)
	}

	stack.Bot.HandleInteraction(componentInteraction(t, domain.TicketCustomID("close", ticketID), channelID, member))
	closedID, closed := labelModalEvent(t, domain.TicketCustomID("close-form", ticketID), channelID, member, map[string]string{
		"reason": "Fixed the pin.",
	})
	stack.Bot.HandleGatewayEvent(closed)
	reply, ok = stack.Discord.InteractionContent(closedID)
	if !ok || !strings.Contains(reply, "will be deleted in 10 seconds") {
		t.Fatalf("close reply %q", reply)
	}

	stack.Bot.HandleInteraction(componentInteraction(t, domain.TicketCustomID("delete", ticketID), channelID, member))
	if _, found := stack.Discord.Channel(channelID); found {
		t.Fatal("delete left the ticket channel")
	}
	status, body = call(t, http.MethodGet, "/api/guilds/"+harness.GuildID+"/tickets", nil, stack.Cookie)
	must(t, status, body, http.StatusOK)
	for _, raw := range asMap(t, body)["closed"].([]any) {
		row := raw.(map[string]any)
		if int64(row["id"].(float64)) == ticketID && row["status"] == "deleted" {
			return
		}
	}
	t.Fatalf("closed tickets %s", body)
}

func TestLevels(t *testing.T) {
	status, body := call(t, http.MethodPut, "/api/guilds/"+harness.GuildID+"/levels", map[string]any{
		"enabled": true, "xpMin": 20, "xpMax": 20, "rateCount": 1, "rateSeconds": 60,
		"announce": "off", "announceMinLevel": 0, "announceMultiple": 0,
		"ignoredChannelIds": []string{}, "ignoredRoleIds": []string{},
		"rewards": []map[string]any{{"level": 1, "roleId": harness.AdminRoleID}},
	}, stack.Cookie)
	must(t, status, body, http.StatusOK)

	const userID = "444444444444444461"
	stack.Discord.AddMember(harness.GuildID, discordmock.Member{
		ID: userID, Username: "pebble", GlobalName: "Pebble", Nick: "Pebble",
		Roles: []string{harness.MemberRoleID},
	})
	stack.Bot.HandleMessage(&discordgo.MessageCreate{Message: &discordgo.Message{
		ID: "level-msg-1", GuildID: harness.GuildID, ChannelID: harness.GeneralChannelID,
		Content: "hello canopy",
		Author:  &discordgo.User{ID: userID, Username: "pebble", GlobalName: "Pebble"},
		Member:  &discordgo.Member{Nick: "Pebble", Roles: []string{harness.MemberRoleID}},
	}})

	status, body = call(t, http.MethodGet, "/api/guilds/"+harness.GuildID+"/levels", nil, stack.Cookie)
	must(t, status, body, http.StatusOK)
	board := asMap(t, body)["leaderboard"].([]any)
	if len(board) != 1 {
		t.Fatalf("leaderboard %s", body)
	}
	row := board[0].(map[string]any)
	if row["userId"] != userID || row["xp"] != float64(20) || row["level"] != float64(0) || row["userName"] != "Pebble" {
		t.Fatalf("standing %s", body)
	}

	interaction := levelAdd(t, userID, 80)
	stack.Bot.HandleInteraction(interaction)
	assertReply(t, interaction.ID, "Added 80 XP. Pebble is level 1 with 100 XP.")

	member, ok := stack.Discord.Member(harness.GuildID, userID)
	if !ok || !containsRole(member.Roles, harness.AdminRoleID) {
		t.Fatalf("reward role missing: %+v", member.Roles)
	}

	status, body = call(t, http.MethodGet, "/api/guilds/"+harness.GuildID+"/levels?page=1", nil, stack.Cookie)
	must(t, status, body, http.StatusOK)
	row = asMap(t, body)["leaderboard"].([]any)[0].(map[string]any)
	if row["xp"] != float64(100) || row["level"] != float64(1) || row["rank"] != float64(1) {
		t.Fatalf("after add %s", body)
	}
	events := asMap(t, body)["events"].([]any)
	if len(events) != 1 || events[0].(map[string]any)["level"] != float64(1) {
		t.Fatalf("events %s", body)
	}
}

func levelAdd(t *testing.T, userID string, amount int) *discordgo.InteractionCreate {
	t.Helper()
	interaction := moderatorCommand(t, "level", harness.GeneralChannelID, &discordgo.ApplicationCommandInteractionDataOption{
		Name: "add", Type: discordgo.ApplicationCommandOptionSubCommand,
		Options: []*discordgo.ApplicationCommandInteractionDataOption{
			userOption(userID),
			intOption("amount", amount),
		},
	})
	data := interaction.ApplicationCommandData()
	data.Resolved = &discordgo.ApplicationCommandInteractionDataResolved{
		Users: map[string]*discordgo.User{
			userID: {ID: userID, Username: "pebble", GlobalName: "Pebble"},
		},
		Members: map[string]*discordgo.Member{
			userID: {Nick: "Pebble", Roles: []string{harness.MemberRoleID}, User: &discordgo.User{ID: userID, GlobalName: "Pebble"}},
		},
	}
	interaction.Data = data
	return interaction
}

func containsRole(roles []string, id string) bool {
	for _, role := range roles {
		if role == id {
			return true
		}
	}
	return false
}

func moderatorCommand(t *testing.T, name, channelID string, options ...*discordgo.ApplicationCommandInteractionDataOption) *discordgo.InteractionCreate {
	t.Helper()
	return commandInteraction(t, name, channelID, &discordgo.Member{
		User:        &discordgo.User{ID: harness.UserID, Username: "acorn", GlobalName: harness.UserName},
		Roles:       []string{harness.AdminRoleID},
		Permissions: domain.PermAdministrator,
	}, options...)
}

func commandInteraction(t *testing.T, name, channelID string, member *discordgo.Member, options ...*discordgo.ApplicationCommandInteractionDataOption) *discordgo.InteractionCreate {
	t.Helper()
	n := interactionSeq.Add(1)
	return &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		ID:        fmt.Sprintf("9100000000000%05d", n),
		AppID:     harness.BotUserID,
		Token:     fmt.Sprintf("token-%d", n),
		Type:      discordgo.InteractionApplicationCommand,
		GuildID:   harness.GuildID,
		ChannelID: channelID,
		Member:    member,
		Data: discordgo.ApplicationCommandInteractionData{
			Name:    name,
			Options: options,
		},
	}}
}

func userOption(id string) *discordgo.ApplicationCommandInteractionDataOption {
	return &discordgo.ApplicationCommandInteractionDataOption{
		Name: "user", Type: discordgo.ApplicationCommandOptionUser, Value: id,
	}
}

func stringOption(name, value string) *discordgo.ApplicationCommandInteractionDataOption {
	return &discordgo.ApplicationCommandInteractionDataOption{
		Name: name, Type: discordgo.ApplicationCommandOptionString, Value: value,
	}
}

func intOption(name string, value int) *discordgo.ApplicationCommandInteractionDataOption {
	return &discordgo.ApplicationCommandInteractionDataOption{
		Name: name, Type: discordgo.ApplicationCommandOptionInteger, Value: float64(value),
	}
}

func componentInteraction(t *testing.T, customID, channelID string, member *discordgo.Member) *discordgo.InteractionCreate {
	t.Helper()
	n := interactionSeq.Add(1)
	return &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		ID:        fmt.Sprintf("9100000000000%05d", n),
		AppID:     harness.BotUserID,
		Token:     fmt.Sprintf("token-%d", n),
		Type:      discordgo.InteractionMessageComponent,
		GuildID:   harness.GuildID,
		ChannelID: channelID,
		Member:    member,
		Data: discordgo.MessageComponentInteractionData{
			CustomID:      customID,
			ComponentType: discordgo.ButtonComponent,
		},
	}}
}

func labelModalEvent(t *testing.T, customID, channelID string, member *discordgo.Member, fields map[string]string) (string, *discordgo.Event) {
	t.Helper()
	n := interactionSeq.Add(1)
	id := fmt.Sprintf("9100000000000%05d", n)
	components := make([]map[string]any, 0, len(fields))
	for key, value := range fields {
		components = append(components, map[string]any{
			"type": 18,
			"component": map[string]any{
				"type": 4, "custom_id": key, "value": value,
			},
		})
	}
	raw, err := json.Marshal(map[string]any{
		"id": id, "application_id": harness.BotUserID, "type": 5,
		"guild_id": harness.GuildID, "channel_id": channelID,
		"token": fmt.Sprintf("token-%d", n), "member": member,
		"data": map[string]any{"custom_id": customID, "components": components},
	})
	if err != nil {
		t.Fatal(err)
	}
	return id, &discordgo.Event{Type: "INTERACTION_CREATE", RawData: raw}
}

func TestTimedPurgeRuns(t *testing.T) {
	ctx := t.Context()
	guild := "/api/guilds/" + harness.GuildID
	status, body := call(t, http.MethodPut, guild+"/purges/enabled", map[string]any{"enabled": true}, stack.Cookie)
	must(t, status, body, http.StatusOK)

	channelID := "777777777777777796"
	stack.Discord.AddChannel(harness.GuildID, channelID, "lobby", int(discordgo.ChannelTypeGuildText))
	member := stack.Discord.AddMessage(channelID, harness.TargetUserID, "member")
	botMessage := stack.Discord.AddMessageAt(channelID, harness.BotUserID, "bot", time.Now(), true, false)
	pinned := stack.Discord.AddMessageAt(channelID, harness.TargetUserID, "pin", time.Now(), false, true)
	old := stack.Discord.AddMessageAt(channelID, harness.TargetUserID, "old", time.Now().Add(-15*24*time.Hour), false, false)
	status, body = call(t, http.MethodPost, guild+"/purges", map[string]any{"channelId": channelID, "preset": "1h"}, stack.Cookie)
	must(t, status, body, http.StatusOK)
	if err := stack.MakePurgeDue(ctx, harness.GuildID, channelID); err != nil {
		t.Fatal(err)
	}
	_, casesBefore := call(t, http.MethodGet, guild+"/cases", nil, stack.Cookie)
	stack.Bot.RunDuePurges(ctx)
	left := map[string]bool{}
	for _, message := range stack.Discord.Messages(channelID) {
		left[message.ID] = true
	}
	if left[member] || left[botMessage] || left[pinned] || !left[old] || len(left) != 1 {
		t.Fatalf("messages after wipe %#v", left)
	}
	_, casesAfter := call(t, http.MethodGet, guild+"/cases", nil, stack.Cookie)
	if string(casesBefore) != string(casesAfter) {
		t.Fatalf("purge opened a case\nbefore %s\nafter %s", casesBefore, casesAfter)
	}
	status, body = call(t, http.MethodGet, guild+"/purges", nil, stack.Cookie)
	must(t, status, body, http.StatusOK)
	row := scheduleRow(t, body, channelID)
	if row["removed"] != float64(3) || row["skipped"] != float64(1) || row["error"] != "" {
		t.Fatalf("result %s", body)
	}

	wideID := "777777777777777797"
	stack.Discord.AddChannel(harness.GuildID, wideID, "wide", int(discordgo.ChannelTypeGuildText))
	for i := 0; i < 150; i++ {
		stack.Discord.AddMessage(wideID, harness.TargetUserID, fmt.Sprintf("m%d", i))
	}
	status, body = call(t, http.MethodPost, guild+"/purges", map[string]any{"channelId": wideID, "preset": "4h"}, stack.Cookie)
	must(t, status, body, http.StatusOK)
	if err := stack.MakePurgeDue(ctx, harness.GuildID, wideID); err != nil {
		t.Fatal(err)
	}
	stack.Bot.RunDuePurges(ctx)
	if got := len(stack.Discord.Messages(wideID)); got != 0 {
		t.Fatalf("wide channel left %d messages", got)
	}

	blockedID := "777777777777777798"
	stack.Discord.AddChannel(harness.GuildID, blockedID, "blocked", int(discordgo.ChannelTypeGuildText))
	kept := stack.Discord.AddMessage(blockedID, harness.TargetUserID, "stay")
	status, body = call(t, http.MethodPost, guild+"/purges", map[string]any{"channelId": blockedID, "preset": "1h"}, stack.Cookie)
	must(t, status, body, http.StatusOK)
	stack.Discord.ForbidChannel(blockedID)
	if err := stack.MakePurgeDue(ctx, harness.GuildID, blockedID); err != nil {
		t.Fatal(err)
	}
	stack.Bot.RunDuePurges(ctx)
	if len(stack.Discord.Messages(blockedID)) != 1 || stack.Discord.Messages(blockedID)[0].ID != kept {
		t.Fatal("forbidden channel lost a message")
	}
	status, body = call(t, http.MethodGet, guild+"/purges", nil, stack.Cookie)
	must(t, status, body, http.StatusOK)
	if scheduleRow(t, body, blockedID)["error"] == "" {
		t.Fatalf("missing failure %s", body)
	}
	stack.Bot.RunDuePurges(ctx)
	if len(stack.Discord.Messages(blockedID)) != 1 {
		t.Fatal("a second run deleted the forbidden channel")
	}

	goneID := "777777777777777799"
	stack.Discord.AddChannel(harness.GuildID, goneID, "gone", int(discordgo.ChannelTypeGuildText))
	status, body = call(t, http.MethodPost, guild+"/purges", map[string]any{"channelId": goneID, "preset": "1h"}, stack.Cookie)
	must(t, status, body, http.StatusOK)
	stack.Discord.RemoveChannel(goneID)
	if err := stack.MakePurgeDue(ctx, harness.GuildID, goneID); err != nil {
		t.Fatal(err)
	}
	stack.Bot.RunDuePurges(ctx)
	status, body = call(t, http.MethodGet, guild+"/purges", nil, stack.Cookie)
	must(t, status, body, http.StatusOK)
	if scheduleRow(t, body, goneID)["state"] != "missing" {
		t.Fatalf("missing channel %s", body)
	}
	stack.Bot.RunDuePurges(ctx)
	status, body = call(t, http.MethodGet, guild+"/purges", nil, stack.Cookie)
	if scheduleRow(t, body, goneID)["state"] != "missing" {
		t.Fatalf("missing channel was claimed again %s", body)
	}

	firstID := "777777777777777800"
	secondID := "777777777777777801"
	stack.Discord.AddChannel(harness.GuildID, firstID, "first", int(discordgo.ChannelTypeGuildText))
	stack.Discord.AddChannel(harness.GuildID, secondID, "second", int(discordgo.ChannelTypeGuildText))
	stack.Discord.AddMessage(firstID, harness.TargetUserID, "one")
	secondMessage := stack.Discord.AddMessage(secondID, harness.TargetUserID, "two")
	status, body = call(t, http.MethodPost, guild+"/purges", map[string]any{"channelId": firstID, "preset": "1h"}, stack.Cookie)
	must(t, status, body, http.StatusOK)
	status, body = call(t, http.MethodPost, guild+"/purges", map[string]any{"channelId": secondID, "preset": "4h"}, stack.Cookie)
	must(t, status, body, http.StatusOK)
	if err := stack.MakePurgeDue(ctx, harness.GuildID, firstID); err != nil {
		t.Fatal(err)
	}
	if err := stack.MakePurgeDue(ctx, harness.GuildID, secondID); err != nil {
		t.Fatal(err)
	}
	stack.Bot.RunDuePurges(ctx)
	if len(stack.Discord.Messages(firstID)) != 0 || len(stack.Discord.Messages(secondID)) != 1 {
		t.Fatalf("first %d second %d", len(stack.Discord.Messages(firstID)), len(stack.Discord.Messages(secondID)))
	}
	status, body = call(t, http.MethodDelete, guild+"/purges/"+secondID, nil, stack.Cookie)
	if status != http.StatusNoContent {
		t.Fatalf("remove second %d %s", status, body)
	}
	stack.Bot.RunDuePurges(ctx)
	if got := stack.Discord.Messages(secondID); len(got) != 1 || got[0].ID != secondMessage {
		t.Fatal("removed schedule still wiped")
	}

	status, body = call(t, http.MethodPut, guild+"/purges/enabled", map[string]any{"enabled": false}, stack.Cookie)
	must(t, status, body, http.StatusOK)
	pausedID := "777777777777777802"
	stack.Discord.AddChannel(harness.GuildID, pausedID, "paused", int(discordgo.ChannelTypeGuildText))
	stack.Discord.AddMessage(pausedID, harness.TargetUserID, "pause")
	status, body = call(t, http.MethodPost, guild+"/purges", map[string]any{"channelId": pausedID, "preset": "1h"}, stack.Cookie)
	must(t, status, body, http.StatusOK)
	if err := stack.MakePurgeDue(ctx, harness.GuildID, pausedID); err != nil {
		t.Fatal(err)
	}
	stack.Bot.RunDuePurges(ctx)
	if len(stack.Discord.Messages(pausedID)) != 1 {
		t.Fatal("a paused schedule wiped a channel")
	}
}

func scheduleRow(t *testing.T, body []byte, channelID string) map[string]any {
	t.Helper()
	for _, item := range asMap(t, body)["schedules"].([]any) {
		row := item.(map[string]any)
		if row["channelId"] == channelID {
			return row
		}
	}
	t.Fatalf("schedule %s missing from %s", channelID, body)
	return nil
}

func assertReply(t *testing.T, interactionID, want string) {
	t.Helper()
	got, ok := stack.Discord.InteractionContent(interactionID)
	if !ok || got != want {
		t.Fatalf("reply %q, want %q", got, want)
	}
}
