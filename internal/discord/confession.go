package discord

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/BoxingOctopus/squirrelbot/internal/domain"
	"github.com/BoxingOctopus/squirrelbot/internal/store"
	"github.com/bwmarrin/discordgo"
)

func (b *Bot) confess(s *discordgo.Session, i *discordgo.InteractionCreate) {
	body, err := domain.CleanConfession(optionString(i, "text"))
	if err != nil {
		_ = respondEphemeral(s, i, err.Error())
		return
	}
	if !b.beginConfession(s, i) {
		return
	}
	b.postConfession(s, i, body, nil)
}

func (b *Bot) beginConfession(s *discordgo.Session, i *discordgo.InteractionCreate) bool {
	if i.GuildID == "" || i.Member == nil || i.Member.User == nil {
		_ = respondEphemeral(s, i, "Use this command in a server.")
		return false
	}
	ctx, cancel := contextTimeout()
	defer cancel()
	if _, err := b.touchGuild(ctx, i.GuildID); err != nil {
		b.log.Error("confession guild", "guild", i.GuildID, "err", err)
		_ = respondEphemeral(s, i, "I could not load this server.")
		return false
	}
	cfg, err := b.store.GetConfessionConfig(ctx, i.GuildID)
	if err != nil {
		b.log.Error("confession config", "guild", i.GuildID, "err", err)
		_ = respondEphemeral(s, i, "I could not load confessions for this server.")
		return false
	}
	if !domain.ValidSnowflake(cfg.ReviewChannelID) {
		_ = respondEphemeral(s, i, "A moderator has not chosen a review channel yet.")
		return false
	}
	waiting, err := b.store.CountPendingConfessions(ctx, i.GuildID, i.Member.User.ID)
	if err != nil {
		b.log.Error("confession count", "guild", i.GuildID, "err", err)
		_ = respondEphemeral(s, i, "I could not check your waiting confessions.")
		return false
	}
	if waiting >= domain.MaxPendingConfessions {
		_ = respondEphemeral(s, i, "You already have confessions waiting. A moderator needs to review those first.")
		return false
	}
	return deferReply(s, i, true) == nil
}

func (b *Bot) postConfession(s *discordgo.Session, i *discordgo.InteractionCreate, body string, file *confessionFile) {
	ctx, cancel := contextTimeout()
	defer cancel()
	guild, err := b.store.GetGuild(ctx, i.GuildID)
	if err != nil {
		editReply(s, i, "I could not load this server.")
		return
	}
	cfg, err := b.store.GetConfessionConfig(ctx, i.GuildID)
	if err != nil || !domain.ValidSnowflake(cfg.ReviewChannelID) {
		editReply(s, i, "I could not load confessions for this server.")
		return
	}
	created, err := b.store.CreateConfession(ctx, store.Confession{
		GuildID: i.GuildID, AuthorID: i.Member.User.ID, AuthorName: clipName(display(i.Member.User, i.Member)), Body: body,
	})
	if err != nil {
		b.log.Error("confession save", "guild", i.GuildID, "err", err)
		editReply(s, i, "I could not save that confession.")
		return
	}
	content, mentions := reviewMention(guild.ModRoleID)
	messageID, err := b.sendConfession(cfg.ReviewChannelID, content, mentions, reviewConfessionEmbed(created), confessionButtons(created.ID), file)
	if err != nil {
		_ = b.store.DeleteConfession(ctx, created.ID)
		editReply(s, i, err.Error())
		return
	}
	if err := b.store.AttachConfessionReview(ctx, created.ID, cfg.ReviewChannelID, messageID); err != nil {
		b.log.Error("confession review id", "guild", i.GuildID, "err", err)
		_ = b.session.ChannelMessageDelete(cfg.ReviewChannelID, messageID)
		_ = b.store.DeleteConfession(ctx, created.ID)
		editReply(s, i, "I could not save that confession.")
		return
	}
	editReply(s, i, "Sent to the moderators. They will post it if they approve it.")
}

func (b *Bot) setConfessionReview(s *discordgo.Session, i *discordgo.InteractionCreate) {
	b.setConfessionChannel(s, i, true)
}

func (b *Bot) setConfessionBoard(s *discordgo.Session, i *discordgo.InteractionCreate) {
	b.setConfessionChannel(s, i, false)
}

func (b *Bot) setConfessionChannel(s *discordgo.Session, i *discordgo.InteractionCreate, review bool) {
	if _, ok := b.permit(i, access{perm: domain.PermModerateMembers, mod: true, admin: true}); !ok {
		return
	}
	channel := optionChannel(i, "channel")
	if channel == nil || !domain.ValidSnowflake(channel.ID) {
		_ = respondEphemeral(s, i, "Choose a channel.")
		return
	}
	if err := deferReply(s, i, true); err != nil {
		return
	}
	ctx, cancel := contextTimeout()
	defer cancel()
	if _, err := b.touchGuild(ctx, i.GuildID); err != nil {
		editReply(s, i, "I could not load this server.")
		return
	}
	cfg, err := b.store.GetConfessionConfig(ctx, i.GuildID)
	if err != nil {
		b.log.Error("confession config", "guild", i.GuildID, "err", err)
		editReply(s, i, "I could not load confessions for this server.")
		return
	}
	if review {
		cfg.ReviewChannelID = channel.ID
	} else {
		cfg.ChannelID = channel.ID
	}
	if err := domain.ValidateConfessionChannels(cfg.ReviewChannelID, cfg.ChannelID); err != nil {
		editReply(s, i, err.Error())
		return
	}
	if err := b.store.SaveConfessionConfig(ctx, cfg); err != nil {
		b.log.Error("confession save config", "guild", i.GuildID, "err", err)
		editReply(s, i, "I could not save that channel.")
		return
	}
	name := channel.Name
	if name == "" {
		name = "that channel"
	} else {
		name = "#" + name
	}
	if review {
		editReply(s, i, "Moderators will review confessions in "+name+". Only moderators should be able to read it.")
		return
	}
	editReply(s, i, "Approved confessions will be posted in "+name+".")
}

func (b *Bot) reviewConfession(s *discordgo.Session, i *discordgo.InteractionCreate) {
	action, id, ok := domain.ParseConfessionAction(i.MessageComponentData().CustomID)
	if !ok {
		_ = respondEphemeral(s, i, "That button is not a confession.")
		return
	}
	if _, allowed := b.permit(i, access{perm: domain.PermModerateMembers, mod: true, admin: true}); !allowed {
		return
	}
	if err := deferComponent(s, i); err != nil {
		return
	}
	ctx, cancel := contextTimeout()
	defer cancel()
	cfg, err := b.store.GetConfessionConfig(ctx, i.GuildID)
	if err != nil {
		b.log.Error("confession config", "guild", i.GuildID, "err", err)
		followupEphemeral(s, i, "I could not load confessions for this server.")
		return
	}
	item, err := b.store.GetConfession(ctx, i.GuildID, id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			followupEphemeral(s, i, "That confession is gone.")
			return
		}
		b.log.Error("confession load", "guild", i.GuildID, "err", err)
		followupEphemeral(s, i, "I could not load that confession.")
		return
	}
	if item.Status != domain.ConfessionPending {
		followupEphemeral(s, i, "That confession was already reviewed.")
		return
	}
	reviewerID, reviewerName := actor(i)
	status := domain.ConfessionRejected
	publicChannel := ""
	if action == "approve" {
		if !domain.ValidSnowflake(cfg.ChannelID) {
			followupEphemeral(s, i, "Set the confessions channel first.")
			return
		}
		status = domain.ConfessionApproved
		publicChannel = cfg.ChannelID
	}
	claimed, err := b.store.ClaimConfession(ctx, i.GuildID, id, status, reviewerID, clipName(reviewerName), publicChannel)
	if err != nil {
		if errors.Is(err, store.ErrConfessionClosed) {
			followupEphemeral(s, i, "That confession was already reviewed.")
			return
		}
		b.log.Error("confession claim", "guild", i.GuildID, "err", err)
		followupEphemeral(s, i, "I could not record that decision.")
		return
	}
	if status == domain.ConfessionApproved {
		var file *confessionFile
		if ref, ok := firstAttachment(i.Message); ok {
			downloaded, err := b.downloadConfessionFile(ref)
			if err != nil {
				if releaseErr := b.store.ReleaseConfession(ctx, claimed.ID); releaseErr != nil {
					b.log.Error("confession release", "guild", i.GuildID, "err", releaseErr)
				}
				followupEphemeral(s, i, err.Error())
				return
			}
			file = downloaded
		}
		messageID, err := b.sendConfession(publicChannel, "", noMentions, publicConfessionEmbed(claimed.Body), submitConfessionButton(), file)
		if err != nil {
			if releaseErr := b.store.ReleaseConfession(ctx, claimed.ID); releaseErr != nil {
				b.log.Error("confession release", "guild", i.GuildID, "err", releaseErr)
			}
			followupEphemeral(s, i, err.Error())
			return
		}
		if err := b.store.SetConfessionPublicMessage(ctx, claimed.ID, publicChannel, messageID); err != nil {
			b.log.Error("confession public id", "guild", i.GuildID, "err", err)
			_ = b.session.ChannelMessageDelete(publicChannel, messageID)
			_ = b.store.ReleaseConfession(ctx, claimed.ID)
			followupEphemeral(s, i, "I posted it, then could not save the record, so I removed the post.")
			return
		}
	}
	b.finishConfessionReview(i, claimed)
	if status == domain.ConfessionApproved {
		followupEphemeral(s, i, "Posted.")
		return
	}
	followupEphemeral(s, i, "Rejected.")
}

func (b *Bot) finishConfessionReview(i *discordgo.InteractionCreate, item store.Confession) {
	channelID := item.ReviewChannelID
	messageID := item.ReviewMessageID
	if i.Message != nil {
		if i.Message.ChannelID != "" {
			channelID = i.Message.ChannelID
		}
		if i.Message.ID != "" {
			messageID = i.Message.ID
		}
	}
	if channelID == "" || messageID == "" {
		return
	}
	empty := []discordgo.MessageComponent{}
	embeds := []*discordgo.MessageEmbed{reviewConfessionEmbed(item)}
	edit := discordgo.NewMessageEdit(channelID, messageID)
	edit.Components = &empty
	edit.Embeds = &embeds
	edit.AllowedMentions = noMentions
	if _, err := b.session.ChannelMessageEditComplex(edit); err != nil {
		b.log.Warn("confession review edit", "guild", i.GuildID, "err", err)
	}
}

func reviewMention(modRoleID string) (string, *discordgo.MessageAllowedMentions) {
	if !domain.ValidSnowflake(modRoleID) {
		return "", noMentions
	}
	return "<@&" + modRoleID + ">", &discordgo.MessageAllowedMentions{
		Parse: []discordgo.AllowedMentionType{},
		Roles: []string{modRoleID},
	}
}

func (b *Bot) sendConfession(channelID, content string, mentions *discordgo.MessageAllowedMentions, embed *discordgo.MessageEmbed, components []discordgo.MessageComponent, file *confessionFile) (string, error) {
	if components == nil {
		components = []discordgo.MessageComponent{}
	}
	if mentions == nil {
		mentions = noMentions
	}
	payload := &discordgo.MessageSend{
		Content:         content,
		Embeds:          []*discordgo.MessageEmbed{embed},
		Components:      components,
		AllowedMentions: mentions,
	}
	if file != nil && len(file.Data) > 0 {
		payload.Files = []*discordgo.File{{
			Name:        file.Name,
			ContentType: file.ContentType,
			Reader:      bytes.NewReader(file.Data),
		}}
	}
	sent, err := b.session.ChannelMessageSendComplex(channelID, payload)
	if err != nil {
		switch statusOf(err) {
		case 403:
			return "", errors.New("I need permission to send messages and files in that channel.")
		case 404:
			return "", errors.New("I cannot see that channel.")
		default:
			return "", errors.New("Discord refused that confession.")
		}
	}
	if sent == nil || sent.ID == "" {
		return "", errors.New("Discord refused that confession.")
	}
	return sent.ID, nil
}

func reviewConfessionEmbed(item store.Confession) *discordgo.MessageEmbed {
	title := "Confession waiting"
	color := 0xD08A4A
	decision := ""
	switch item.Status {
	case domain.ConfessionApproved:
		title = "Confession approved"
		color = 0x3C8F62
		decision = "Approved by " + item.ReviewerName
	case domain.ConfessionRejected:
		title = "Confession rejected"
		color = 0xC4483F
		decision = "Rejected by " + item.ReviewerName
	}
	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: item.Body,
		Color:       color,
		Fields: []*discordgo.MessageEmbedField{{
			Name:  "From",
			Value: fmt.Sprintf("%s (<@%s>)", item.AuthorName, item.AuthorID),
		}},
	}
	if decision != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  "Decision",
			Value: decision,
		})
	}
	return embed
}

func publicConfessionEmbed(body string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "Anonymous confession",
		Description: body,
		Color:       0xD08A4A,
		Footer:      &discordgo.MessageEmbedFooter{Text: "SquirrelBot"},
	}
}

func confessionButtons(id int64) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Approve",
				Style:    discordgo.SuccessButton,
				CustomID: domain.ConfessionCustomID("approve", id),
			},
			discordgo.Button{
				Label:    "Reject",
				Style:    discordgo.DangerButton,
				CustomID: domain.ConfessionCustomID("reject", id),
			},
		}},
	}
}

func deferComponent(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
}

func followupEphemeral(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content:         content,
		Flags:           discordgo.MessageFlagsEphemeral,
		AllowedMentions: noMentions,
	})
}

func clipName(name string) string {
	name = strings.TrimSpace(strings.ReplaceAll(name, "\n", " "))
	if name == "" {
		return "unknown"
	}
	if utf8.RuneCountInString(name) <= 80 {
		return name
	}
	return string([]rune(name)[:80])
}
