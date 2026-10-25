package discord

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/BoxingOctopus/squirrelbot/internal/domain"
	"github.com/BoxingOctopus/squirrelbot/internal/store"
	"github.com/bwmarrin/discordgo"
)

const (
	colorCopper  = 0xD08A4A
	colorFoxfire = 0x7DCEA0
	colorBerry   = 0xE15D52
	colorMist    = 0x8E99AE
)

type Bot struct {
	session *discordgo.Session
	store   *store.Store
	log     *slog.Logger
	ready   atomic.Bool
	spam    *spamTracker
	muted   *recentIDs
	once    sync.Once
}

func New(token string, st *store.Store, log *slog.Logger) (*Bot, error) {
	if !strings.HasPrefix(token, "Bot ") {
		token = "Bot " + token
	}
	session, err := discordgo.New(token)
	if err != nil {
		return nil, err
	}
	session.Identify.Intents = discordgo.IntentGuilds |
		discordgo.IntentGuildMembers |
		discordgo.IntentGuildModeration |
		discordgo.IntentGuildMessages |
		discordgo.IntentGuildMessageReactions |
		discordgo.IntentMessageContent
	session.State.MaxMessageCount = 80
	session.State.TrackMembers = false
	session.State.TrackPresences = false
	session.State.TrackVoice = false
	session.State.TrackEmojis = false
	session.State.TrackStickers = false
	session.State.TrackThreadMembers = false

	bot := &Bot{
		session: session,
		store:   st,
		log:     log,
		spam:    newSpamTracker(),
		muted:   newRecentIDs(),
	}
	session.AddHandler(bot.onReady)
	session.AddHandler(bot.onGuildCreate)
	session.AddHandler(bot.onInteraction)
	session.AddHandler(bot.onMessage)
	session.AddHandler(bot.onJoin)
	session.AddHandler(bot.onLeave)
	session.AddHandler(bot.onMemberUpdate)
	session.AddHandler(bot.onDelete)
	session.AddHandler(bot.onEdit)
	session.AddHandler(bot.onBulkDelete)
	session.AddHandler(bot.onBan)
	session.AddHandler(bot.onReactionAdd)
	session.AddHandler(bot.onReactionRemove)
	session.AddHandler(bot.onReactionRemoveAll)
	return bot, nil
}

func (b *Bot) Open() error { return b.session.Open() }

func (b *Bot) Close() error { return b.session.Close() }

func (b *Bot) Ready() bool { return b.ready.Load() }

// SetReady records whether the gateway is connected.
// The ready event sets this during a normal run. Tests set it when Discord is stubbed.
func (b *Bot) SetReady(ready bool) { b.ready.Store(ready) }

func (b *Bot) Session() *discordgo.Session { return b.session }

func (b *Bot) registerCommands(s *discordgo.Session, appID string) error {
	body, err := commandPayload()
	if err != nil {
		return err
	}
	endpoint := discordgo.EndpointApplicationGlobalCommands(appID)
	_, err = s.RequestWithBucketID("PUT", endpoint, json.RawMessage(body), endpoint)
	return err
}

func (b *Bot) onReady(s *discordgo.Session, r *discordgo.Ready) {
	b.ready.Store(true)
	if err := s.UpdateWatchStatus(0, "the canopy"); err != nil {
		b.log.Warn("set status", "err", err)
	}
	b.once.Do(func() {
		if err := b.registerCommands(s, r.User.ID); err != nil {
			b.log.Error("register commands", "err", err)
		} else {
			b.log.Info("registered slash commands")
		}
	})
}

func (b *Bot) onGuildCreate(s *discordgo.Session, g *discordgo.GuildCreate) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := b.store.EnsureGuild(ctx, g.ID, g.Name, g.Icon); err != nil {
		b.log.Error("ensure guild", "guild", g.ID, "err", err)
	}
}

func (b *Bot) RememberGuild(ctx context.Context, guildID string) error {
	if b == nil || !domain.ValidSnowflake(guildID) {
		return nil
	}
	guild, err := b.session.Guild(guildID, discordgo.WithContext(ctx))
	if err != nil {
		return err
	}
	return b.store.EnsureGuild(ctx, guild.ID, guild.Name, guild.Icon)
}

func (b *Bot) touchGuild(ctx context.Context, guildID string) (store.Guild, error) {
	name, icon := "", ""
	if g, err := b.session.State.Guild(guildID); err == nil && g != nil {
		name, icon = g.Name, g.Icon
	}
	if err := b.store.EnsureGuild(ctx, guildID, name, icon); err != nil {
		return store.Guild{}, err
	}
	return b.store.GetGuild(ctx, guildID)
}

func (b *Bot) guildName(guildID string) string {
	if g, err := b.session.State.Guild(guildID); err == nil && g != nil && g.Name != "" {
		return g.Name
	}
	return "this server"
}
