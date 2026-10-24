package discord

import (
	"context"
	"fmt"
	"time"

	"github.com/BoxingOctopus/squirrelbot/internal/domain"
	"github.com/BoxingOctopus/squirrelbot/internal/store"
	"github.com/bwmarrin/discordgo"
)

func (b *Bot) RunScheduler(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !b.Ready() {
				continue
			}
			b.sendDueAnnouncements(ctx)
			b.expireEscalations(ctx)
			b.warnGate(ctx)
			b.kickGate(ctx)
			if err := b.store.DeleteExpiredSessions(ctx); err != nil {
				b.log.Warn("expire sessions", "err", err)
			}
		}
	}
}

func (b *Bot) sendDueAnnouncements(ctx context.Context) {
	items, err := b.store.ClaimDueAnnouncements(ctx, 10)
	if err != nil {
		b.log.Error("claim announcements", "err", err)
		return
	}
	for _, item := range items {
		messageID, err := b.DeliverAnnouncement(item)
		if err != nil {
			b.log.Warn("send announcement", "id", item.ID, "err", err)
			_ = b.store.MarkAnnouncementFailed(ctx, item.ID, discordFailure(err))
			continue
		}
		if err := b.store.MarkAnnouncementSent(ctx, item.ID, messageID); err != nil {
			b.log.Error("record announcement", "id", item.ID, "err", err)
		}
	}
}

func (b *Bot) expireEscalations(ctx context.Context) {
	items, err := b.store.ClaimExpiredEscalations(ctx, 20)
	if err != nil {
		b.log.Error("claim escalations", "err", err)
		return
	}
	for _, item := range items {
		err := b.session.GuildMemberRoleRemove(item.GuildID, item.UserID, item.RoleID)
		status := "expired"
		end := "Admin firecall access reached its deadline."
		if err != nil && statusOf(err) != 404 {
			status = "expire_failed"
			end = discordFailure(err)
			b.log.Warn("expire role", "id", item.ID, "err", err)
		}
		if err := b.store.MarkEscalationEnded(ctx, item.ID, status, end); err != nil {
			b.log.Error("close escalation", "id", item.ID, "err", err)
		}
		if status == "expired" {
			b.dm(item.UserID, fmt.Sprintf("Your temporary %s role in %s has ended.", item.Tier, b.guildName(item.GuildID)))
			guild, _ := b.store.GetGuild(ctx, item.GuildID)
			b.sendLog(guild.ModLogChannelID, &discordgo.MessageEmbed{
				Title:       fmt.Sprintf("Temporary %s ended", item.Tier),
				Description: fmt.Sprintf("<@%s> lost <@&%s>.", item.UserID, item.RoleID),
				Color:       colorMist,
			})
		}
	}
}

func (b *Bot) warnGate(ctx context.Context) {
	items, err := b.store.DueWarnings(ctx, 20)
	if err != nil {
		b.log.Error("gate warnings", "err", err)
		return
	}
	for _, item := range items {
		b.dm(item.UserID, fmt.Sprintf("You still need to finish the server application in %s. You will be removed at %s if you do not.",
			b.guildName(item.GuildID), item.DeadlineAt.Format(time.RFC1123)))
		if err := b.store.MarkGateWarned(ctx, item.GuildID, item.UserID); err != nil {
			b.log.Warn("mark gate warning", "user", item.UserID, "err", err)
		}
	}
}

func (b *Bot) kickGate(ctx context.Context) {
	items, err := b.store.ClaimDueKicks(ctx, 20)
	if err != nil {
		b.log.Error("claim kicks", "err", err)
		return
	}
	for _, item := range items {
		b.finishKick(ctx, item)
	}
}

func (b *Bot) finishKick(ctx context.Context, item store.DueGateMember) {
	member, err := b.session.GuildMember(item.GuildID, item.UserID)
	if err != nil {
		if statusOf(err) == 404 {
			_ = b.store.SetGateStatus(ctx, item.GuildID, item.UserID, "left")
			return
		}
		_ = b.store.SetGateStatus(ctx, item.GuildID, item.UserID, "pending")
		b.log.Warn("read gate member", "user", item.UserID, "err", err)
		return
	}
	if overlaps(member.Roles, item.ExemptRoleIDs) {
		_ = b.store.SetGateStatus(ctx, item.GuildID, item.UserID, "exempt")
		return
	}
	reason := item.KickReason
	if reason == "" {
		reason = "Did not finish the server application in time"
	}
	description := fmt.Sprintf("<@%s> did not finish the server application in time.", item.UserID)
	if err := b.removeForGate(ctx, item.GuildID, item.UserID, item.UserName, reason, item.LogChannelID, description); err != nil {
		if statusOf(err) == 404 {
			_ = b.store.SetGateStatus(ctx, item.GuildID, item.UserID, "left")
			return
		}
		_ = b.store.SetGateStatus(ctx, item.GuildID, item.UserID, "kick_failed")
		b.log.Warn("gate kick", "user", item.UserID, "err", err)
	}
}

func (b *Bot) removeForGate(ctx context.Context, guildID, userID, userName, reason, logChannelID, description string) error {
	if err := b.session.GuildMemberDeleteWithReason(guildID, userID, reason, discordgo.WithAuditLogReason(domain.AuditReason(reason))); err != nil {
		return err
	}
	_ = b.store.SetGateStatus(ctx, guildID, userID, "kicked")
	self := "0"
	if b.session.State != nil && b.session.State.User != nil {
		self = b.session.State.User.ID
	}
	created, caseErr := b.store.CreateCase(ctx, store.NewCase{
		GuildID: guildID, Action: "gate_kick", TargetID: userID, TargetName: userName,
		ModeratorID: self, ModeratorName: "SquirrelBot", Reason: reason,
	})
	if logChannelID == "" {
		if guild, err := b.store.GetGuild(ctx, guildID); err == nil {
			logChannelID = guild.ModLogChannelID
		}
	}
	title := "Application kick"
	if caseErr == nil {
		title = fmt.Sprintf("Case #%d · application kick", created.CaseNumber)
	}
	b.sendLog(logChannelID, &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       colorBerry,
	})
	b.dm(userID, fmt.Sprintf("You were removed from %s. %s", b.guildName(guildID), reason))
	return nil
}
