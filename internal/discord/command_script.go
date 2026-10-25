package discord

import (
	"errors"
	"strings"

	"github.com/BoxingOctopus/squirrelbot/internal/domain"
	"github.com/BoxingOctopus/squirrelbot/internal/script"
	"github.com/bwmarrin/discordgo"
)

func splitCommand(content, prefix string) (name string, args []string, argsText string, ok bool) {
	body := strings.TrimSpace(strings.TrimPrefix(content, prefix))
	fields := strings.Fields(body)
	if len(fields) == 0 {
		return "", nil, "", false
	}
	name = strings.ToLower(fields[0])
	if len(fields) == 1 {
		return name, []string{}, "", true
	}
	argsText = strings.TrimSpace(body[len(fields[0]):])
	return name, fields[1:], argsText, true
}

func commandAuthor(m *discordgo.MessageCreate) (name, display string, roles []string) {
	name = m.Author.Username
	display = name
	if m.Author.GlobalName != "" {
		display = m.Author.GlobalName
	}
	if m.Member != nil {
		if m.Member.Nick != "" {
			display = m.Member.Nick
		}
		roles = append([]string{}, m.Member.Roles...)
	}
	if roles == nil {
		roles = []string{}
	}
	return name, display, roles
}

func (b *Bot) commandRoles(guildID string, ids []string) ([]script.Role, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	raw, err := b.session.GuildRoles(guildID)
	if err != nil {
		return nil, err
	}
	want := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		want[id] = struct{}{}
	}
	out := make([]script.Role, 0, len(ids))
	for _, role := range raw {
		if role == nil || role.Managed || role.ID == guildID {
			continue
		}
		if _, ok := want[role.ID]; !ok {
			continue
		}
		out = append(out, script.Role{ID: role.ID, Name: role.Name})
	}
	return out, nil
}

func (b *Bot) applyCommandEffect(s *discordgo.Session, m *discordgo.MessageCreate, effect script.Effect) error {
	switch effect.Op {
	case script.OpReply:
		return sendPlain(s, m.ChannelID, effect.Value)
	case script.OpReact:
		if err := s.MessageReactionAdd(m.ChannelID, m.ID, effect.Value); err != nil {
			return err
		}
		return nil
	case script.OpDelete:
		if err := s.ChannelMessageDelete(m.ChannelID, m.ID); err != nil {
			return err
		}
		return nil
	case script.OpAddRole:
		if err := b.canAssign(m.GuildID, effect.Value); err != nil {
			return err
		}
		return s.GuildMemberRoleAdd(m.GuildID, m.Author.ID, effect.Value, discordgo.WithAuditLogReason(domain.AuditReason("custom command")))
	case script.OpRemoveRole:
		if err := b.canAssign(m.GuildID, effect.Value); err != nil {
			return err
		}
		return s.GuildMemberRoleRemove(m.GuildID, m.Author.ID, effect.Value, discordgo.WithAuditLogReason(domain.AuditReason("custom command")))
	default:
		return errors.New("I could not run that command.")
	}
}

func sendPlain(s *discordgo.Session, channelID, content string) error {
	_, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content:         content,
		AllowedMentions: noMentions,
	})
	return err
}

func publicEffectError(err error) string {
	message := script.ErrorText(err)
	if message == "" || strings.Contains(message, "HTTP") || strings.Contains(message, "{") {
		return "Discord refused that action."
	}
	return message
}
