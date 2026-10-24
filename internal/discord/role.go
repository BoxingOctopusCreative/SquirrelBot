package discord

import (
	"errors"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) EnsureRole(guildID, name string) (id, roleName string, position int, created bool, err error) {
	roles, err := b.session.GuildRoles(guildID)
	if err != nil {
		return "", "", 0, false, errors.New("I could not read this server's roles.")
	}
	for _, role := range roles {
		if role.Managed || role.ID == guildID {
			continue
		}
		if strings.EqualFold(role.Name, name) {
			return role.ID, role.Name, role.Position, false, nil
		}
	}
	perms := int64(0)
	mention := false
	hoist := false
	role, err := b.session.GuildRoleCreate(guildID, &discordgo.RoleParams{
		Name:        name,
		Permissions: &perms,
		Mentionable: &mention,
		Hoist:       &hoist,
	})
	if err != nil {
		if statusOf(err) == 403 {
			return "", "", 0, false, errors.New("I need Manage Roles to create that role.")
		}
		return "", "", 0, false, errors.New("Discord refused that role.")
	}
	return role.ID, role.Name, role.Position, true, nil
}
