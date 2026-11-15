package harness

import "github.com/BoxingOctopus/squirrelbot/internal/discordmock"

func newDiscord(access, guest, refresh string) *discordmock.Server {
	mock := discordmock.New()
	mock.SetOAuth(access, refresh)
	mock.AddToken(access, discordmock.User{ID: UserID, Username: "acorn", GlobalName: UserName}, []discordmock.UserGuild{
		{ID: GuildID, Name: GuildName, Owner: true, Permissions: "32"},
		{ID: HollowGuildID, Name: "Hollow", Owner: true, Permissions: "32"},
	})
	mock.AddToken(guest, discordmock.User{ID: GuestUserID, Username: "guest", GlobalName: "Guest"}, []discordmock.UserGuild{
		{ID: GuildID, Name: GuildName, Owner: false, Permissions: "0"},
	})
	mock.AddGuild(GuildID, GuildName, OwnerID)
	mock.AddRole(GuildID, GuildID, "@everyone", 0, false)
	mock.AddRole(GuildID, MemberRoleID, "Members", 10, false)
	mock.AddRole(GuildID, ModRoleID, "Mods", 20, false)
	mock.AddRole(GuildID, AdminRoleID, "Admins", 30, false)
	mock.AddRole(GuildID, BotRoleID, "SquirrelBot", 50, true)
	mock.AddChannel(GuildID, GeneralChannelID, "general", 0)
	mock.AddChannel(GuildID, AnnounceChannelID, "announcements", 0)
	mock.AddEmoji(GuildID, EmojiID, "acorn", false)
	mock.AddMember(GuildID, discordmock.Member{
		ID: UserID, Username: "acorn", GlobalName: UserName, Roles: []string{AdminRoleID},
	})
	mock.AddMember(GuildID, discordmock.Member{
		ID: GuestUserID, Username: "guest", GlobalName: "Guest", Roles: []string{MemberRoleID},
	})
	mock.AddMember(GuildID, discordmock.Member{
		ID: TargetUserID, Username: TargetUsername, GlobalName: TargetName, Nick: TargetName, Roles: []string{MemberRoleID},
	})
	mock.AddMember(GuildID, discordmock.Member{
		ID: BotUserID, Username: "squirrelbot", GlobalName: "squirrelbot", Roles: []string{BotRoleID}, Bot: true,
	})
	return mock
}
