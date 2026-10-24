package discord

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/BoxingOctopus/squirrelbot/internal/domain"
	"github.com/BoxingOctopus/squirrelbot/internal/store"
	"github.com/bwmarrin/discordgo"
)

func (b *Bot) gateCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	action := optionString(i, "action")
	need := access{perm: domain.PermModerateMembers, mod: true, admin: true}
	if action == "panel" {
		need = access{perm: domain.PermManageGuild, admin: true}
	}
	if _, ok := b.permit(i, need); !ok {
		return
	}
	if err := deferReply(s, i, true); err != nil {
		return
	}
	ctx, cancel := contextTimeout()
	defer cancel()
	gate, err := b.store.GetGate(ctx, i.GuildID)
	if err != nil {
		editReply(s, i, "I could not load the application.")
		return
	}
	if action == "panel" {
		channel := optionChannel(i, "channel")
		if channel == nil {
			editReply(s, i, "Choose a channel for the button.")
			return
		}
		if err := b.PostGatePanel(i.GuildID, channel.ID); err != nil {
			editReply(s, i, err.Error())
			return
		}
		editReply(s, i, "Posted the application message.")
		return
	}
	pending, _ := b.store.CountPending(ctx, i.GuildID)
	if !gate.Enabled {
		editReply(s, i, "Application kicks are off. Turn them on in the dashboard.")
		return
	}
	editReply(s, i, fmt.Sprintf("Members must finish the server application within %s of joining. %d people are still waiting.",
		domain.FormatDuration(time.Duration(gate.DeadlineSeconds)*time.Second), pending))
}

func (b *Bot) PostGatePanel(guildID, channelID string) error {
	ctx, cancel := contextTimeout()
	defer cancel()
	gate, err := b.store.GetGate(ctx, guildID)
	if err != nil {
		return errors.New("I could not load the application.")
	}
	content := domain.GatePanelText()
	components := gatePanelComponents()
	messageID := ""
	if gate.ChannelID == channelID && gate.MessageID != "" {
		edit := discordgo.NewMessageEdit(channelID, gate.MessageID)
		edit.Content = &content
		edit.Components = &components
		edit.AllowedMentions = noMentions
		if _, err := b.session.ChannelMessageEditComplex(edit); err == nil {
			messageID = gate.MessageID
		} else if statusOf(err) != 404 {
			return errors.New("I could not update the application message.")
		}
	}
	if messageID == "" {
		sent, err := b.session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
			Content:         content,
			Components:      components,
			AllowedMentions: noMentions,
		})
		if err != nil {
			switch statusOf(err) {
			case 403:
				return errors.New("I need permission to send messages in that channel.")
			case 404:
				return errors.New("I cannot see that channel.")
			default:
				return errors.New("Discord refused the application message.")
			}
		}
		messageID = sent.ID
		if gate.MessageID != "" && gate.MessageID != messageID {
			b.RemoveReactionPanel(gate.ChannelID, gate.MessageID)
		}
	}
	if err := b.store.SetGateMessage(ctx, guildID, channelID, messageID); err != nil {
		if errors.Is(err, store.ErrGateUnsaved) {
			return err
		}
		return errors.New("I could not save the application message.")
	}
	b.dropOtherReactions(channelID, messageID, "")
	return nil
}

func gatePanelComponents() []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{Label: "Answer questions", Style: discordgo.PrimaryButton, CustomID: "sb:verify"},
		}},
	}
}

func (b *Bot) dropOtherReactions(channelID, messageID, keep string) {
	if b.session == nil {
		return
	}
	msg, err := b.session.ChannelMessage(channelID, messageID)
	if err != nil || msg == nil {
		return
	}
	for _, reaction := range msg.Reactions {
		if reaction == nil || !reaction.Me || reaction.Emoji == nil {
			continue
		}
		key := emojiKey(*reaction.Emoji)
		if key == "" || key == keep {
			continue
		}
		b.LiftReaction(channelID, messageID, key)
	}
}

func (b *Bot) verifyMember(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.GuildID == "" || i.Member == nil || i.Member.User == nil || i.Member.User.Bot {
		_ = respondEphemeral(s, i, "Use this button in a server.")
		return
	}
	ctx, cancel := contextTimeout()
	defer cancel()
	gate, err := b.store.GetGate(ctx, i.GuildID)
	if err != nil || !gate.Enabled {
		_ = respondEphemeral(s, i, "The server application is not set up.")
		return
	}
	rows := []discordgo.MessageComponent{questionRow("age", domain.AgeQuestion, "Whole number", 3)}
	for n, prompt := range gate.Questions {
		rows = append(rows, questionRow(fmt.Sprintf("q%d", n+1), prompt, "", 300))
	}
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID:   "sb:verify",
			Title:      "Server application",
			Components: rows,
		},
	})
	if err != nil {
		b.log.Warn("application form", "guild", i.GuildID, "err", err)
	}
}

func questionRow(customID, label, placeholder string, max int) discordgo.MessageComponent {
	return discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		discordgo.TextInput{
			CustomID:    customID,
			Label:       label,
			Style:       discordgo.TextInputShort,
			Placeholder: placeholder,
			Required:    true,
			MinLength:   1,
			MaxLength:   max,
		},
	}}
}

func (b *Bot) submitVerification(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.GuildID == "" || i.Member == nil || i.Member.User == nil || i.Member.User.Bot {
		_ = respondEphemeral(s, i, "Use this form in a server.")
		return
	}
	if err := deferReply(s, i, true); err != nil {
		return
	}
	ctx, cancel := contextTimeout()
	defer cancel()
	gate, err := b.store.GetGate(ctx, i.GuildID)
	if err != nil || !gate.Enabled {
		editReply(s, i, "The server application is not set up.")
		return
	}
	values := modalValues(i.ModalSubmitData())
	under, ok := domain.ClassifyAge(values["age"])
	if !ok {
		editReply(s, i, "Enter your age as a whole number.")
		return
	}
	userID := i.Member.User.ID
	userName := display(i.Member.User, i.Member)
	if under {
		description := fmt.Sprintf("<@%s> was removed for being under 18.", userID)
		if err := b.removeForGate(ctx, i.GuildID, userID, userName, domain.AgeKickReason(), gate.LogChannelID, description); err != nil {
			if statusOf(err) == 404 {
				_ = b.store.SetGateStatus(ctx, i.GuildID, userID, "left")
				editReply(s, i, "You are no longer in the server.")
				return
			}
			_ = b.store.SetGateStatus(ctx, i.GuildID, userID, "kick_failed")
			b.log.Warn("application age kick", "guild", i.GuildID, "err", err)
			editReply(s, i, "I could not remove you. A moderator still needs to review this.")
			return
		}
		editReply(s, i, "You were removed because you are under 18.")
		return
	}
	for n := range gate.Questions {
		if strings.TrimSpace(values[fmt.Sprintf("q%d", n+1)]) == "" {
			editReply(s, i, "The questions changed. Open the form again.")
			return
		}
	}
	if err := b.store.SetGateStatus(ctx, i.GuildID, userID, "cleared"); err != nil {
		editReply(s, i, "I could not record that you finished.")
		return
	}
	editReply(s, i, "Your application is in. You can stay.")
}

func modalValues(data discordgo.ModalSubmitInteractionData) map[string]string {
	values := map[string]string{}
	for _, row := range data.Components {
		action, ok := row.(*discordgo.ActionsRow)
		if !ok {
			continue
		}
		for _, component := range action.Components {
			input, ok := component.(*discordgo.TextInput)
			if !ok {
				continue
			}
			values[input.CustomID] = input.Value
		}
	}
	return values
}

func (b *Bot) onJoin(s *discordgo.Session, event *discordgo.GuildMemberAdd) {
	if event.User == nil || event.User.Bot {
		return
	}
	ctx, cancel := contextTimeout()
	defer cancel()
	guild, err := b.touchGuild(ctx, event.GuildID)
	if err != nil {
		b.log.Error("join guild", "guild", event.GuildID, "err", err)
		return
	}
	if domain.ValidSnowflake(guild.AutoroleID) && !memberHas(event.Member, guild.AutoroleID) {
		if err := b.canAssign(event.GuildID, guild.AutoroleID); err == nil {
			if err := s.GuildMemberRoleAdd(event.GuildID, event.User.ID, guild.AutoroleID); err != nil {
				b.log.Warn("autorole", "guild", event.GuildID, "err", err)
			} else {
				event.Roles = append(event.Roles, guild.AutoroleID)
			}
		}
	}
	b.trackGate(event)
	if guild.WelcomeChannelID != "" && guild.WelcomeMessage != "" {
		count := 0
		if g, err := s.Guild(event.GuildID); err == nil {
			count = g.MemberCount
		}
		content := domain.ApplyTemplate(guild.WelcomeMessage, map[string]string{
			"user":        "<@" + event.User.ID + ">",
			"username":    event.User.Username,
			"server":      b.guildName(event.GuildID),
			"membercount": fmt.Sprintf("%d", count),
		})
		_, _ = s.ChannelMessageSendComplex(guild.WelcomeChannelID, &discordgo.MessageSend{
			Content:         content,
			AllowedMentions: &discordgo.MessageAllowedMentions{Users: []string{event.User.ID}},
		})
	}
	logs, _ := b.store.GetLogConfig(ctx, event.GuildID)
	if logs.MemberJoin {
		channel := logs.ChannelID
		if channel == "" {
			channel = guild.ModLogChannelID
		}
		b.sendLog(channel, &discordgo.MessageEmbed{
			Title:       "Member joined",
			Description: fmt.Sprintf("<@%s> · %s", event.User.ID, event.User.Username),
			Color:       colorFoxfire,
		})
	}
}

func (b *Bot) trackGate(event *discordgo.GuildMemberAdd) {
	ctx, cancel := contextTimeout()
	defer cancel()
	gate, err := b.store.GetGate(ctx, event.GuildID)
	if err != nil || !gate.Enabled {
		return
	}
	status := "pending"
	if event.Member != nil && overlaps(event.Member.Roles, gate.ExemptRoleIDs) {
		status = "exempt"
	}
	joined := event.JoinedAt
	if joined.IsZero() {
		joined = time.Now()
	}
	member := store.GateMember{
		GuildID: event.GuildID, UserID: event.User.ID, UserName: display(event.User, event.Member),
		JoinedAt: joined, DeadlineAt: domain.GateDeadline(joined, time.Duration(gate.DeadlineSeconds)*time.Second),
		Status: status,
	}
	if err := b.store.UpsertGateMember(ctx, member); err != nil {
		b.log.Error("track gate", "guild", event.GuildID, "err", err)
		return
	}
	if status != "pending" {
		return
	}
	message := gate.DMMessage
	if message == "" {
		message = "Welcome to {server}. Finish the server application within {duration} or you will be removed. You must be 18 or older."
	}
	b.dm(event.User.ID, domain.ApplyTemplate(message, map[string]string{
		"server":   b.guildName(event.GuildID),
		"duration": domain.FormatDuration(time.Duration(gate.DeadlineSeconds) * time.Second),
	}))
}

func (b *Bot) onMemberUpdate(s *discordgo.Session, event *discordgo.GuildMemberUpdate) {
	if event.Member == nil || event.Member.User == nil || event.Member.User.Bot {
		return
	}
	ctx, cancel := contextTimeout()
	defer cancel()
	gate, err := b.store.GetGate(ctx, event.GuildID)
	if err != nil || !gate.Enabled {
		return
	}
	if event.Member != nil && overlaps(event.Member.Roles, gate.ExemptRoleIDs) {
		_ = b.store.SetGateStatus(ctx, event.GuildID, event.Member.User.ID, "exempt")
	}
}

func (b *Bot) onLeave(s *discordgo.Session, event *discordgo.GuildMemberRemove) {
	if event.User == nil {
		return
	}
	ctx, cancel := contextTimeout()
	defer cancel()
	_ = b.store.SetGateStatus(ctx, event.GuildID, event.User.ID, "left")
	logs, err := b.store.GetLogConfig(ctx, event.GuildID)
	if err != nil || !logs.MemberLeave {
		return
	}
	channel := logs.ChannelID
	if channel == "" {
		if guild, gerr := b.store.GetGuild(ctx, event.GuildID); gerr == nil {
			channel = guild.ModLogChannelID
		}
	}
	b.sendLog(channel, &discordgo.MessageEmbed{
		Title:       "Member left",
		Description: fmt.Sprintf("<@%s> · %s", event.User.ID, event.User.Username),
		Color:       colorMist,
	})
}

func overlaps(have, want []string) bool {
	for _, id := range want {
		if domain.ContainsID(have, id) {
			return true
		}
	}
	return false
}
