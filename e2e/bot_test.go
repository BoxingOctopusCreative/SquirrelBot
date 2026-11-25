//go:build e2e

package e2e_test

import (
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
	opened := modalInteraction(t, domain.TicketCustomID("form", 0), harness.GeneralChannelID, member,
		textRow("subject", "Map pin"),
		textRow("details", "The pin is missing."),
	)
	stack.Bot.HandleInteraction(opened)
	reply, ok := stack.Discord.InteractionContent(opened.ID)
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
	closed := modalInteraction(t, domain.TicketCustomID("close-form", ticketID), channelID, member, textRow("reason", "Fixed the pin."))
	stack.Bot.HandleInteraction(closed)
	reply, ok = stack.Discord.InteractionContent(closed.ID)
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

func modalInteraction(t *testing.T, customID, channelID string, member *discordgo.Member, rows ...discordgo.MessageComponent) *discordgo.InteractionCreate {
	t.Helper()
	n := interactionSeq.Add(1)
	return &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		ID:        fmt.Sprintf("9100000000000%05d", n),
		AppID:     harness.BotUserID,
		Token:     fmt.Sprintf("token-%d", n),
		Type:      discordgo.InteractionModalSubmit,
		GuildID:   harness.GuildID,
		ChannelID: channelID,
		Member:    member,
		Data: discordgo.ModalSubmitInteractionData{
			CustomID:   customID,
			Components: rows,
		},
	}}
}

func textRow(customID, value string) discordgo.MessageComponent {
	return &discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		&discordgo.TextInput{CustomID: customID, Value: value},
	}}
}

func assertReply(t *testing.T, interactionID, want string) {
	t.Helper()
	got, ok := stack.Discord.InteractionContent(interactionID)
	if !ok || got != want {
		t.Fatalf("reply %q, want %q", got, want)
	}
}
