package discord

import (
	"context"
	"encoding/base64"

	"github.com/bwmarrin/discordgo"
)

// GuildProfileBody is the Modify Current Member payload.
// A nil nick or avatar clears that field. Omitted fields stay as they are.
func GuildProfileBody(nick string, nickChanged bool, avatar []byte, avatarType string, avatarChanged bool) map[string]any {
	body := map[string]any{}
	if nickChanged {
		if nick == "" {
			body["nick"] = nil
		} else {
			body["nick"] = nick
		}
	}
	if avatarChanged {
		if len(avatar) == 0 {
			body["avatar"] = nil
		} else {
			body["avatar"] = "data:" + avatarType + ";base64," + base64.StdEncoding.EncodeToString(avatar)
		}
	}
	if len(body) == 0 {
		return nil
	}
	return body
}

// ApplyGuildProfile sets this bot's display name and avatar in one server.
func (b *Bot) ApplyGuildProfile(ctx context.Context, guildID, nick string, nickChanged bool, avatar []byte, avatarType string, avatarChanged bool) error {
	body := GuildProfileBody(nick, nickChanged, avatar, avatarType, avatarChanged)
	if body == nil {
		return nil
	}
	endpoint := discordgo.EndpointGuildMember(guildID, "@me")
	_, err := b.session.RequestWithBucketID(
		"PATCH", endpoint, body, discordgo.EndpointGuildMember(guildID, ""), discordgo.WithContext(ctx),
	)
	return err
}

// Account returns the bot username and its application avatar URL.
func (b *Bot) Account() (username, avatarURL string) {
	if b == nil || b.session == nil || b.session.State == nil {
		return "", ""
	}
	b.session.State.RLock()
	user := b.session.State.User
	b.session.State.RUnlock()
	if user == nil {
		return "", ""
	}
	return user.Username, user.AvatarURL("128")
}
