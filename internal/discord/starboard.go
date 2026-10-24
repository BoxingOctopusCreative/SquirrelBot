package discord

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/BoxingOctopus/squirrelbot/internal/domain"
	"github.com/BoxingOctopus/squirrelbot/internal/store"
	"github.com/bwmarrin/discordgo"
)

const starColor = 0xE0B44C

func (b *Bot) onReactionRemoveAll(s *discordgo.Session, event *discordgo.MessageReactionRemoveAll) {
	b.refreshStar(event.GuildID, event.ChannelID, event.MessageID, nil)
}

func (b *Bot) refreshStar(guildID, channelID, messageID string, emoji *discordgo.Emoji) {
	if guildID == "" || channelID == "" || messageID == "" {
		return
	}
	if emoji != nil && !domain.IsStarEmoji(emoji.Name) {
		return
	}
	ctx, cancel := contextTimeout()
	defer cancel()
	board, err := b.store.GetStarboard(ctx, guildID)
	if err != nil || !board.Enabled || board.ChannelID == "" {
		return
	}
	post, found, err := b.store.StarPost(ctx, guildID, messageID)
	if err != nil {
		b.log.Warn("starboard lookup", "guild", guildID, "err", err)
		return
	}
	if channelID == board.ChannelID || domain.ContainsID(board.IgnoredChannelIDs, channelID) {
		if found {
			b.dropStarPost(ctx, post)
		}
		return
	}
	msg, err := b.session.ChannelMessage(channelID, messageID)
	if err != nil {
		if statusOf(err) == 404 && found {
			b.dropStarPost(ctx, post)
		}
		return
	}
	if msg.Author == nil {
		return
	}
	reactors, err := b.starReactors(channelID, messageID)
	if err != nil {
		b.log.Warn("starboard reactions", "guild", guildID, "err", err)
		return
	}
	count := domain.CountStars(msg.Author.ID, reactors, board.SelfStar)
	if count < board.Threshold {
		if found {
			b.dropStarPost(ctx, post)
		}
		return
	}
	b.publishStar(ctx, board, msg, count, starImage(msg, b.channelNSFW(channelID), board.NSFW), post, found)
}

func (b *Bot) dropStarPost(ctx context.Context, post store.StarPost) {
	if err := b.session.ChannelMessageDelete(post.StarChannelID, post.StarMessageID); err != nil && statusOf(err) != 404 {
		b.log.Warn("starboard delete", "guild", post.GuildID, "err", err)
	}
	if err := b.store.DeleteStarPost(ctx, post.GuildID, post.SourceMessageID); err != nil {
		b.log.Warn("starboard forget", "guild", post.GuildID, "err", err)
	}
}

func (b *Bot) publishStar(ctx context.Context, board store.Starboard, msg *discordgo.Message, count int, image string, existing store.StarPost, found bool) {
	embed := starEmbed(msg, count, image)
	if found {
		embeds := []*discordgo.MessageEmbed{embed}
		edit := discordgo.NewMessageEdit(existing.StarChannelID, existing.StarMessageID)
		edit.Embeds = &embeds
		edit.AllowedMentions = noMentions
		if _, err := b.session.ChannelMessageEditComplex(edit); err == nil {
			return
		} else if statusOf(err) != 404 {
			b.log.Warn("starboard edit", "guild", msg.GuildID, "err", err)
			return
		}
	}
	sent, err := b.session.ChannelMessageSendComplex(board.ChannelID, &discordgo.MessageSend{
		Embeds:          []*discordgo.MessageEmbed{embed},
		AllowedMentions: noMentions,
	})
	if err != nil {
		b.log.Warn("starboard post", "guild", msg.GuildID, "err", err)
		return
	}
	err = b.store.SaveStarPost(ctx, store.StarPost{
		GuildID:         msg.GuildID,
		SourceChannelID: msg.ChannelID,
		SourceMessageID: msg.ID,
		StarChannelID:   sent.ChannelID,
		StarMessageID:   sent.ID,
	})
	if err != nil {
		b.log.Warn("starboard save", "guild", msg.GuildID, "err", err)
	}
}

func (b *Bot) starReactors(channelID, messageID string) ([]domain.StarReactor, error) {
	emoji := url.PathEscape(domain.StarEmoji)
	var out []domain.StarReactor
	after := ""
	for page := 0; page < 5; page++ {
		users, err := b.session.MessageReactions(channelID, messageID, emoji, 100, "", after)
		if err != nil {
			return nil, err
		}
		for _, user := range users {
			if user == nil || user.ID == "" {
				continue
			}
			out = append(out, domain.StarReactor{ID: user.ID, Bot: user.Bot})
			after = user.ID
		}
		if len(users) < 100 {
			break
		}
	}
	return out, nil
}

func (b *Bot) channelNSFW(channelID string) bool {
	if b.session.State != nil {
		if channel, err := b.session.State.Channel(channelID); err == nil && channel != nil {
			return channel.NSFW
		}
	}
	channel, err := b.session.Channel(channelID)
	return err == nil && channel != nil && channel.NSFW
}

func starEmbed(msg *discordgo.Message, count int, image string) *discordgo.MessageEmbed {
	text := starBody(msg.Content)
	if text == "" {
		text = "(no text)"
	}
	jump := fmt.Sprintf("https://discord.com/channels/%s/%s/%s", msg.GuildID, msg.ChannelID, msg.ID)
	embed := &discordgo.MessageEmbed{
		Description: text + "\n\n[Jump to message](" + jump + ")",
		Color:       starColor,
		Footer:      &discordgo.MessageEmbedFooter{Text: fmt.Sprintf("⭐ %d", count)},
	}
	if msg.Author != nil {
		embed.Author = &discordgo.MessageEmbedAuthor{
			Name:    display(msg.Author, nil),
			IconURL: msg.Author.AvatarURL("64"),
		}
	}
	if !msg.Timestamp.IsZero() {
		embed.Timestamp = msg.Timestamp.Format(time.RFC3339)
	}
	if image != "" {
		embed.Image = &discordgo.MessageEmbedImage{URL: image}
	}
	return embed
}

func starBody(content string) string {
	content = strings.ReplaceAll(content, "@everyone", "@\u200beveryone")
	content = strings.ReplaceAll(content, "@here", "@\u200bhere")
	content = strings.TrimSpace(content)
	if utf8.RuneCountInString(content) <= 1000 {
		return content
	}
	return string([]rune(content)[:1000]) + "..."
}

func starImage(msg *discordgo.Message, nsfwChannel, allowNSFW bool) string {
	if nsfwChannel && !allowNSFW {
		return ""
	}
	for _, file := range msg.Attachments {
		if file == nil || file.URL == "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(file.ContentType), "image/") || imageName(file.Filename) {
			return file.URL
		}
	}
	for _, embed := range msg.Embeds {
		if embed != nil && embed.Image != nil && embed.Image.URL != "" {
			return embed.Image.URL
		}
	}
	return ""
}

func imageName(name string) bool {
	name = strings.ToLower(name)
	return strings.HasSuffix(name, ".png") ||
		strings.HasSuffix(name, ".jpg") ||
		strings.HasSuffix(name, ".jpeg") ||
		strings.HasSuffix(name, ".gif") ||
		strings.HasSuffix(name, ".webp")
}
