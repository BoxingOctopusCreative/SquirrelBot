package discord

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/BoxingOctopus/squirrelbot/internal/domain"
	"github.com/BoxingOctopus/squirrelbot/internal/store"
	"github.com/bwmarrin/discordgo"
)

func (b *Bot) PostTicketPanel(guildID string) error {
	ctx, cancel := contextTimeout()
	defer cancel()
	cfg, err := b.store.GetTicketConfig(ctx, guildID)
	if err != nil {
		return errors.New("I could not load tickets for this server.")
	}
	if err := domain.TicketReady(cfg.PanelChannelID, cfg.CategoryID, cfg.SupportRoleID); err != nil {
		return err
	}
	if err := b.requireTicketPlaces(guildID, cfg); err != nil {
		return err
	}
	content := domain.TicketPanelText(cfg.PanelTitle, cfg.PanelBody)
	components := ticketPanelComponents()
	messageID := ""
	if cfg.PanelMessageID != "" {
		edit := discordgo.NewMessageEdit(cfg.PanelChannelID, cfg.PanelMessageID)
		edit.Content = &content
		edit.Components = &components
		edit.AllowedMentions = noMentions
		if _, err := b.session.ChannelMessageEditComplex(edit); err == nil {
			messageID = cfg.PanelMessageID
		} else if statusOf(err) != 404 {
			return errors.New("I could not update the panel message.")
		}
	}
	fresh := false
	if messageID == "" {
		sent, err := b.session.ChannelMessageSendComplex(cfg.PanelChannelID, &discordgo.MessageSend{
			Content: content, Components: components, AllowedMentions: noMentions,
		})
		if err != nil {
			switch statusOf(err) {
			case 403:
				return errors.New("I need permission to send messages in that channel.")
			case 404:
				return errors.New("I cannot see that channel.")
			default:
				return errors.New("Discord refused the panel message.")
			}
		}
		messageID = sent.ID
		fresh = true
	}
	if err := b.store.SaveTicketPanelMessage(ctx, guildID, messageID); err != nil {
		if fresh {
			_ = b.session.ChannelMessageDelete(cfg.PanelChannelID, messageID)
		}
		b.log.Error("ticket panel id", "guild", guildID, "err", err)
		return errors.New("I could not save the panel.")
	}
	return nil
}

func (b *Bot) DeleteChannelMessage(channelID, messageID string) {
	if channelID == "" || messageID == "" {
		return
	}
	_ = b.session.ChannelMessageDelete(channelID, messageID)
}

func (b *Bot) ticketComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	action, id, ok := domain.ParseTicketAction(i.MessageComponentData().CustomID)
	if !ok {
		_ = respondEphemeral(s, i, "That button is not a ticket.")
		return
	}
	switch action {
	case "open":
		b.openTicketForm(s, i)
	case "close":
		b.promptCloseTicket(s, i, id)
	case "delete":
		b.deleteTicket(s, i, id)
	case "claim", "release", "reopen":
		b.ticketButton(s, i, action, id)
	default:
		_ = respondEphemeral(s, i, "That button is not a ticket.")
	}
}

func (b *Bot) ticketModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	action, id, ok := domain.ParseTicketAction(i.ModalSubmitData().CustomID)
	if !ok {
		_ = respondEphemeral(s, i, "That form is not a ticket.")
		return
	}
	values := modalValues(i.ModalSubmitData())
	switch action {
	case "form":
		b.submitTicket(s, i, values)
	case "close-form":
		b.submitCloseTicket(s, i, id, values)
	case "delete-form":
		b.submitDeleteTicket(s, i, id, values)
	default:
		_ = respondEphemeral(s, i, "That form is not a ticket.")
	}
}

func (b *Bot) openTicketForm(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if _, _, ok := ticketMember(i); !ok {
		_ = respondEphemeral(s, i, "Use this button in a server.")
		return
	}
	ctx, cancel := contextTimeout()
	defer cancel()
	cfg, err := b.readyTicketConfig(ctx, i.GuildID)
	if err != nil {
		_ = respondEphemeral(s, i, err.Error())
		return
	}
	if !cfg.Enabled {
		_ = respondEphemeral(s, i, "Support tickets are off. Turn them on in the dashboard.")
		return
	}
	b.rememberTicketPanel(ctx, i, cfg)
	existing, err := b.store.OpenTicketFor(ctx, i.GuildID, i.Member.User.ID)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		b.log.Error("ticket lookup", "guild", i.GuildID, "err", err)
		_ = respondEphemeral(s, i, "I could not check your open ticket.")
		return
	}
	if err == nil {
		found, lookErr := b.channelPresent(i.GuildID, existing.ChannelID)
		if lookErr != nil {
			_ = respondEphemeral(s, i, "I could not check your open ticket.")
			return
		}
		if found {
			_ = respondEphemeral(s, i, fmt.Sprintf("You already have ticket %d open: <#%s>", existing.Number, existing.ChannelID))
			return
		}
	}
	if err := s.InteractionRespond(i.Interaction, ticketOpenModal()); err != nil {
		b.log.Warn("ticket form", "guild", i.GuildID, "err", err)
	}
}

func (b *Bot) submitTicket(s *discordgo.Session, i *discordgo.InteractionCreate, values map[string]string) {
	userID, name, ok := ticketMember(i)
	if !ok {
		_ = respondEphemeral(s, i, "Use this form in a server.")
		return
	}
	subject, err := domain.CleanTicketField(values["subject"], domain.MaxTicketSubject, "Write a subject.", true)
	if err != nil {
		_ = respondEphemeral(s, i, err.Error())
		return
	}
	body, err := domain.CleanTicketField(values["details"], domain.MaxTicketBody, "Say what you need help with.", false)
	if err != nil {
		_ = respondEphemeral(s, i, err.Error())
		return
	}
	if err := deferReply(s, i, true); err != nil {
		return
	}
	ctx, cancel := contextTimeout()
	defer cancel()
	cfg, err := b.readyTicketConfig(ctx, i.GuildID)
	if err != nil {
		editReply(s, i, err.Error())
		return
	}
	if err := b.requireTicketPlaces(i.GuildID, cfg); err != nil {
		editReply(s, i, err.Error())
		return
	}
	retired, blocked := b.retireStaleTicket(ctx, i, userID)
	if blocked != "" {
		editReply(s, i, blocked)
		return
	}
	created, err := b.store.CreateTicket(ctx, store.Ticket{
		GuildID: i.GuildID, OpenerID: userID, OpenerName: name, Subject: subject, Body: body,
	})
	if err != nil {
		b.log.Error("ticket save", "guild", i.GuildID, "err", err)
		editReply(s, i, "I could not save that ticket.")
		return
	}
	channelID, err := b.createTicketChannel(i.GuildID, cfg, created, userID)
	if err != nil {
		_ = b.store.DeleteUnopenedTicket(ctx, i.GuildID, created.ID)
		editReply(s, i, err.Error())
		return
	}
	created.ChannelID = channelID
	messageID, err := b.sendTicketControl(cfg, created)
	if err != nil {
		_, _ = b.session.ChannelDelete(channelID)
		_ = b.store.DeleteUnopenedTicket(ctx, i.GuildID, created.ID)
		editReply(s, i, err.Error())
		return
	}
	if err := b.store.AttachTicketChannel(ctx, i.GuildID, created.ID, channelID, messageID); err != nil {
		_, _ = b.session.ChannelDelete(channelID)
		_ = b.store.DeleteUnopenedTicket(ctx, i.GuildID, created.ID)
		b.log.Error("ticket attach", "guild", i.GuildID, "err", err)
		editReply(s, i, "I could not save that ticket.")
		return
	}
	reply := fmt.Sprintf("Opened ticket %d in <#%s>.", created.Number, channelID)
	if retired > 0 {
		reply = fmt.Sprintf("Opened ticket %d in <#%s>. Ticket %d was already gone, so that one is closed.", created.Number, channelID, retired)
	}
	editReply(s, i, reply)
}

func (b *Bot) ticketButton(s *discordgo.Session, i *discordgo.InteractionCreate, action string, id int64) {
	userID, name, ok := ticketMember(i)
	if !ok {
		_ = respondEphemeral(s, i, "Use this button in a server.")
		return
	}
	if err := deferComponent(s, i); err != nil {
		return
	}
	ctx, cancel := contextTimeout()
	defer cancel()
	cfg, item, errText := b.ticketForButton(ctx, i, id)
	if errText != "" {
		followupEphemeral(s, i, errText)
		return
	}
	staff := b.memberIsTicketStaff(ctx, i, cfg.SupportRoleID)
	switch action {
	case "claim":
		if !staff {
			followupEphemeral(s, i, "You don't have permission to do that.")
			return
		}
		claimedItem, claimed, err := b.store.ClaimTicket(ctx, i.GuildID, id, userID, name)
		if err != nil {
			b.log.Error("ticket claim", "guild", i.GuildID, "err", err)
			followupEphemeral(s, i, "I could not claim that ticket.")
			return
		}
		if !claimed {
			followupEphemeral(s, i, claimBlocked(claimedItem, userID))
			return
		}
		item = claimedItem
	case "release":
		if !staff {
			followupEphemeral(s, i, "You don't have permission to do that.")
			return
		}
		releasedItem, released, err := b.store.ReleaseTicket(ctx, i.GuildID, id)
		if err != nil {
			b.log.Error("ticket release", "guild", i.GuildID, "err", err)
			followupEphemeral(s, i, "I could not release that ticket.")
			return
		}
		if !released {
			if releasedItem.Status != domain.TicketOpen {
				followupEphemeral(s, i, "This ticket is already closed.")
				return
			}
			followupEphemeral(s, i, "Nobody has claimed this ticket.")
			return
		}
		item = releasedItem
	case "reopen":
		if !staff && !(userID == item.OpenerID && userID == item.ClosedByID) {
			if item.Status == domain.TicketClosed && userID == item.OpenerID {
				followupEphemeral(s, i, "A member of the support team closed this ticket.")
				return
			}
			followupEphemeral(s, i, "You don't have permission to do that.")
			return
		}
		openedItem, opened, err := b.store.ReopenTicket(ctx, i.GuildID, id)
		if err != nil {
			b.log.Error("ticket reopen", "guild", i.GuildID, "err", err)
			followupEphemeral(s, i, "I could not reopen that ticket.")
			return
		}
		if !opened {
			followupEphemeral(s, i, reopenBlocked(openedItem))
			return
		}
		item = openedItem
		if err := b.applyTicketAccess(item.ChannelID, i.GuildID, cfg.SupportRoleID, item.OpenerID, true); err != nil {
			b.log.Warn("ticket access", "guild", i.GuildID, "err", err)
			followupEphemeral(s, i, "Reopened ticket "+fmt.Sprint(item.Number)+", but I could not restore channel access. Check that my role can manage channels.")
		}
	}
	if err := b.presentTicket(s, i, cfg, item); err != nil {
		b.log.Warn("ticket message", "guild", i.GuildID, "err", err)
		followupEphemeral(s, i, "I updated the ticket, but I could not update the message in the channel.")
	}
}

func (b *Bot) promptCloseTicket(s *discordgo.Session, i *discordgo.InteractionCreate, id int64) {
	userID, _, ok := ticketMember(i)
	if !ok {
		_ = respondEphemeral(s, i, "Use this button in a server.")
		return
	}
	ctx, cancel := contextTimeout()
	defer cancel()
	cfg, item, errText := b.ticketForButton(ctx, i, id)
	if errText != "" {
		_ = respondEphemeral(s, i, errText)
		return
	}
	if item.Status != domain.TicketOpen {
		_ = respondEphemeral(s, i, "This ticket is already closed.")
		return
	}
	if !b.memberIsTicketStaff(ctx, i, cfg.SupportRoleID) && userID != item.OpenerID {
		_ = respondEphemeral(s, i, "You don't have permission to do that.")
		return
	}
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: domain.TicketCustomID("close-form", id),
			Title:    clipModalTitle(fmt.Sprintf("Close ticket %d", item.Number)),
			Components: []discordgo.MessageComponent{
				modalLabel{Label: "Reason", Component: modalText{
					CustomID: "reason", Style: discordgo.TextInputShort,
					Placeholder: "What was the outcome?", Required: true, MinLength: 1, MaxLength: domain.MaxTicketReason,
				}},
			},
		},
	})
	if err != nil {
		b.log.Warn("ticket close form", "guild", i.GuildID, "err", err)
	}
}

func (b *Bot) submitCloseTicket(s *discordgo.Session, i *discordgo.InteractionCreate, id int64, values map[string]string) {
	userID, name, ok := ticketMember(i)
	if !ok {
		_ = respondEphemeral(s, i, "Use this form in a server.")
		return
	}
	reason, err := domain.CleanTicketField(values["reason"], domain.MaxTicketReason, "Write a reason. It is saved with the transcript.", true)
	if err != nil {
		_ = respondEphemeral(s, i, err.Error())
		return
	}
	if err := deferReply(s, i, true); err != nil {
		return
	}
	ctx, cancel := contextTimeout()
	defer cancel()
	cfg, item, errText := b.ticketForButton(ctx, i, id)
	if errText != "" {
		editReply(s, i, errText)
		return
	}
	if item.Status != domain.TicketOpen {
		editReply(s, i, "This ticket is already closed.")
		return
	}
	if !b.memberIsTicketStaff(ctx, i, cfg.SupportRoleID) && userID != item.OpenerID {
		editReply(s, i, "You don't have permission to do that.")
		return
	}
	transcript := b.ticketTranscript(item, name, reason)
	item, closed, err := b.store.CloseTicket(ctx, i.GuildID, id, userID, name, reason, transcript)
	if err != nil {
		b.log.Error("ticket close", "guild", i.GuildID, "err", err)
		editReply(s, i, "I could not close that ticket.")
		return
	}
	if !closed {
		editReply(s, i, "This ticket is already closed.")
		return
	}
	if err := b.syncTicketMessage(cfg, item); err != nil {
		b.log.Warn("ticket message", "guild", i.GuildID, "err", err)
	}
	if err := b.applyTicketAccess(item.ChannelID, i.GuildID, cfg.SupportRoleID, item.OpenerID, false); err != nil {
		b.log.Warn("ticket lock", "guild", i.GuildID, "err", err)
	}
	reply := fmt.Sprintf("Closed ticket %d. The transcript is saved. This channel will be deleted in %d seconds.", item.Number, domain.TicketDeleteSeconds)
	if domain.ValidSnowflake(cfg.LogChannelID) {
		if err := b.postTicketTranscript(cfg.LogChannelID, item, transcript); err != nil {
			b.log.Warn("ticket transcript", "guild", i.GuildID, "err", err)
			reply = fmt.Sprintf("Closed ticket %d. The transcript is saved in the dashboard. I could not post it to the transcript channel. This channel will be deleted in %d seconds.", item.Number, domain.TicketDeleteSeconds)
		}
	}
	editReply(s, i, reply)
	go b.removeClosedTicket(i.GuildID, id, item.ChannelID)
}

func (b *Bot) removeClosedTicket(guildID string, id int64, channelID string) {
	timer := time.NewTimer(time.Duration(domain.TicketDeleteSeconds) * time.Second)
	defer timer.Stop()
	<-timer.C
	ctx, cancel := contextTimeout()
	defer cancel()
	if _, err := b.finishClosedTicket(ctx, guildID, id, channelID); err != nil {
		b.log.Warn("ticket channel delete", "guild", guildID, "ticket", id, "channel", channelID, "err", err)
	}
}

func (b *Bot) deleteTicket(s *discordgo.Session, i *discordgo.InteractionCreate, id int64) {
	if _, _, ok := ticketMember(i); !ok {
		_ = respondEphemeral(s, i, "Use this button in a server.")
		return
	}
	if err := deferComponent(s, i); err != nil {
		return
	}
	ctx, cancel := contextTimeout()
	defer cancel()
	cfg, item, errText := b.ticketForButton(ctx, i, id)
	if errText != "" {
		followupEphemeral(s, i, errText)
		return
	}
	if !b.memberIsTicketStaff(ctx, i, cfg.SupportRoleID) {
		followupEphemeral(s, i, "You don't have permission to do that.")
		return
	}
	if item.Status != domain.TicketClosed {
		if item.Status == domain.TicketOpen {
			followupEphemeral(s, i, "Close the ticket first. Closing saves the transcript.")
			return
		}
		followupEphemeral(s, i, "That ticket channel was already deleted.")
		return
	}
	removed, err := b.finishClosedTicket(ctx, i.GuildID, id, item.ChannelID)
	if err != nil {
		if statusOf(err) == 0 {
			b.log.Error("ticket delete", "guild", i.GuildID, "ticket", id, "err", err)
			followupEphemeral(s, i, "I could not delete that ticket.")
			return
		}
		followupEphemeral(s, i, "Discord refused to delete that channel. The transcript is still saved.")
		return
	}
	if !removed {
		current, getErr := b.store.GetTicket(ctx, i.GuildID, id)
		if getErr == nil && current.Status == domain.TicketOpen {
			followupEphemeral(s, i, "This ticket is already open.")
			return
		}
		followupEphemeral(s, i, "That ticket channel was already deleted.")
		return
	}
	followupEphemeral(s, i, fmt.Sprintf("Deleted ticket %d. The transcript stays in the dashboard.", item.Number))
}

func (b *Bot) finishClosedTicket(ctx context.Context, guildID string, id int64, channelID string) (bool, error) {
	deleted, err := b.store.MarkTicketDeleted(ctx, guildID, id)
	if err != nil || !deleted {
		return false, err
	}
	if channelID == "" {
		return true, nil
	}
	if _, err := b.session.ChannelDelete(channelID); err != nil && statusOf(err) != 404 {
		if _, restoreErr := b.store.UnmarkTicketDeleted(ctx, guildID, id); restoreErr != nil {
			b.log.Error("ticket delete restore", "guild", guildID, "ticket", id, "err", restoreErr)
		}
		return false, err
	}
	return true, nil
}

func (b *Bot) submitDeleteTicket(s *discordgo.Session, i *discordgo.InteractionCreate, id int64, values map[string]string) {
	if _, _, ok := ticketMember(i); !ok {
		_ = respondEphemeral(s, i, "Use this form in a server.")
		return
	}
	if err := deferReply(s, i, true); err != nil {
		return
	}
	ctx, cancel := contextTimeout()
	defer cancel()
	cfg, item, errText := b.ticketForButton(ctx, i, id)
	if errText != "" {
		editReply(s, i, errText)
		return
	}
	if !b.memberIsTicketStaff(ctx, i, cfg.SupportRoleID) {
		editReply(s, i, "You don't have permission to do that.")
		return
	}
	if item.Status != domain.TicketClosed {
		if item.Status == domain.TicketOpen {
			editReply(s, i, "Close the ticket first. Closing saves the transcript.")
			return
		}
		editReply(s, i, "That ticket channel was already deleted.")
		return
	}
	typed := values["number"]
	if !domain.TicketNumberMatches(typed, item.Number) {
		editReply(s, i, fmt.Sprintf("Type %d to delete this ticket. The channel is still there.", item.Number))
		return
	}
	removed, err := b.finishClosedTicket(ctx, i.GuildID, id, item.ChannelID)
	if err != nil {
		if statusOf(err) == 0 {
			b.log.Error("ticket delete", "guild", i.GuildID, "ticket", id, "err", err)
			editReply(s, i, "I could not delete that ticket.")
			return
		}
		editReply(s, i, "Discord refused to delete that channel. The transcript is still saved.")
		return
	}
	if !removed {
		editReply(s, i, "That ticket channel was already deleted.")
		return
	}
	editReply(s, i, fmt.Sprintf("Deleted ticket %d. The transcript stays in the dashboard.", item.Number))
}

func (b *Bot) readyTicketConfig(ctx context.Context, guildID string) (store.TicketConfig, error) {
	if _, err := b.touchGuild(ctx, guildID); err != nil {
		b.log.Error("ticket guild", "guild", guildID, "err", err)
		return store.TicketConfig{}, errors.New("I could not load this server.")
	}
	cfg, err := b.store.GetTicketConfig(ctx, guildID)
	if err != nil {
		b.log.Error("ticket config", "guild", guildID, "err", err)
		return store.TicketConfig{}, errors.New("I could not load tickets for this server.")
	}
	if err := domain.TicketOpenReady(cfg.CategoryID, cfg.SupportRoleID); err != nil {
		return store.TicketConfig{}, err
	}
	return cfg, nil
}

func (b *Bot) rememberTicketPanel(ctx context.Context, i *discordgo.InteractionCreate, cfg store.TicketConfig) {
	if i.Message == nil {
		return
	}
	channelID := i.ChannelID
	messageID := i.Message.ID
	if !domain.ValidSnowflake(channelID) || !domain.ValidSnowflake(messageID) {
		return
	}
	if cfg.PanelChannelID == channelID && cfg.PanelMessageID == messageID {
		return
	}
	if err := b.store.SaveTicketPanelLocation(ctx, i.GuildID, channelID, messageID); err != nil {
		b.log.Warn("ticket panel location", "guild", i.GuildID, "err", err)
	}
}

func (b *Bot) requireTicketPlaces(guildID string, cfg store.TicketConfig) error {
	channel, found, err := b.guildChannel(guildID, cfg.CategoryID)
	if err != nil {
		return errors.New("I could not read this server's channels.")
	}
	if !found || channel.Type != discordgo.ChannelTypeGuildCategory {
		return errors.New("Choose a category for ticket channels.")
	}
	ok, err := b.roleExists(guildID, cfg.SupportRoleID)
	if err != nil {
		return errors.New("I could not read this server's roles.")
	}
	if !ok {
		return errors.New("The support role is missing. Pick it again in the dashboard.")
	}
	return nil
}

func (b *Bot) retireStaleTicket(ctx context.Context, i *discordgo.InteractionCreate, userID string) (int, string) {
	existing, err := b.store.OpenTicketFor(ctx, i.GuildID, userID)
	if errors.Is(err, store.ErrNotFound) {
		return 0, ""
	}
	if err != nil {
		b.log.Error("ticket lookup", "guild", i.GuildID, "err", err)
		return 0, "I could not check your open ticket."
	}
	found, lookErr := b.channelPresent(i.GuildID, existing.ChannelID)
	if lookErr != nil {
		return 0, "I could not check your open ticket."
	}
	if found {
		return 0, fmt.Sprintf("You already have ticket %d open: <#%s>", existing.Number, existing.ChannelID)
	}
	if err := b.store.RetireMissingTicket(ctx, i.GuildID, existing.ID); err != nil {
		b.log.Error("ticket retire", "guild", i.GuildID, "err", err)
		return 0, "I could not close the ticket whose channel is gone."
	}
	return existing.Number, ""
}

func (b *Bot) createTicketChannel(guildID string, cfg store.TicketConfig, item store.Ticket, openerID string) (string, error) {
	overwrites, err := b.ticketOverwrites(guildID, cfg.SupportRoleID, openerID, true)
	if err != nil {
		return "", err
	}
	channel, err := b.session.GuildChannelCreateComplex(guildID, discordgo.GuildChannelCreateData{
		Name:                 fmt.Sprintf("ticket-%d", item.Number),
		Type:                 discordgo.ChannelTypeGuildText,
		Topic:                clipName(fmt.Sprintf("Ticket %d: %s", item.Number, item.Subject)),
		ParentID:             cfg.CategoryID,
		PermissionOverwrites: overwrites,
	})
	if err != nil {
		return "", errors.New(channelCreateError(err))
	}
	return channel.ID, nil
}

func (b *Bot) sendTicketControl(cfg store.TicketConfig, item store.Ticket) (string, error) {
	content := domain.TicketChannelText(ticketView(cfg, item))
	components := ticketComponents(item)
	sent, err := b.session.ChannelMessageSendComplex(item.ChannelID, &discordgo.MessageSend{
		Content: content, Components: components, AllowedMentions: ticketMentions(item.OpenerID, item.ClaimerID, cfg.SupportRoleID),
	})
	if err != nil {
		switch statusOf(err) {
		case 403:
			return "", errors.New("I could not post in the new ticket channel.")
		default:
			return "", errors.New("Discord refused the ticket message.")
		}
	}
	return sent.ID, nil
}

func (b *Bot) presentTicket(s *discordgo.Session, i *discordgo.InteractionCreate, cfg store.TicketConfig, item store.Ticket) error {
	content := domain.TicketChannelText(ticketView(cfg, item))
	components := ticketComponents(item)
	mentions := ticketMentions(item.OpenerID, item.ClaimerID, cfg.SupportRoleID)
	clicked := ""
	if i != nil && i.Message != nil {
		clicked = i.Message.ID
	}
	if clicked != "" {
		_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &content, Components: &components, AllowedMentions: mentions,
		})
		if err != nil {
			b.log.Warn("ticket interaction edit", "guild", item.GuildID, "err", err)
		} else if clicked == item.ControlMessageID {
			return nil
		}
	}
	return b.syncTicketMessage(cfg, item)
}

func (b *Bot) syncTicketMessage(cfg store.TicketConfig, item store.Ticket) error {
	content := domain.TicketChannelText(ticketView(cfg, item))
	components := ticketComponents(item)
	mentions := ticketMentions(item.OpenerID, item.ClaimerID, cfg.SupportRoleID)
	if item.ControlMessageID != "" {
		edit := discordgo.NewMessageEdit(item.ChannelID, item.ControlMessageID)
		edit.Content = &content
		edit.Components = &components
		edit.AllowedMentions = mentions
		if _, err := b.session.ChannelMessageEditComplex(edit); err == nil {
			return nil
		} else if statusOf(err) != 404 {
			return err
		}
	}
	sent, err := b.session.ChannelMessageSendComplex(item.ChannelID, &discordgo.MessageSend{
		Content: content, Components: components, AllowedMentions: mentions,
	})
	if err != nil {
		return err
	}
	ctx, cancel := contextTimeout()
	defer cancel()
	return b.store.SetTicketControl(ctx, item.GuildID, item.ID, sent.ID)
}

func (b *Bot) applyTicketAccess(channelID, guildID, supportRoleID, openerID string, chat bool) error {
	if channelID == "" {
		return nil
	}
	overwrites, err := b.ticketOverwrites(guildID, supportRoleID, openerID, chat)
	if err != nil {
		return err
	}
	_, err = b.session.ChannelEditComplex(channelID, &discordgo.ChannelEdit{PermissionOverwrites: overwrites})
	return err
}

func (b *Bot) ticketOverwrites(guildID, supportRoleID, openerID string, chat bool) ([]*discordgo.PermissionOverwrite, error) {
	botRoleID, err := b.ownRoleID(guildID)
	if err != nil {
		return nil, err
	}
	view := int64(discordgo.PermissionViewChannel)
	read := int64(discordgo.PermissionReadMessageHistory)
	send := int64(discordgo.PermissionSendMessages)
	files := int64(discordgo.PermissionAttachFiles | discordgo.PermissionEmbedLinks)
	talk := view | read | files
	var deny int64
	if chat {
		talk |= send
	} else {
		deny = send
	}
	overwrites := []*discordgo.PermissionOverwrite{{
		ID: guildID, Type: discordgo.PermissionOverwriteTypeRole, Deny: view,
	}}
	seen := map[string]bool{guildID: true}
	add := func(id string, kind discordgo.PermissionOverwriteType, allow, denyBits int64) {
		if id == "" || seen[id] || !domain.ValidSnowflake(id) {
			return
		}
		seen[id] = true
		overwrites = append(overwrites, &discordgo.PermissionOverwrite{
			ID: id, Type: kind, Allow: allow, Deny: denyBits,
		})
	}
	add(botRoleID, discordgo.PermissionOverwriteTypeRole, talk|send|int64(discordgo.PermissionManageChannels), 0)
	add(supportRoleID, discordgo.PermissionOverwriteTypeRole, talk, deny)
	add(openerID, discordgo.PermissionOverwriteTypeMember, talk, deny)
	return overwrites, nil
}

func (b *Bot) ticketTranscript(item store.Ticket, closedBy, reason string) string {
	lines, note := b.ticketLines(item)
	return domain.FormatTicketTranscript(domain.TicketTranscript{
		Number: item.Number, OpenerName: item.OpenerName, Subject: item.Subject, Body: item.Body,
		ClosedBy: closedBy, Reason: reason, Note: note, Lines: lines,
	})
}

func (b *Bot) ticketLines(item store.Ticket) ([]string, string) {
	if item.ChannelID == "" {
		return nil, "The channel was already gone."
	}
	messages, err := b.session.ChannelMessages(item.ChannelID, 100, "", "", "")
	if err != nil {
		b.log.Warn("ticket messages", "guild", item.GuildID, "channel", item.ChannelID, "err", err)
		return nil, "The channel messages could not be read."
	}
	lines := make([]string, 0, len(messages))
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg == nil || msg.ID == item.ControlMessageID {
			continue
		}
		text := strings.TrimSpace(msg.Content)
		for _, file := range msg.Attachments {
			if file == nil || file.Filename == "" {
				continue
			}
			if text != "" {
				text += " "
			}
			text += "[file: " + file.Filename + "]"
		}
		if text == "" {
			continue
		}
		cleaned, err := domain.CleanTicketField(text, 4000, "", false)
		if err != nil || cleaned == "" {
			cleaned = "[message omitted]"
		}
		author := "unknown"
		if msg.Author != nil {
			author = clipName(display(msg.Author, msg.Member))
		}
		lines = append(lines, msg.Timestamp.UTC().Format("2006-01-02 15:04")+" "+author+": "+cleaned)
	}
	note := ""
	if len(messages) == 100 {
		note = "This copy includes the latest 100 messages."
	}
	return lines, note
}

func (b *Bot) postTicketTranscript(channelID string, item store.Ticket, transcript string) error {
	summary := fmt.Sprintf("Ticket %d is closed.\nOpened by %s.\nClosed by %s.\nReason: %s\nThe transcript is attached.",
		item.Number, item.OpenerName, item.ClosedByName, item.CloseReason)
	_, err := b.session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content: summary,
		Files: []*discordgo.File{{
			Name:        fmt.Sprintf("ticket-%d.txt", item.Number),
			ContentType: "text/plain; charset=utf-8",
			Reader:      strings.NewReader(transcript),
		}},
		AllowedMentions: noMentions,
	})
	if err != nil {
		return err
	}
	return nil
}

func (b *Bot) ticketForButton(ctx context.Context, i *discordgo.InteractionCreate, id int64) (store.TicketConfig, store.Ticket, string) {
	cfg, err := b.store.GetTicketConfig(ctx, i.GuildID)
	if err != nil {
		b.log.Error("ticket config", "guild", i.GuildID, "err", err)
		return store.TicketConfig{}, store.Ticket{}, "I could not load tickets for this server."
	}
	item, err := b.store.GetTicket(ctx, i.GuildID, id)
	if errors.Is(err, store.ErrNotFound) {
		return cfg, store.Ticket{}, "That ticket is gone."
	}
	if err != nil {
		b.log.Error("ticket load", "guild", i.GuildID, "err", err)
		return cfg, store.Ticket{}, "I could not load that ticket."
	}
	if item.ChannelID != "" && i.ChannelID != "" && item.ChannelID != i.ChannelID {
		return cfg, item, "Use the buttons in the ticket channel."
	}
	return cfg, item, ""
}

func (b *Bot) memberIsTicketStaff(ctx context.Context, i *discordgo.InteractionCreate, supportRoleID string) bool {
	if i.Member == nil {
		return false
	}
	if domain.HasPermission(i.Member.Permissions, domain.PermAdministrator) {
		return true
	}
	if domain.ContainsID(i.Member.Roles, supportRoleID) {
		return true
	}
	guild, err := b.store.GetGuild(ctx, i.GuildID)
	if err != nil {
		return false
	}
	if guild.ModRoleID != "" && domain.ContainsID(i.Member.Roles, guild.ModRoleID) {
		return true
	}
	if guild.AdminRoleID != "" && domain.ContainsID(i.Member.Roles, guild.AdminRoleID) {
		return true
	}
	return false
}

func (b *Bot) channelPresent(guildID, channelID string) (bool, error) {
	_, found, err := b.guildChannel(guildID, channelID)
	return found, err
}

func (b *Bot) guildChannel(guildID, id string) (*discordgo.Channel, bool, error) {
	if !domain.ValidSnowflake(id) {
		return nil, false, nil
	}
	channels, err := b.session.GuildChannels(guildID)
	if err != nil {
		return nil, false, err
	}
	for _, channel := range channels {
		if channel != nil && channel.ID == id {
			return channel, true, nil
		}
	}
	return nil, false, nil
}

func (b *Bot) roleExists(guildID, roleID string) (bool, error) {
	if !domain.ValidSnowflake(roleID) {
		return false, nil
	}
	roles, err := b.session.GuildRoles(guildID)
	if err != nil {
		return false, err
	}
	for _, role := range roles {
		if role != nil && role.ID == roleID {
			return true, nil
		}
	}
	return false, nil
}

func ticketMember(i *discordgo.InteractionCreate) (string, string, bool) {
	if i.GuildID == "" || i.Member == nil || i.Member.User == nil || i.Member.User.Bot {
		return "", "", false
	}
	return i.Member.User.ID, clipName(display(i.Member.User, i.Member)), true
}

func ticketView(cfg store.TicketConfig, item store.Ticket) domain.TicketView {
	return domain.TicketView{
		Number: item.Number, OpenerID: item.OpenerID, SupportRoleID: cfg.SupportRoleID,
		Subject: item.Subject, Body: item.Body, Status: item.Status, ClaimerID: item.ClaimerID,
		ClosedByName: item.ClosedByName, CloseReason: item.CloseReason,
	}
}

func ticketMentions(openerID, claimerID, roleID string) *discordgo.MessageAllowedMentions {
	var users []string
	if domain.ValidSnowflake(openerID) {
		users = append(users, openerID)
	}
	if domain.ValidSnowflake(claimerID) && claimerID != openerID {
		users = append(users, claimerID)
	}
	var roles []string
	if domain.ValidSnowflake(roleID) {
		roles = append(roles, roleID)
	}
	return &discordgo.MessageAllowedMentions{Parse: []discordgo.AllowedMentionType{}, Users: users, Roles: roles}
}

func ticketPanelComponents() []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{Label: "Open a ticket", Style: discordgo.PrimaryButton, CustomID: domain.TicketCustomID("open", 0)},
		}},
	}
}

func ticketOpenModal() *discordgo.InteractionResponse {
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID:   domain.TicketCustomID("form", 0),
			Title:      "Open a ticket",
			Components: ticketOpenFields(),
		},
	}
}

func ticketOpenFields() []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		modalLabel{Label: "Subject", Component: modalText{
			CustomID: "subject", Style: discordgo.TextInputShort,
			Placeholder: "Billing, a bug, a question", Required: true,
			MinLength: 1, MaxLength: domain.MaxTicketSubject,
		}},
		modalLabel{Label: "What do you need help with?", Component: modalText{
			CustomID: "details", Style: discordgo.TextInputParagraph,
			Placeholder: "What happened, and what you already tried", Required: true,
			MinLength: 1, MaxLength: domain.MaxTicketBody,
		}},
	}
}

type modalText struct {
	CustomID    string
	Style       discordgo.TextInputStyle
	Placeholder string
	MinLength   int
	MaxLength   int
	Required    bool
}

func (modalText) Type() discordgo.ComponentType { return discordgo.TextInputComponent }

func (m modalText) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type        int    `json:"type"`
		CustomID    string `json:"custom_id"`
		Style       int    `json:"style"`
		Placeholder string `json:"placeholder,omitempty"`
		Required    bool   `json:"required"`
		MinLength   int    `json:"min_length,omitempty"`
		MaxLength   int    `json:"max_length,omitempty"`
	}{
		Type:        int(discordgo.TextInputComponent),
		CustomID:    m.CustomID,
		Style:       int(m.Style),
		Placeholder: m.Placeholder,
		Required:    m.Required,
		MinLength:   m.MinLength,
		MaxLength:   m.MaxLength,
	})
}

func (b *Bot) submitTicketFromGateway(s *discordgo.Session, interaction *discordgo.Interaction, customID string, values map[string]string) {
	i := &discordgo.InteractionCreate{Interaction: interaction}
	action, id, ok := domain.ParseTicketAction(customID)
	if !ok {
		_ = respondEphemeral(s, i, "That form is not a ticket.")
		return
	}
	switch action {
	case "form":
		b.submitTicket(s, i, values)
	case "close-form":
		b.submitCloseTicket(s, i, id, values)
	case "delete-form":
		b.submitDeleteTicket(s, i, id, values)
	default:
		_ = respondEphemeral(s, i, "That form is not a ticket.")
	}
}

func ticketFormFromEvent(raw []byte) (*discordgo.Interaction, string, map[string]string, bool) {
	var payload struct {
		ID        string                    `json:"id"`
		AppID     string                    `json:"application_id"`
		Type      discordgo.InteractionType `json:"type"`
		GuildID   string                    `json:"guild_id"`
		ChannelID string                    `json:"channel_id"`
		Token     string                    `json:"token"`
		Member    *discordgo.Member         `json:"member"`
		User      *discordgo.User           `json:"user"`
		Data      struct {
			CustomID   string          `json:"custom_id"`
			Components json.RawMessage `json:"components"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, "", nil, false
	}
	if payload.Type != discordgo.InteractionModalSubmit || !strings.HasPrefix(payload.Data.CustomID, "sb:ticket:") {
		return nil, "", nil, false
	}
	if !componentTreeHasType(payload.Data.Components, 18) {
		return nil, "", nil, false
	}
	values := map[string]string{}
	collectModalText(payload.Data.Components, values)
	return &discordgo.Interaction{
		ID:        payload.ID,
		AppID:     payload.AppID,
		Type:      payload.Type,
		GuildID:   payload.GuildID,
		ChannelID: payload.ChannelID,
		Token:     payload.Token,
		Member:    payload.Member,
		User:      payload.User,
	}, payload.Data.CustomID, values, true
}

func componentTreeHasType(raw json.RawMessage, want int) bool {
	if len(raw) == 0 || string(raw) == "null" {
		return false
	}
	if raw[0] == '[' {
		var items []json.RawMessage
		if json.Unmarshal(raw, &items) != nil {
			return false
		}
		for _, item := range items {
			if componentTreeHasType(item, want) {
				return true
			}
		}
		return false
	}
	var part struct {
		Type       int             `json:"type"`
		Component  json.RawMessage `json:"component"`
		Components json.RawMessage `json:"components"`
	}
	if json.Unmarshal(raw, &part) != nil {
		return false
	}
	return part.Type == want || componentTreeHasType(part.Component, want) || componentTreeHasType(part.Components, want)
}

func collectModalText(raw json.RawMessage, values map[string]string) {
	if len(raw) == 0 || string(raw) == "null" {
		return
	}
	if raw[0] == '[' {
		var items []json.RawMessage
		if json.Unmarshal(raw, &items) != nil {
			return
		}
		for _, item := range items {
			collectModalText(item, values)
		}
		return
	}
	var part struct {
		CustomID   string          `json:"custom_id"`
		Value      *string         `json:"value"`
		Component  json.RawMessage `json:"component"`
		Components json.RawMessage `json:"components"`
	}
	if json.Unmarshal(raw, &part) != nil {
		return
	}
	if part.CustomID != "" && part.Value != nil {
		values[part.CustomID] = *part.Value
	}
	collectModalText(part.Component, values)
	collectModalText(part.Components, values)
}

func ticketComponents(item store.Ticket) []discordgo.MessageComponent {
	if item.Status == domain.TicketClosed {
		return []discordgo.MessageComponent{
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				discordgo.Button{Label: "Reopen", Style: discordgo.PrimaryButton, CustomID: domain.TicketCustomID("reopen", item.ID)},
			}},
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				discordgo.Button{Label: "Delete ticket", Style: discordgo.DangerButton, CustomID: domain.TicketCustomID("delete", item.ID)},
			}},
		}
	}
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{Label: "Claim", Style: discordgo.PrimaryButton, CustomID: domain.TicketCustomID("claim", item.ID)},
			discordgo.Button{Label: "Release", Style: discordgo.SecondaryButton, CustomID: domain.TicketCustomID("release", item.ID)},
		}},
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{Label: "Close", Style: discordgo.DangerButton, CustomID: domain.TicketCustomID("close", item.ID)},
		}},
	}
}

func claimBlocked(item store.Ticket, userID string) string {
	if item.Status != domain.TicketOpen {
		return "This ticket is already closed."
	}
	if item.ClaimerID == userID {
		return "You already claimed this ticket."
	}
	if item.ClaimerName != "" {
		return item.ClaimerName + " claimed this ticket. Release it if you are taking over."
	}
	return "Someone already claimed this ticket. Release it if you are taking over."
}

func reopenBlocked(item store.Ticket) string {
	if item.Status == domain.TicketDeleted {
		return "That ticket channel was already deleted."
	}
	if item.Status == domain.TicketOpen {
		return "This ticket is already open."
	}
	return "I could not reopen that ticket."
}

func channelCreateError(err error) string {
	if statusOf(err) == 403 {
		return "I need Manage Channels, and my role should sit above the support role."
	}
	var rest *discordgo.RESTError
	if errors.As(err, &rest) && rest.Message != nil && (rest.Message.Code == 30011 || rest.Message.Code == 30013) {
		return "This category is full. Discord allows 50 channels in a category. Close or delete a ticket, then try again."
	}
	return "Discord refused that channel."
}

func clipModalTitle(title string) string {
	if len(title) <= 45 {
		return title
	}
	return title[:45]
}
