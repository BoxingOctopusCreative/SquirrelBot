package discord

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) onDelete(s *discordgo.Session, event *discordgo.MessageDelete) {
	if event.GuildID == "" || b.muted.seen(event.ID) {
		return
	}
	ctx, cancel := contextTimeout()
	defer cancel()
	logs, err := b.store.GetLogConfig(ctx, event.GuildID)
	if err != nil || !logs.MessageDelete {
		return
	}
	channel := b.logChannel(ctx, event.GuildID, logs.ChannelID)
	author := "Unknown"
	content := "(content was not cached)"
	source := event.BeforeDelete
	if source == nil {
		source = event.Message
	}
	if source != nil {
		if source.Author != nil {
			if source.Author.Bot {
				return
			}
			author = "<@" + source.Author.ID + ">"
		}
		if strings.TrimSpace(source.Content) != "" {
			content = trimLog(source.Content)
		}
	}
	b.sendLog(channel, &discordgo.MessageEmbed{
		Title:       "Message deleted",
		Description: fmt.Sprintf("%s in <#%s>\n%s", author, event.ChannelID, content),
		Color:       colorMist,
	})
}

func (b *Bot) onEdit(s *discordgo.Session, event *discordgo.MessageUpdate) {
	if event.GuildID == "" || event.Author != nil && event.Author.Bot {
		return
	}
	before := ""
	after := trimLog(event.Content)
	if event.BeforeUpdate != nil {
		before = trimLog(event.BeforeUpdate.Content)
		if event.BeforeUpdate.Author != nil && event.BeforeUpdate.Author.Bot {
			return
		}
	}
	if before == "" && after == "" || before == after {
		return
	}
	ctx, cancel := contextTimeout()
	defer cancel()
	logs, err := b.store.GetLogConfig(ctx, event.GuildID)
	if err != nil || !logs.MessageEdit {
		return
	}
	author := "Someone"
	if event.Author != nil {
		author = "<@" + event.Author.ID + ">"
	}
	b.sendLog(b.logChannel(ctx, event.GuildID, logs.ChannelID), &discordgo.MessageEmbed{
		Title:       "Message edited",
		Description: fmt.Sprintf("%s in <#%s>\nBefore: %s\nAfter: %s", author, event.ChannelID, emptyLog(before), emptyLog(after)),
		Color:       colorMist,
	})
}

func (b *Bot) onBulkDelete(s *discordgo.Session, event *discordgo.MessageDeleteBulk) {
	if event.GuildID == "" {
		return
	}
	for _, id := range event.Messages {
		if b.muted.seen(id) {
			return
		}
	}
	ctx, cancel := contextTimeout()
	defer cancel()
	logs, err := b.store.GetLogConfig(ctx, event.GuildID)
	if err != nil || !logs.MessageDelete {
		return
	}
	b.sendLog(b.logChannel(ctx, event.GuildID, logs.ChannelID), &discordgo.MessageEmbed{
		Title:       "Messages deleted",
		Description: fmt.Sprintf("%d messages were removed in <#%s>.", len(event.Messages), event.ChannelID),
		Color:       colorMist,
	})
}

func (b *Bot) onBan(s *discordgo.Session, event *discordgo.GuildBanAdd) {
	if event.User == nil {
		return
	}
	ctx, cancel := contextTimeout()
	defer cancel()
	logs, err := b.store.GetLogConfig(ctx, event.GuildID)
	if err != nil || !logs.MemberBan {
		return
	}
	b.sendLog(b.logChannel(ctx, event.GuildID, logs.ChannelID), &discordgo.MessageEmbed{
		Title:       "Member banned",
		Description: fmt.Sprintf("<@%s> · %s", event.User.ID, event.User.Username),
		Color:       colorBerry,
	})
}

func (b *Bot) logChannel(ctx context.Context, guildID, configured string) string {
	if configured != "" {
		return configured
	}
	guild, err := b.store.GetGuild(ctx, guildID)
	if err != nil {
		return ""
	}
	return guild.ModLogChannelID
}

func trimLog(content string) string {
	content = strings.TrimSpace(content)
	if len(content) > 500 {
		return content[:500] + "..."
	}
	return content
}

func emptyLog(content string) string {
	if content == "" {
		return "(empty)"
	}
	return content
}
