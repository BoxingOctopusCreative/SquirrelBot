package discord

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/BoxingOctopus/squirrelbot/internal/domain"
	"github.com/BoxingOctopus/squirrelbot/internal/store"
	"github.com/bwmarrin/discordgo"
	"math/rand/v2"
)

type xpGrants struct {
	mu   sync.Mutex
	hits map[string][]time.Time
}

func newXPGrants() *xpGrants {
	return &xpGrants{hits: map[string][]time.Time{}}
}

func (g *xpGrants) allow(key string, now time.Time, count, seconds int) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.hits) > 20000 {
		g.hits = map[string][]time.Time{}
	}
	next, ok := domain.AllowGrant(g.hits[key], now, count, seconds)
	if len(next) == 0 {
		delete(g.hits, key)
	} else {
		g.hits[key] = next
	}
	return ok
}

type voiceSeat struct {
	channel string
	roles   []string
	known   bool
	name    string
}

type voiceRoster struct {
	mu    sync.Mutex
	seats map[string]map[string]voiceSeat
}

func newVoiceRoster() *voiceRoster {
	return &voiceRoster{seats: map[string]map[string]voiceSeat{}}
}

func (r *voiceRoster) set(guildID, userID, channel, name string, roles []string, known bool) {
	if guildID == "" || userID == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	users := r.seats[guildID]
	if channel == "" {
		if users != nil {
			delete(users, userID)
			if len(users) == 0 {
				delete(r.seats, guildID)
			}
		}
		return
	}
	if users == nil {
		users = map[string]voiceSeat{}
		r.seats[guildID] = users
	}
	seat := users[userID]
	seat.channel = channel
	if name != "" {
		seat.name = name
	}
	if known {
		seat.roles = append([]string(nil), roles...)
		seat.known = true
	}
	users[userID] = seat
}

func (r *voiceRoster) rememberRoles(guildID, userID, name string, roles []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	users := r.seats[guildID]
	if users == nil {
		return
	}
	seat, ok := users[userID]
	if !ok {
		return
	}
	seat.roles = append([]string(nil), roles...)
	seat.known = true
	if name != "" {
		seat.name = name
	}
	users[userID] = seat
}

func (r *voiceRoster) guilds() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, 0, len(r.seats))
	for guildID := range r.seats {
		out = append(out, guildID)
	}
	return out
}

type voicePerson struct {
	userID string
	name   string
	roles  []string
	known  bool
}

func (r *voiceRoster) channels(guildID string) map[string][]voicePerson {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := map[string][]voicePerson{}
	for userID, seat := range r.seats[guildID] {
		out[seat.channel] = append(out[seat.channel], voicePerson{
			userID: userID,
			name:   seat.name,
			roles:  append([]string(nil), seat.roles...),
			known:  seat.known,
		})
	}
	return out
}

func (b *Bot) onVoiceState(_ *discordgo.Session, event *discordgo.VoiceStateUpdate) {
	if event == nil {
		return
	}
	b.trackVoice(event.VoiceState)
}

func (b *Bot) seedVoice(states []*discordgo.VoiceState) {
	for _, state := range states {
		b.trackVoice(state)
	}
}

func (b *Bot) trackVoice(state *discordgo.VoiceState) {
	if b == nil || b.voices == nil || state == nil || state.GuildID == "" || state.UserID == "" {
		return
	}
	if b.session != nil && b.session.State != nil && b.session.State.User != nil && state.UserID == b.session.State.User.ID {
		return
	}
	name := ""
	var roles []string
	known := false
	if state.Member != nil {
		if state.Member.User != nil && state.Member.User.Bot {
			b.voices.set(state.GuildID, state.UserID, "", "", nil, false)
			return
		}
		name = display(state.Member.User, state.Member)
		roles = state.Member.Roles
		known = true
	}
	channel := ""
	if state.ChannelID != "" && !state.Deaf && !state.SelfDeaf {
		channel = state.ChannelID
	}
	b.voices.set(state.GuildID, state.UserID, channel, name, roles, known)
}

func (b *Bot) awardMessageXP(m *discordgo.MessageCreate) {
	if m == nil || m.Author == nil || m.GuildID == "" || m.WebhookID != "" {
		return
	}
	if m.Type != discordgo.MessageTypeDefault && m.Type != discordgo.MessageTypeReply {
		return
	}
	if strings.TrimSpace(m.Content) == "" && len(m.Attachments) == 0 && len(m.StickerItems) == 0 {
		return
	}
	ctx, cancel := contextTimeout()
	defer cancel()
	cfg, err := b.store.GetLevels(ctx, m.GuildID)
	if err != nil {
		b.log.Warn("levels", "guild", m.GuildID, "err", err)
		return
	}
	if !cfg.Enabled {
		return
	}
	if b.channelBlocked(m.ChannelID, cfg.IgnoredChannelIDs) {
		return
	}
	roles := []string(nil)
	if m.Member != nil {
		roles = m.Member.Roles
		if roles == nil {
			roles = []string{}
		}
	}
	if overlaps(roles, cfg.IgnoredRoleIDs) {
		return
	}
	if b.xp == nil || !b.xp.allow("m:"+m.GuildID+":"+m.Author.ID, time.Now(), cfg.RateCount, cfg.RateSeconds) {
		return
	}
	gain := domain.RollXP(cfg.XPMin, cfg.XPMax, rand.IntN)
	b.grantXP(ctx, cfg, m.GuildID, m.Author.ID, display(m.Author, m.Member), m.ChannelID, roles, gain, true)
}

func (b *Bot) awardVoiceXP(parent context.Context) {
	if b == nil || b.voices == nil {
		return
	}
	for _, guildID := range b.voices.guilds() {
		if parent.Err() != nil {
			return
		}
		ctx, cancel := context.WithTimeout(parent, 20*time.Second)
		b.awardGuildVoice(ctx, guildID)
		cancel()
	}
}

func (b *Bot) awardGuildVoice(ctx context.Context, guildID string) {
	cfg, err := b.store.GetLevels(ctx, guildID)
	if err != nil {
		b.log.Warn("voice xp", "guild", guildID, "err", err)
		return
	}
	if !cfg.Enabled || !cfg.VoiceXP {
		return
	}
	afk := ""
	if b.session != nil && b.session.State != nil {
		if guild, err := b.session.State.Guild(guildID); err == nil && guild != nil {
			afk = guild.AfkChannelID
		}
	}
	for channelID, seats := range b.voiceChannels(guildID) {
		if channelID == "" || channelID == afk || len(seats) < 2 {
			continue
		}
		if domain.ContainsID(cfg.IgnoredChannelIDs, channelID) {
			continue
		}
		for _, person := range seats {
			if ctx.Err() != nil {
				return
			}
			roles := person.roles
			name := person.name
			if !person.known {
				member, err := b.session.GuildMember(guildID, person.userID, discordgo.WithContext(ctx))
				if err != nil {
					continue
				}
				roles = member.Roles
				name = display(member.User, member)
				b.voices.rememberRoles(guildID, person.userID, name, roles)
			}
			if overlaps(roles, cfg.IgnoredRoleIDs) {
				continue
			}
			if b.xp == nil || !b.xp.allow("v:"+guildID+":"+person.userID, time.Now(), 1, 60) {
				continue
			}
			gain := domain.RollXP(cfg.XPMin, cfg.XPMax, rand.IntN)
			b.grantXP(ctx, cfg, guildID, person.userID, name, "", roles, gain, true)
		}
	}
}

func (b *Bot) voiceChannels(guildID string) map[string][]voicePerson {
	return b.voices.channels(guildID)
}

func (b *Bot) channelBlocked(channelID string, ignored []string) bool {
	if domain.ContainsID(ignored, channelID) {
		return true
	}
	if b.session == nil || b.session.State == nil {
		return false
	}
	channel, err := b.session.State.Channel(channelID)
	if err != nil || channel == nil || channel.ParentID == "" {
		return false
	}
	return domain.ContainsID(ignored, channel.ParentID)
}

func (b *Bot) grantXP(ctx context.Context, cfg store.LevelSettings, guildID, userID, name, channelID string, roles []string, gain int, announce bool) {
	if gain < 1 || userID == "" {
		return
	}
	if b.session != nil && b.session.State != nil && b.session.State.User != nil && userID == b.session.State.User.ID {
		return
	}
	oldXP, newXP, err := b.store.GainMemberXP(ctx, guildID, userID, name, gain)
	if err != nil {
		b.log.Warn("grant xp", "guild", guildID, "user", userID, "err", err)
		return
	}
	b.finishLevelChange(ctx, cfg, guildID, userID, name, channelID, roles, oldXP, newXP, announce)
}

func (b *Bot) finishLevelChange(ctx context.Context, cfg store.LevelSettings, guildID, userID, name, channelID string, roles []string, oldXP, newXP int, announce bool) {
	oldLevel := domain.LevelForXP(oldXP)
	newLevel := domain.LevelForXP(newXP)
	if oldLevel != newLevel {
		b.applyLevelRoles(guildID, userID, newLevel, cfg.Rewards, cfg.ReplaceRewards, roles)
	}
	if newLevel <= oldLevel {
		return
	}
	if err := b.store.RecordLevelUp(ctx, guildID, userID, name, newLevel); err != nil {
		b.log.Warn("level log", "guild", guildID, "err", err)
	}
	if !announce || !domain.ShouldAnnounce(oldLevel, newLevel, cfg.AnnounceMinLevel, cfg.AnnounceMultiple, cfg.RewardOnly, rewardInputs(cfg.Rewards)) {
		return
	}
	b.sendLevelUp(cfg, guildID, channelID, userID, name, newLevel, newXP)
}

func rewardInputs(rewards []store.LevelReward) []domain.LevelReward {
	out := make([]domain.LevelReward, 0, len(rewards))
	for _, reward := range rewards {
		out = append(out, domain.LevelReward{Level: reward.Level, RoleID: reward.RoleID})
	}
	return out
}

func (b *Bot) sendLevelUp(cfg store.LevelSettings, guildID, channelID, userID, name string, level, xp int) {
	if name == "" {
		name = "member"
	}
	view := domain.LevelView{
		Mention: "<@" + userID + ">",
		User:    name,
		Server:  b.guildName(guildID),
		Level:   level,
		XP:      xp,
	}
	text, err := domain.RenderLevelMessage(cfg.Message, view)
	if err != nil {
		text, err = domain.RenderLevelMessage("", view)
	}
	if err != nil {
		b.log.Warn("level message", "guild", guildID, "err", err)
		return
	}
	switch cfg.Announce {
	case domain.AnnounceOff:
		return
	case domain.AnnounceDM:
		b.dm(userID, text)
	case domain.AnnounceChannel:
		b.sendLevelText(cfg.AnnounceChannelID, text)
	default:
		if channelID != "" {
			b.sendLevelText(channelID, text)
			return
		}
		b.sendLevelText(cfg.AnnounceChannelID, text)
	}
}

func (b *Bot) sendLevelText(channelID, text string) {
	if channelID == "" || text == "" || b.session == nil {
		return
	}
	_, err := b.session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content:         text,
		AllowedMentions: noMentions,
	})
	if err != nil {
		b.log.Warn("level message", "channel", channelID, "err", err)
	}
}

func (b *Bot) applyLevelRoles(guildID, userID string, level int, rewards []store.LevelReward, replace bool, have []string) {
	if b.session == nil || userID == "" || len(rewards) == 0 {
		return
	}
	inputs := rewardInputs(rewards)
	keep := domain.KeptRewardRoles(inputs, level, replace)
	keepSet := make(map[string]struct{}, len(keep))
	for _, roleID := range keep {
		keepSet[roleID] = struct{}{}
	}
	if have == nil {
		member, err := b.session.GuildMember(guildID, userID)
		if err != nil {
			b.log.Warn("level member", "guild", guildID, "user", userID, "err", err)
			return
		}
		have = member.Roles
	}
	for _, roleID := range domain.AllRewardRoles(inputs) {
		_, should := keepSet[roleID]
		has := domain.ContainsID(have, roleID)
		if should == has {
			continue
		}
		var err error
		if should {
			err = b.session.GuildMemberRoleAdd(guildID, userID, roleID)
		} else {
			err = b.session.GuildMemberRoleRemove(guildID, userID, roleID)
		}
		if err != nil && statusOf(err) != 404 {
			b.log.Warn("level role", "guild", guildID, "user", userID, "role", roleID, "err", err)
		}
	}
}

// SyncLevelMembers updates reward roles after a reset or a restore.
// A reset passes the scores from before the clear and useXP false, so every reward role is removed.
// A restore passes the restored scores and useXP true.
func (b *Bot) SyncLevelMembers(guildID string, members []store.LevelMember, rewards []store.LevelReward, replace, useXP bool) {
	if b == nil || b.session == nil || len(members) == 0 || len(rewards) == 0 {
		return
	}
	copied := append([]store.LevelMember(nil), members...)
	rewards = append([]store.LevelReward(nil), rewards...)
	go func() {
		for _, member := range copied {
			level := 0
			if useXP {
				level = domain.LevelForXP(member.XP)
			}
			b.applyLevelRoles(guildID, member.UserID, level, rewards, replace, nil)
			time.Sleep(200 * time.Millisecond)
		}
	}()
}
