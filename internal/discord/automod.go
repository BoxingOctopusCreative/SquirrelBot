package discord

import (
	"strings"
	"sync"
	"time"

	"github.com/BoxingOctopus/squirrelbot/internal/domain"
	"github.com/BoxingOctopus/squirrelbot/internal/script"
	"github.com/BoxingOctopus/squirrelbot/internal/store"
	"github.com/bwmarrin/discordgo"
)

type spamTracker struct {
	mu   sync.Mutex
	hits map[string][]time.Time
}

func newSpamTracker() *spamTracker {
	return &spamTracker{hits: map[string][]time.Time{}}
}

func (t *spamTracker) burst(guildID, userID, content string, now time.Time) bool {
	content = strings.ToLower(strings.TrimSpace(content))
	if len(content) < 4 {
		return false
	}
	key := guildID + "\n" + userID + "\n" + content
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.hits) > 4000 {
		t.hits = map[string][]time.Time{}
	}
	cutoff := now.Add(-8 * time.Second)
	kept := make([]time.Time, 0, 6)
	for _, at := range t.hits[key] {
		if at.After(cutoff) {
			kept = append(kept, at)
		}
	}
	kept = append(kept, now)
	t.hits[key] = kept
	return len(kept) >= 5
}

type recentIDs struct {
	mu  sync.Mutex
	ids map[string]time.Time
}

func newRecentIDs() *recentIDs {
	return &recentIDs{ids: map[string]time.Time{}}
}

func (r *recentIDs) mark(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.ids) > 2000 {
		r.ids = map[string]time.Time{}
	}
	r.ids[id] = time.Now()
}

func (r *recentIDs) seen(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	at, ok := r.ids[id]
	if !ok {
		return false
	}
	if time.Since(at) > 2*time.Minute {
		delete(r.ids, id)
		return false
	}
	delete(r.ids, id)
	return true
}

func (b *Bot) onMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author == nil || m.Author.Bot || m.GuildID == "" {
		return
	}
	if b.customCommand(s, m) {
		return
	}
	b.automod(s, m)
}

func (b *Bot) customCommand(s *discordgo.Session, m *discordgo.MessageCreate) bool {
	ctx, cancel := contextTimeout()
	defer cancel()
	guild, err := b.store.GetGuild(ctx, m.GuildID)
	if err != nil || guild.Prefix == "" {
		return false
	}
	if !strings.HasPrefix(m.Content, guild.Prefix) {
		return false
	}
	name, args, argsText, ok := splitCommand(m.Content, guild.Prefix)
	if !ok || !domain.ValidCommandName(name) {
		return false
	}
	cmd, err := b.store.GetCommand(ctx, m.GuildID, name)
	if err != nil {
		return false
	}
	if cmd.Kind != script.KindLua {
		return sendPlain(s, m.ChannelID, cmd.Response) == nil
	}
	allowed, err := b.commandRoles(m.GuildID, cmd.RoleIDs)
	if err != nil {
		b.log.Warn("custom command roles", "guild", m.GuildID, "name", name, "err", err)
		_ = sendPlain(s, m.ChannelID, "That command failed. I could not read this server's roles.")
		return true
	}
	authorName, display, roles := commandAuthor(m)
	effects, err := script.Run(ctx, script.Input{
		Source:        cmd.Response,
		Kind:          cmd.Kind,
		Prefix:        guild.Prefix,
		Args:          args,
		ArgsText:      argsText,
		AuthorID:      m.Author.ID,
		AuthorName:    authorName,
		AuthorDisplay: display,
		AuthorRoleIDs: roles,
		ChannelID:     m.ChannelID,
		GuildID:       m.GuildID,
		GuildName:     guild.Name,
		Allowed:       allowed,
	})
	if err != nil {
		b.log.Warn("custom command", "guild", m.GuildID, "name", name, "err", err)
		_ = sendPlain(s, m.ChannelID, "That command failed. "+script.ErrorText(err))
		return true
	}
	for _, effect := range effects {
		if err := b.applyCommandEffect(s, m, effect); err != nil {
			b.log.Warn("custom command", "guild", m.GuildID, "name", name, "op", effect.Op, "err", err)
			_ = sendPlain(s, m.ChannelID, "That command failed. "+publicEffectError(err))
			return true
		}
	}
	return true
}

func (b *Bot) automod(s *discordgo.Session, m *discordgo.MessageCreate) {
	ctx, cancel := contextTimeout()
	defer cancel()
	cfg, err := b.store.GetAutomod(ctx, m.GuildID)
	if err != nil || !cfg.Enabled {
		return
	}
	if domain.ContainsID(cfg.IgnoredChannelIDs, m.ChannelID) {
		return
	}
	if m.Member != nil && overlaps(m.Member.Roles, cfg.IgnoredRoleIDs) {
		return
	}
	if b.memberIsStaff(m) {
		return
	}
	rules := domain.AutomodConfig{
		Enabled: cfg.Enabled, BlockInvites: cfg.BlockInvites, MentionLimit: cfg.MentionLimit,
		CapsPercent: cfg.CapsPercent, CapsMinLength: cfg.CapsMinLength, BadWords: cfg.BadWords,
	}
	hit, ok := domain.DetectAutomod(m.Content, len(m.Mentions), len(m.MentionRoles), m.MentionEveryone, rules)
	if !ok && cfg.BlockSpam && b.spam.burst(m.GuildID, m.Author.ID, m.Content, time.Now()) {
		hit = domain.AutomodHit{Rule: "spam", Detail: "repeated the same message"}
		ok = true
	}
	if !ok {
		return
	}
	b.muted.mark(m.ID)
	if err := s.ChannelMessageDelete(m.ChannelID, m.ID); err != nil {
		b.log.Warn("automod delete", "guild", m.GuildID, "err", err)
	}
	b.punishAutomod(m, cfg, hit)
}

func (b *Bot) memberIsStaff(m *discordgo.MessageCreate) bool {
	if m.Member == nil {
		return false
	}
	if domain.HasPermission(m.Member.Permissions, domain.PermManageMessages) {
		return true
	}
	guild, err := b.session.State.Guild(m.GuildID)
	if err != nil {
		guild, err = b.session.Guild(m.GuildID)
	}
	if err == nil && guild != nil {
		if guild.OwnerID == m.Author.ID {
			return true
		}
		var bits int64
		for _, role := range guild.Roles {
			if role.ID == m.GuildID || domain.ContainsID(m.Member.Roles, role.ID) {
				bits |= role.Permissions
			}
		}
		if domain.HasPermission(bits, domain.PermManageMessages) {
			return true
		}
	}
	ctx, cancel := contextTimeout()
	defer cancel()
	settings, err := b.store.GetGuild(ctx, m.GuildID)
	if err != nil {
		return false
	}
	return (settings.ModRoleID != "" && domain.ContainsID(m.Member.Roles, settings.ModRoleID)) ||
		(settings.AdminRoleID != "" && domain.ContainsID(m.Member.Roles, settings.AdminRoleID))
}

func (b *Bot) punishAutomod(m *discordgo.MessageCreate, cfg store.Automod, hit domain.AutomodHit) {
	ctx, cancel := contextTimeout()
	defer cancel()
	reason := "Automod: " + hit.Detail
	action := cfg.Action
	var duration *int
	switch action {
	case "timeout":
		seconds := cfg.TimeoutSeconds
		if seconds < 60 {
			seconds = 60
		}
		if seconds > int(domain.MaxDiscordTimeout.Seconds()) {
			seconds = int(domain.MaxDiscordTimeout.Seconds())
		}
		until := time.Now().Add(time.Duration(seconds) * time.Second)
		if err := b.session.GuildMemberTimeout(m.GuildID, m.Author.ID, &until); err != nil {
			b.log.Warn("automod timeout", "guild", m.GuildID, "err", err)
			action = "delete"
		} else {
			duration = &seconds
		}
	case "kick":
		if err := b.session.GuildMemberDeleteWithReason(m.GuildID, m.Author.ID, reason); err != nil {
			b.log.Warn("automod kick", "guild", m.GuildID, "err", err)
			action = "delete"
		}
	case "warn":
		b.dm(m.Author.ID, "A message you sent in "+b.guildName(m.GuildID)+" was removed. "+hit.Detail)
	default:
		action = "delete"
	}
	if action != "delete" {
		self := "0"
		if b.session.State.User != nil {
			self = b.session.State.User.ID
		}
		_, _ = b.store.CreateCase(ctx, store.NewCase{
			GuildID: m.GuildID, Action: "automod_" + action, TargetID: m.Author.ID,
			TargetName: display(m.Author, m.Member), ModeratorID: self, ModeratorName: "SquirrelBot",
			Reason: reason, DurationSeconds: duration,
		})
	}
	guild, _ := b.store.GetGuild(ctx, m.GuildID)
	b.sendLog(guild.ModLogChannelID, &discordgo.MessageEmbed{
		Title:       "Automod · " + hit.Rule,
		Description: "<@" + m.Author.ID + "> in <#" + m.ChannelID + ">\n" + hit.Detail,
		Color:       colorCopper,
	})
}
