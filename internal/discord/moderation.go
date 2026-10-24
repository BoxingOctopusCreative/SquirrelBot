package discord

import (
	"fmt"
	"time"

	"github.com/BoxingOctopus/squirrelbot/internal/domain"
	"github.com/BoxingOctopus/squirrelbot/internal/store"
	"github.com/bwmarrin/discordgo"
)

func (b *Bot) moderate(s *discordgo.Session, i *discordgo.InteractionCreate, kind string) {
	need := access{mod: true, admin: true}
	switch kind {
	case "ban":
		need = access{perm: domain.PermBanMembers, admin: true}
	case "kick":
		need = access{perm: domain.PermKickMembers, mod: true, admin: true}
	default:
		need.perm = domain.PermModerateMembers
	}
	if _, ok := b.permit(i, need); !ok {
		return
	}
	target := optionUser(i, "user")
	if target == nil {
		_ = respondEphemeral(s, i, "Choose a member.")
		return
	}
	reason, err := domain.CleanReason(optionString(i, "reason"))
	if err != nil {
		_ = respondEphemeral(s, i, err.Error())
		return
	}
	if reason == "" && kind != "untimeout" {
		_ = respondEphemeral(s, i, "Add a reason.")
		return
	}
	var until time.Time
	if kind == "timeout" {
		length, err := domain.ParseDuration(optionString(i, "duration"))
		if err != nil {
			_ = respondEphemeral(s, i, err.Error())
			return
		}
		if err := domain.ValidateTimeout(length); err != nil {
			_ = respondEphemeral(s, i, err.Error())
			return
		}
		until = time.Now().Add(length)
	}
	if err := deferReply(s, i, false); err != nil {
		return
	}
	member, err := s.GuildMember(i.GuildID, target.ID)
	if err != nil {
		editReply(s, i, discordFailure(err))
		return
	}
	actorID, actorName := actor(i)
	actorRoles := []string{}
	if i.Member != nil {
		actorRoles = i.Member.Roles
	}
	if err := b.outranks(i.GuildID, actorID, target.ID, actorRoles, member.Roles); err != nil {
		editReply(s, i, err.Error())
		return
	}
	audit := discordgo.WithAuditLogReason(domain.AuditReason(reason))
	switch kind {
	case "kick":
		err = s.GuildMemberDeleteWithReason(i.GuildID, target.ID, reason, audit)
	case "ban":
		days := optionInt(i, "delete_days", 0)
		err = s.GuildBanCreateWithReason(i.GuildID, target.ID, reason, days, audit)
	case "timeout":
		err = s.GuildMemberTimeout(i.GuildID, target.ID, &until, audit)
	case "untimeout":
		err = s.GuildMemberTimeout(i.GuildID, target.ID, nil, audit)
	}
	if err != nil {
		editReply(s, i, discordFailure(err))
		return
	}
	var duration *int
	if kind == "timeout" {
		seconds := int(time.Until(until).Seconds())
		duration = &seconds
	}
	ctx, cancel := contextTimeout()
	defer cancel()
	created, err := b.store.CreateCase(ctx, store.NewCase{
		GuildID: i.GuildID, Action: kind, TargetID: target.ID, TargetName: display(target, member),
		ModeratorID: actorID, ModeratorName: actorName, Reason: reason, DurationSeconds: duration,
	})
	if err != nil {
		b.log.Error("create case", "guild", i.GuildID, "err", err)
		editReply(s, i, "The action succeeded, but I could not record the case.")
		return
	}
	b.dm(target.ID, fmt.Sprintf("A moderator used %s on you in %s. Reason: %s", kind, b.guildName(i.GuildID), reasonOrNone(reason)))
	guild, _ := b.store.GetGuild(ctx, i.GuildID)
	b.sendLog(guild.ModLogChannelID, &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Case #%d · %s", created.CaseNumber, kind),
		Description: fmt.Sprintf("<@%s>\n%s", target.ID, reasonOrNone(reason)),
		Color:       colorFor(kind),
	})
	editReply(s, i, caseLine(created.CaseNumber, kind, "<@"+target.ID+">", reason))
}

func (b *Bot) cases(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if _, ok := b.permit(i, access{perm: domain.PermModerateMembers, mod: true, admin: true}); !ok {
		return
	}
	target := optionUser(i, "user")
	if target == nil {
		_ = respondEphemeral(s, i, "Choose a member.")
		return
	}
	if err := deferReply(s, i, true); err != nil {
		return
	}
	ctx, cancel := contextTimeout()
	defer cancel()
	items, err := b.store.ListCasesForUser(ctx, i.GuildID, target.ID, 5)
	if err != nil {
		editReply(s, i, "I could not load cases.")
		return
	}
	if len(items) == 0 {
		editReply(s, i, "No cases for that member.")
		return
	}
	body := ""
	for _, item := range items {
		body += fmt.Sprintf("#%d %s · %s · %s\n", item.CaseNumber, item.Action, item.CreatedAt.Format(time.RFC822), reasonOrNone(item.Reason))
	}
	editReply(s, i, body)
}

func (b *Bot) purge(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if _, ok := b.permit(i, access{perm: domain.PermManageMessages, mod: true, admin: true}); !ok {
		return
	}
	count := optionInt(i, "count", 0)
	if count < 1 || count > 100 {
		_ = respondEphemeral(s, i, "Choose between 1 and 100 messages.")
		return
	}
	if err := deferReply(s, i, true); err != nil {
		return
	}
	messages, err := s.ChannelMessages(i.ChannelID, count, "", "", "")
	if err != nil {
		editReply(s, i, discordFailure(err))
		return
	}
	cutoff := time.Now().Add(-13 * 24 * time.Hour)
	ids := make([]string, 0, len(messages))
	for _, message := range messages {
		if message.Timestamp.Before(cutoff) {
			continue
		}
		ids = append(ids, message.ID)
		b.muted.mark(message.ID)
	}
	if len(ids) == 0 {
		editReply(s, i, "Those messages are older than 14 days, so Discord will not delete them in bulk.")
		return
	}
	if len(ids) == 1 {
		err = s.ChannelMessageDelete(i.ChannelID, ids[0])
	} else {
		err = s.ChannelMessagesBulkDelete(i.ChannelID, ids)
	}
	if err != nil {
		editReply(s, i, discordFailure(err))
		return
	}
	editReply(s, i, fmt.Sprintf("Deleted %d messages.", len(ids)))
}

func (b *Bot) slowmode(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if _, ok := b.permit(i, access{perm: domain.PermManageChannels, admin: true}); !ok {
		return
	}
	seconds := optionInt(i, "seconds", -1)
	if seconds < 0 || seconds > 21600 {
		_ = respondEphemeral(s, i, "Slowmode must be between 0 and 21600 seconds.")
		return
	}
	if err := deferReply(s, i, false); err != nil {
		return
	}
	_, err := s.ChannelEdit(i.ChannelID, &discordgo.ChannelEdit{RateLimitPerUser: &seconds})
	if err != nil {
		editReply(s, i, discordFailure(err))
		return
	}
	if seconds == 0 {
		editReply(s, i, "Slowmode is off in this channel.")
		return
	}
	editReply(s, i, fmt.Sprintf("Slowmode is %d seconds in this channel.", seconds))
}

func colorFor(kind string) int {
	switch kind {
	case "ban", "kick", "gate_kick":
		return colorBerry
	case "timeout", "warn":
		return colorCopper
	default:
		return colorMist
	}
}

func reasonOrNone(reason string) string {
	if reason == "" {
		return "No reason given."
	}
	return reason
}
