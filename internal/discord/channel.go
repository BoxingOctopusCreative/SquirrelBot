package discord

import (
	"errors"
	"strings"

	"github.com/BoxingOctopus/squirrelbot/internal/domain"
	"github.com/bwmarrin/discordgo"
)

func (b *Bot) EnsureTextChannel(guildID string, spec domain.SetupChannel, staffRoleIDs []string, userID string) (id, name string, created bool, err error) {
	channels, err := b.session.GuildChannels(guildID)
	if err != nil {
		return "", "", false, errors.New("I could not read this server's channels.")
	}
	for _, channel := range channels {
		if channel.Type != discordgo.ChannelTypeGuildText && channel.Type != discordgo.ChannelTypeGuildNews {
			continue
		}
		if strings.EqualFold(channel.Name, spec.Name) {
			return channel.ID, channel.Name, false, nil
		}
	}
	data := discordgo.GuildChannelCreateData{
		Name:  spec.Name,
		Type:  discordgo.ChannelTypeGuildText,
		Topic: spec.Topic,
	}
	if spec.Private {
		overwrites, err := b.privateOverwrites(guildID, staffRoleIDs, userID)
		if err != nil {
			return "", "", false, err
		}
		data.PermissionOverwrites = overwrites
	}
	channel, err := b.session.GuildChannelCreateComplex(guildID, data)
	if err != nil {
		if statusOf(err) == 403 {
			return "", "", false, errors.New("I need Manage Channels to create that channel.")
		}
		return "", "", false, errors.New("Discord refused that channel.")
	}
	return channel.ID, channel.Name, true, nil
}

func (b *Bot) privateOverwrites(guildID string, staffRoleIDs []string, userID string) ([]*discordgo.PermissionOverwrite, error) {
	botRoleID, err := b.ownRoleID(guildID)
	if err != nil {
		return nil, err
	}
	view := int64(discordgo.PermissionViewChannel)
	read := int64(discordgo.PermissionReadMessageHistory)
	botAllow := view | int64(discordgo.PermissionSendMessages) | int64(discordgo.PermissionEmbedLinks) | read
	overwrites := []*discordgo.PermissionOverwrite{{
		ID:   guildID,
		Type: discordgo.PermissionOverwriteTypeRole,
		Deny: view,
	}}
	seen := map[string]bool{guildID: true}
	add := func(id string, kind discordgo.PermissionOverwriteType, allow int64) {
		if id == "" || seen[id] || !domain.ValidSnowflake(id) {
			return
		}
		seen[id] = true
		overwrites = append(overwrites, &discordgo.PermissionOverwrite{
			ID: id, Type: kind, Allow: allow,
		})
	}
	add(botRoleID, discordgo.PermissionOverwriteTypeRole, botAllow)
	for _, id := range staffRoleIDs {
		add(id, discordgo.PermissionOverwriteTypeRole, view|read)
	}
	add(userID, discordgo.PermissionOverwriteTypeMember, view|read)
	return overwrites, nil
}

func (b *Bot) ownRoleID(guildID string) (string, error) {
	if b.session.State.User == nil {
		return "", errors.New("I am not connected.")
	}
	member, err := b.session.GuildMember(guildID, b.session.State.User.ID)
	if err != nil {
		return "", errors.New("I could not read my own roles.")
	}
	roles, err := b.session.GuildRoles(guildID)
	if err != nil {
		return "", errors.New("I could not read this server's roles.")
	}
	positions := map[string]int{}
	for _, role := range roles {
		positions[role.ID] = role.Position
	}
	best := ""
	bestPos := -1
	for _, id := range member.Roles {
		if positions[id] > bestPos {
			best = id
			bestPos = positions[id]
		}
	}
	if best == "" {
		return "", errors.New("Give me a role before creating a private log channel.")
	}
	return best, nil
}
