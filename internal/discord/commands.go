package discord

import (
	"bytes"
	"encoding/json"

	"github.com/bwmarrin/discordgo"
)

func commandDefs() []*discordgo.ApplicationCommand {
	dm := false
	user := func(name, desc string, required bool) *discordgo.ApplicationCommandOption {
		return &discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionUser, Name: name, Description: desc, Required: required}
	}
	text := func(name, desc string, required bool) *discordgo.ApplicationCommandOption {
		return &discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionString, Name: name, Description: desc, Required: required, MaxLength: 400}
	}
	channel := func(name, desc string, required bool) *discordgo.ApplicationCommandOption {
		return &discordgo.ApplicationCommandOption{
			Type: discordgo.ApplicationCommandOptionChannel, Name: name, Description: desc, Required: required,
			ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildText, discordgo.ChannelTypeGuildNews},
		}
	}
	role := func(name, desc string, required bool) *discordgo.ApplicationCommandOption {
		return &discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionRole, Name: name, Description: desc, Required: required}
	}
	integer := func(name, desc string, required bool, min, max float64) *discordgo.ApplicationCommandOption {
		minCopy := min
		return &discordgo.ApplicationCommandOption{
			Type: discordgo.ApplicationCommandOptionInteger, Name: name, Description: desc, Required: required,
			MinValue: &minCopy, MaxValue: max,
		}
	}
	cmd := func(name, desc string, options ...*discordgo.ApplicationCommandOption) *discordgo.ApplicationCommand {
		if options == nil {
			options = []*discordgo.ApplicationCommandOption{}
		}
		return &discordgo.ApplicationCommand{
			Name: name, Description: desc, DMPermission: &dm, Options: options,
		}
	}
	return []*discordgo.ApplicationCommand{
		cmd("warn", "Warn a member and record a case",
			user("user", "Member to warn", true),
			text("reason", "Why you are warning them", true)),
		cmd("kick", "Remove a member from the server",
			user("user", "Member to kick", true),
			text("reason", "Why you are kicking them", true)),
		cmd("ban", "Ban a member from the server",
			user("user", "Member to ban", true),
			text("reason", "Why you are banning them", true),
			integer("delete_days", "Days of their messages to delete", false, 0, 7)),
		cmd("timeout", "Stop a member from chatting for a while",
			user("user", "Member to time out", true),
			text("duration", "How long, such as 30m, 2h, or 1d", true),
			text("reason", "Why you are timing them out", true)),
		cmd("untimeout", "Remove a member's timeout",
			user("user", "Member to restore", true),
			text("reason", "Why you are ending the timeout", false)),
		cmd("cases", "Show recent cases for a member",
			user("user", "Member to look up", true)),
		cmd("purge", "Delete recent messages in this channel",
			integer("count", "How many messages, up to 100", true, 1, 100)),
		cmd("slowmode", "Set how slowly people can talk in this channel",
			integer("seconds", "Seconds between messages. 0 turns slowmode off", true, 0, 21600)),
		cmd("announce", "Post or schedule a mod team announcement",
			channel("channel", "Where to post it", true),
			text("title", "Announcement title", true),
			&discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionString, Name: "body", Description: "Announcement text", Required: true, MaxLength: 4000},
			role("ping", "Role to mention", false),
			&discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionString, Name: "in", Description: "Delay, such as 2h. Leave empty to send now", Required: false, MaxLength: 20}),
		cmd("escalate", "Grant a mod or admin role until it expires",
			user("user", "Member to promote", true),
			role("role", "Admin firecall role from the allowlist", true),
			text("duration", "How long, such as 4h, 2d, or 1w", true),
			text("reason", "Why they need the role", true)),
		cmd("deescalate", "Remove a temporary mod or admin role now",
			user("user", "Member to demote", true),
			role("role", "Admin firecall role to remove", true),
			text("reason", "Why you are ending it early", false)),
		cmd("escalations", "List active admin firecall access"),
		cmd("gate", "Check the application deadline or post the application message",
			&discordgo.ApplicationCommandOption{
				Type: discordgo.ApplicationCommandOptionString, Name: "action", Description: "What to do", Required: true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "Status", Value: "status"},
					{Name: "Post the message", Value: "panel"},
				},
			},
			channel("channel", "Channel for the message. Required for Post the message", false)),
	}
}

// commandPayload drops null option fields Discord rejects on command registration.
func commandPayload() ([]byte, error) {
	body, err := json.Marshal(commandDefs())
	if err != nil {
		return nil, err
	}
	for _, field := range []string{"channel_types", "choices", "options"} {
		body = bytes.ReplaceAll(body, []byte(`,"`+field+`":null`), nil)
		body = bytes.ReplaceAll(body, []byte(`"`+field+`":null,`), nil)
	}
	return body, nil
}
