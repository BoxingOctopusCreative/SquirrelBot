package discord

import (
	"errors"
	"fmt"
	"time"

	"github.com/BoxingOctopus/squirrelbot/internal/domain"
	"github.com/BoxingOctopus/squirrelbot/internal/store"
	"github.com/bwmarrin/discordgo"
)

func (b *Bot) announce(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if _, ok := b.permit(i, access{perm: domain.PermManageGuild, mod: true, admin: true}); !ok {
		return
	}
	channel := optionChannel(i, "channel")
	title := optionString(i, "title")
	body := optionString(i, "body")
	if channel == nil || !domain.ValidSnowflake(channel.ID) {
		_ = respondEphemeral(s, i, "Choose a channel.")
		return
	}
	var when *time.Time
	if raw := optionString(i, "in"); raw != "" {
		length, err := domain.ParseDuration(raw)
		if err != nil {
			_ = respondEphemeral(s, i, err.Error())
			return
		}
		moment := time.Now().Add(length)
		when = &moment
	}
	if err := domain.ValidateAnnouncement(title, body, when, time.Now()); err != nil {
		_ = respondEphemeral(s, i, err.Error())
		return
	}
	ping := ""
	if role := optionRole(i, "ping"); role != nil && domain.ValidSnowflake(role.ID) && role.ID != i.GuildID {
		ping = role.ID
	}
	if err := deferReply(s, i, true); err != nil {
		return
	}
	actorID, actorName := actor(i)
	item := store.Announcement{
		GuildID: i.GuildID, ChannelID: channel.ID, AuthorID: actorID, AuthorName: actorName,
		Title: title, Body: body, PingRoleID: ping, ScheduledAt: when, Status: "scheduled",
	}
	ctx, cancel := contextTimeout()
	defer cancel()
	if when == nil {
		messageID, err := b.DeliverAnnouncement(item)
		if err != nil {
			editReply(s, i, discordFailure(err))
			return
		}
		item.Status = "sent"
		item.MessageID = messageID
		now := time.Now()
		item.SentAt = &now
		item.ScheduledAt = &now
	}
	created, err := b.store.CreateAnnouncement(ctx, item)
	if err != nil {
		b.log.Error("save announcement", "guild", i.GuildID, "err", err)
		editReply(s, i, "Discord accepted it, but I could not save the record.")
		return
	}
	if when == nil {
		editReply(s, i, "Announcement sent.")
		return
	}
	editReply(s, i, fmt.Sprintf("Announcement #%d will post <t:%d:R>.", created.ID, when.Unix()))
}

func (b *Bot) DeliverAnnouncement(item store.Announcement) (string, error) {
	embed := &discordgo.MessageEmbed{
		Title:       item.Title,
		Description: item.Body,
		Color:       colorCopper,
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer:      &discordgo.MessageEmbedFooter{Text: "Mod team announcement · " + item.AuthorName},
	}
	message := &discordgo.MessageSend{
		Embeds:          []*discordgo.MessageEmbed{embed},
		AllowedMentions: noMentions,
	}
	if item.PingRoleID != "" {
		message.Content = "<@&" + item.PingRoleID + ">"
		message.AllowedMentions = &discordgo.MessageAllowedMentions{Roles: []string{item.PingRoleID}}
	}
	sent, err := b.session.ChannelMessageSendComplex(item.ChannelID, message)
	if err != nil {
		return "", err
	}
	return sent.ID, nil
}

type GrantInput struct {
	GuildID    string
	UserID     string
	UserName   string
	RoleID     string
	ActorID    string
	ActorName  string
	ActorRoles []string
	CheckActor bool
	Reason     string
	Duration   time.Duration
}

func (b *Bot) GrantEscalation(in GrantInput) (store.Escalation, error) {
	ctx, cancel := contextTimeout()
	defer cancel()
	role, err := b.store.EscalationRole(ctx, in.GuildID, in.RoleID)
	if errors.Is(err, store.ErrNotFound) {
		return store.Escalation{}, errors.New("That role is not on the escalation allowlist. Add it in the dashboard.")
	}
	if err != nil {
		return store.Escalation{}, err
	}
	if err := domain.ValidateEscalation(in.Duration, time.Duration(role.MaxDurationSeconds)*time.Second); err != nil {
		return store.Escalation{}, err
	}
	if err := b.canAssign(in.GuildID, in.RoleID); err != nil {
		return store.Escalation{}, err
	}
	member, err := b.session.GuildMember(in.GuildID, in.UserID)
	if err != nil {
		return store.Escalation{}, errors.New(discordFailure(err))
	}
	if memberHas(member, in.RoleID) {
		return store.Escalation{}, errors.New("They already hold that role.")
	}
	if in.CheckActor {
		if err := b.outranks(in.GuildID, in.ActorID, in.UserID, in.ActorRoles, member.Roles); err != nil {
			return store.Escalation{}, err
		}
	}
	if _, err := b.store.ActiveEscalation(ctx, in.GuildID, in.UserID, in.RoleID); err == nil {
		return store.Escalation{}, errors.New("They already have an active grant for that role.")
	} else if !errors.Is(err, store.ErrNotFound) {
		return store.Escalation{}, err
	}
	if err := b.session.GuildMemberRoleAdd(in.GuildID, in.UserID, in.RoleID, discordgo.WithAuditLogReason(domain.AuditReason(in.Reason))); err != nil {
		return store.Escalation{}, errors.New(discordFailure(err))
	}
	item := store.Escalation{
		GuildID: in.GuildID, UserID: in.UserID, UserName: in.UserName, RoleID: in.RoleID, Tier: role.Tier,
		GrantedBy: in.ActorID, GrantedByName: in.ActorName, Reason: in.Reason, ExpiresAt: time.Now().Add(in.Duration),
	}
	created, err := b.store.CreateEscalation(ctx, item)
	if err != nil {
		_ = b.session.GuildMemberRoleRemove(in.GuildID, in.UserID, in.RoleID)
		return store.Escalation{}, err
	}
	seconds := int(in.Duration.Seconds())
	caseItem, caseErr := b.store.CreateCase(ctx, store.NewCase{
		GuildID: in.GuildID, Action: "escalate", TargetID: in.UserID, TargetName: in.UserName,
		ModeratorID: in.ActorID, ModeratorName: in.ActorName, Reason: in.Reason, DurationSeconds: &seconds,
	})
	guild, _ := b.store.GetGuild(ctx, in.GuildID)
	b.dm(in.UserID, fmt.Sprintf("You have a temporary %s role in %s until %s. Reason: %s",
		role.Tier, b.guildName(in.GuildID), created.ExpiresAt.Format(time.RFC1123), reasonOrNone(in.Reason)))
	if caseErr == nil {
		b.sendLog(guild.ModLogChannelID, &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("Case #%d · temporary %s", caseItem.CaseNumber, role.Tier),
			Description: fmt.Sprintf("<@%s> received <@&%s> until <t:%d:f>.\n%s", in.UserID, in.RoleID, created.ExpiresAt.Unix(), reasonOrNone(in.Reason)),
			Color:       colorCopper,
		})
	}
	return created, nil
}

func (b *Bot) RevokeEscalation(guildID string, id int64, actorID, actorName, reason string) error {
	ctx, cancel := contextTimeout()
	defer cancel()
	item, err := b.store.GetEscalation(ctx, guildID, id)
	if err != nil {
		return err
	}
	if item.Status != "active" {
		return errors.New("That grant is already over.")
	}
	if err := b.session.GuildMemberRoleRemove(guildID, item.UserID, item.RoleID, discordgo.WithAuditLogReason(domain.AuditReason(reason))); err != nil && statusOf(err) != 404 {
		return errors.New(discordFailure(err))
	}
	if err := b.store.MarkEscalationEnded(ctx, item.ID, "revoked", reason); err != nil {
		return err
	}
	_, _ = b.store.CreateCase(ctx, store.NewCase{
		GuildID: guildID, Action: "deescalate", TargetID: item.UserID, TargetName: item.UserName,
		ModeratorID: actorID, ModeratorName: actorName, Reason: reason,
	})
	b.dm(item.UserID, fmt.Sprintf("Your admin firecall access in %s was removed. %s", b.guildName(guildID), reasonOrNone(reason)))
	return nil
}

func (b *Bot) escalate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	role := optionRole(i, "role")
	if role == nil {
		_ = respondEphemeral(s, i, "Choose a role.")
		return
	}
	ctx, cancel := contextTimeout()
	defer cancel()
	allowed, err := b.store.EscalationRole(ctx, i.GuildID, role.ID)
	if err != nil && i.GuildID != "" {
		_ = b.store.EnsureGuild(ctx, i.GuildID, "", "")
		allowed, err = b.store.EscalationRole(ctx, i.GuildID, role.ID)
	}
	need := access{perm: domain.PermManageRoles, mod: true, admin: true}
	if err == nil && allowed.Tier == "admin" {
		need = access{perm: domain.PermAdministrator, admin: true}
	}
	if _, ok := b.permit(i, need); !ok {
		return
	}
	if errors.Is(err, store.ErrNotFound) {
		_ = respondEphemeral(s, i, "That role is not on the escalation allowlist. Add it in the dashboard.")
		return
	}
	if err != nil {
		_ = respondEphemeral(s, i, "I could not read the escalation allowlist.")
		return
	}
	target := optionUser(i, "user")
	if target == nil {
		_ = respondEphemeral(s, i, "Choose a member.")
		return
	}
	length, err := domain.ParseDuration(optionString(i, "duration"))
	if err != nil {
		_ = respondEphemeral(s, i, err.Error())
		return
	}
	reason, err := domain.CleanReason(optionString(i, "reason"))
	if err != nil || reason == "" {
		_ = respondEphemeral(s, i, "Add a reason.")
		return
	}
	if err := deferReply(s, i, true); err != nil {
		return
	}
	actorID, actorName := actor(i)
	member, memberErr := s.GuildMember(i.GuildID, target.ID)
	if memberErr != nil {
		editReply(s, i, discordFailure(memberErr))
		return
	}
	actorRoles := []string{}
	if i.Member != nil {
		actorRoles = i.Member.Roles
	}
	if err := b.outranks(i.GuildID, actorID, target.ID, actorRoles, member.Roles); err != nil {
		editReply(s, i, err.Error())
		return
	}
	created, err := b.GrantEscalation(GrantInput{
		GuildID: i.GuildID, UserID: target.ID, UserName: display(target, member), RoleID: role.ID,
		ActorID: actorID, ActorName: actorName, Reason: reason, Duration: length,
	})
	if err != nil {
		editReply(s, i, err.Error())
		return
	}
	editReply(s, i, fmt.Sprintf("Granted <@&%s> to <@%s> until <t:%d:f>.", role.ID, target.ID, created.ExpiresAt.Unix()))
}

func (b *Bot) deescalate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if _, ok := b.permit(i, access{perm: domain.PermManageRoles, mod: true, admin: true}); !ok {
		return
	}
	target := optionUser(i, "user")
	role := optionRole(i, "role")
	if target == nil || role == nil {
		_ = respondEphemeral(s, i, "Choose a member and a role.")
		return
	}
	reason, err := domain.CleanReason(optionString(i, "reason"))
	if err != nil {
		_ = respondEphemeral(s, i, err.Error())
		return
	}
	if err := deferReply(s, i, true); err != nil {
		return
	}
	ctx, cancel := contextTimeout()
	defer cancel()
	active, err := b.store.ActiveEscalation(ctx, i.GuildID, target.ID, role.ID)
	if errors.Is(err, store.ErrNotFound) {
		editReply(s, i, "That member does not have an active grant for that role.")
		return
	}
	if err != nil {
		editReply(s, i, "I could not load that grant.")
		return
	}
	actorID, actorName := actor(i)
	if err := b.RevokeEscalation(i.GuildID, active.ID, actorID, actorName, reason); err != nil {
		editReply(s, i, err.Error())
		return
	}
	editReply(s, i, fmt.Sprintf("Removed admin firecall access from <@%s>.", target.ID))
}

func (b *Bot) listEscalations(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if _, ok := b.permit(i, access{perm: domain.PermModerateMembers, mod: true, admin: true}); !ok {
		return
	}
	if err := deferReply(s, i, true); err != nil {
		return
	}
	ctx, cancel := contextTimeout()
	defer cancel()
	items, err := b.store.ListEscalations(ctx, i.GuildID)
	if err != nil {
		editReply(s, i, "I could not load admin firecall access.")
		return
	}
	body := ""
	for _, item := range items {
		if item.Status != "active" {
			continue
		}
		body += fmt.Sprintf("<@%s> · <@&%s> · %s · until <t:%d:R>\n", item.UserID, item.RoleID, item.Tier, item.ExpiresAt.Unix())
	}
	if body == "" {
		body = "No admin firecall access is active."
	}
	editReply(s, i, body)
}
