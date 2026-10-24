package discord

import (
	"context"
	"errors"

	"github.com/BoxingOctopus/squirrelbot/internal/domain"
	"github.com/bwmarrin/discordgo"
)

func (b *Bot) onReactionAdd(s *discordgo.Session, event *discordgo.MessageReactionAdd) {
	b.applyReaction(event.GuildID, event.ChannelID, event.UserID, event.MessageID, event.Emoji, true, event.Member)
}

func (b *Bot) onReactionRemove(s *discordgo.Session, event *discordgo.MessageReactionRemove) {
	b.applyReaction(event.GuildID, event.ChannelID, event.UserID, event.MessageID, event.Emoji, false, nil)
}

func (b *Bot) applyReaction(guildID, channelID, userID, messageID string, emoji discordgo.Emoji, added bool, member *discordgo.Member) {
	if guildID == "" || userID == "" || channelID == "" {
		return
	}
	if b.session.State.User != nil && userID == b.session.State.User.ID {
		return
	}
	if member != nil && member.User != nil && member.User.Bot {
		return
	}
	ctx, cancel := contextTimeout()
	defer cancel()
	b.grantReactionRole(ctx, guildID, userID, messageID, emoji, added)
	b.refreshStar(guildID, channelID, messageID, &emoji)
}

func (b *Bot) grantReactionRole(ctx context.Context, guildID, userID, messageID string, emoji discordgo.Emoji, added bool) bool {
	rows, err := b.store.ReactionRolesForMessage(ctx, messageID)
	if err != nil || len(rows) == 0 {
		return false
	}
	bindings := make([]domain.ReactionBinding, len(rows))
	for i, row := range rows {
		bindings[i] = domain.ReactionBinding{Emoji: row.Emoji, RoleID: row.RoleID}
	}
	roleID, grant, ok := domain.ApplyReaction(bindings, emojiKey(emoji), added)
	if !ok {
		return false
	}
	if grant {
		if err := b.session.GuildMemberRoleAdd(guildID, userID, roleID); err != nil {
			b.log.Warn("reaction role add", "guild", guildID, "err", err)
		}
		return true
	}
	if err := b.session.GuildMemberRoleRemove(guildID, userID, roleID); err != nil && statusOf(err) != 404 {
		b.log.Warn("reaction role remove", "guild", guildID, "err", err)
	}
	return true
}

func (b *Bot) PublishReactionPanel(channelID, messageID string, lines []domain.ReactionLine) (string, error) {
	content := domain.ReactionPanelText(lines)
	if messageID != "" {
		_, err := b.session.ChannelMessageEdit(channelID, messageID, content)
		if err == nil {
			return messageID, nil
		}
		if statusOf(err) != 404 {
			return "", errors.New("I could not update the roles message.")
		}
	}
	sent, err := b.session.ChannelMessageSend(channelID, content)
	if err != nil {
		switch statusOf(err) {
		case 403:
			return "", errors.New("I need permission to send messages in that channel.")
		case 404:
			return "", errors.New("I cannot see that channel.")
		default:
			return "", errors.New("Discord refused the roles message.")
		}
	}
	return sent.ID, nil
}

func (b *Bot) RemoveReactionPanel(channelID, messageID string) {
	if b.session == nil || channelID == "" || messageID == "" {
		return
	}
	_ = b.session.ChannelMessageDelete(channelID, messageID)
}

func (b *Bot) RoleNames(guildID string) (map[string]string, error) {
	roles, err := b.session.GuildRoles(guildID)
	if err != nil {
		return nil, err
	}
	names := make(map[string]string, len(roles))
	for _, role := range roles {
		names[role.ID] = role.Name
	}
	return names, nil
}

func (b *Bot) PlaceReaction(guildID, channelID, messageID, emoji, roleID string) error {
	if err := b.canAssign(guildID, roleID); err != nil {
		return err
	}
	if err := b.session.MessageReactionAdd(channelID, messageID, emoji); err != nil {
		switch statusOf(err) {
		case 403:
			return errors.New("I need Add Reactions in that channel, and my role must sit above the role you picked.")
		case 404:
			return errors.New("That message is gone, or I cannot see the channel.")
		default:
			return errors.New("Discord refused that emoji. For a custom emoji, use name:id from this server.")
		}
	}
	return nil
}

func (b *Bot) LiftReaction(channelID, messageID, emoji string) {
	if b.session == nil {
		return
	}
	_ = b.session.MessageReactionRemove(channelID, messageID, emoji, "@me")
}

func emojiKey(emoji discordgo.Emoji) string {
	if emoji.ID != "" {
		return emoji.Name + ":" + emoji.ID
	}
	return emoji.Name
}
