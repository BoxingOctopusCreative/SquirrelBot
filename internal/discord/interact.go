package discord

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/BoxingOctopus/squirrelbot/internal/domain"
	"github.com/BoxingOctopus/squirrelbot/internal/store"
	"github.com/bwmarrin/discordgo"
)

var noMentions = &discordgo.MessageAllowedMentions{Parse: []discordgo.AllowedMentionType{}}

type access struct {
	perm  int64
	mod   bool
	admin bool
}

func (b *Bot) onInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	defer func() {
		if recovered := recover(); recovered != nil {
			b.log.Error("interaction panic", "recover", recovered)
			_ = respondEphemeral(s, i, "Something went wrong while running that command.")
		}
	}()
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		b.routeCommand(s, i)
	case discordgo.InteractionMessageComponent:
		if i.MessageComponentData().CustomID == "sb:verify" {
			b.verifyMember(s, i)
		}
	case discordgo.InteractionModalSubmit:
		if i.ModalSubmitData().CustomID == "sb:verify" {
			b.submitVerification(s, i)
		}
	}
}

func (b *Bot) routeCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.ApplicationCommandData().Name {
	case "warn":
		b.moderate(s, i, "warn")
	case "kick":
		b.moderate(s, i, "kick")
	case "ban":
		b.moderate(s, i, "ban")
	case "timeout":
		b.moderate(s, i, "timeout")
	case "untimeout":
		b.moderate(s, i, "untimeout")
	case "cases":
		b.cases(s, i)
	case "purge":
		b.purge(s, i)
	case "slowmode":
		b.slowmode(s, i)
	case "announce":
		b.announce(s, i)
	case "escalate":
		b.escalate(s, i)
	case "deescalate":
		b.deescalate(s, i)
	case "escalations":
		b.listEscalations(s, i)
	case "gate":
		b.gateCommand(s, i)
	default:
		_ = respondEphemeral(s, i, "I don't know that command.")
	}
}

func (b *Bot) permit(i *discordgo.InteractionCreate, need access) (store.Guild, bool) {
	if i.GuildID == "" || i.Member == nil || i.Member.User == nil {
		_ = respondEphemeral(b.session, i, "Use this command in a server.")
		return store.Guild{}, false
	}
	ctx, cancel := contextTimeout()
	defer cancel()
	guild, err := b.touchGuild(ctx, i.GuildID)
	if err != nil {
		b.log.Error("load guild", "guild", i.GuildID, "err", err)
		_ = respondEphemeral(b.session, i, "I could not load this server.")
		return store.Guild{}, false
	}
	bits := i.Member.Permissions
	if domain.HasPermission(bits, domain.PermAdministrator) {
		return guild, true
	}
	if need.perm != 0 && bits&need.perm == need.perm {
		return guild, true
	}
	if need.admin && guild.AdminRoleID != "" && domain.ContainsID(i.Member.Roles, guild.AdminRoleID) {
		return guild, true
	}
	if need.mod && guild.ModRoleID != "" && domain.ContainsID(i.Member.Roles, guild.ModRoleID) {
		return guild, true
	}
	_ = respondEphemeral(b.session, i, "You don't have permission to do that.")
	return guild, false
}

func respond(s *discordgo.Session, i *discordgo.InteractionCreate, content string, ephemeral bool) error {
	data := &discordgo.InteractionResponseData{Content: content, AllowedMentions: noMentions}
	if ephemeral {
		data.Flags = discordgo.MessageFlagsEphemeral
	}
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: data,
	})
}

func respondEphemeral(s *discordgo.Session, i *discordgo.InteractionCreate, content string) error {
	return respond(s, i, content, true)
}

func deferReply(s *discordgo.Session, i *discordgo.InteractionCreate, ephemeral bool) error {
	data := &discordgo.InteractionResponseData{AllowedMentions: noMentions}
	if ephemeral {
		data.Flags = discordgo.MessageFlagsEphemeral
	}
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: data,
	})
}

func editReply(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content:         &content,
		AllowedMentions: noMentions,
	})
	if err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: ptr("I finished the action, but could not edit the reply.")})
	}
}

func ptr(s string) *string { return &s }

func contextTimeout() (ctx context.Context, cancel func()) {
	return context.WithTimeout(context.Background(), 20*time.Second)
}

func option(i *discordgo.InteractionCreate, name string) *discordgo.ApplicationCommandInteractionDataOption {
	for _, opt := range i.ApplicationCommandData().Options {
		if opt.Name == name {
			return opt
		}
	}
	return nil
}

func optionString(i *discordgo.InteractionCreate, name string) string {
	opt := option(i, name)
	if opt == nil || opt.Type != discordgo.ApplicationCommandOptionString {
		return ""
	}
	return strings.TrimSpace(opt.StringValue())
}

func optionInt(i *discordgo.InteractionCreate, name string, fallback int) int {
	opt := option(i, name)
	if opt == nil || opt.Type != discordgo.ApplicationCommandOptionInteger {
		return fallback
	}
	return int(opt.IntValue())
}

func optionUser(i *discordgo.InteractionCreate, name string) *discordgo.User {
	opt := option(i, name)
	if opt == nil {
		return nil
	}
	id, _ := opt.Value.(string)
	if id == "" {
		return nil
	}
	data := i.ApplicationCommandData()
	if data.Resolved != nil {
		if user := data.Resolved.Users[id]; user != nil {
			return user
		}
	}
	return &discordgo.User{ID: id}
}

func optionChannel(i *discordgo.InteractionCreate, name string) *discordgo.Channel {
	opt := option(i, name)
	if opt == nil {
		return nil
	}
	id, _ := opt.Value.(string)
	if id == "" {
		return nil
	}
	data := i.ApplicationCommandData()
	if data.Resolved != nil {
		if channel := data.Resolved.Channels[id]; channel != nil {
			return channel
		}
	}
	return &discordgo.Channel{ID: id}
}

func optionRole(i *discordgo.InteractionCreate, name string) *discordgo.Role {
	opt := option(i, name)
	if opt == nil {
		return nil
	}
	id, _ := opt.Value.(string)
	if id == "" {
		return nil
	}
	data := i.ApplicationCommandData()
	if data.Resolved != nil {
		if role := data.Resolved.Roles[id]; role != nil {
			return role
		}
	}
	return &discordgo.Role{ID: id}
}

func display(user *discordgo.User, member *discordgo.Member) string {
	if member != nil && member.Nick != "" {
		return member.Nick
	}
	if user == nil {
		return "unknown"
	}
	if user.GlobalName != "" {
		return user.GlobalName
	}
	if user.Username != "" {
		return user.Username
	}
	return user.ID
}

func actor(i *discordgo.InteractionCreate) (string, string) {
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User.ID, display(i.Member.User, i.Member)
	}
	if i.User != nil {
		return i.User.ID, display(i.User, nil)
	}
	return "", "unknown"
}

func statusOf(err error) int {
	var rest *discordgo.RESTError
	if errors.As(err, &rest) && rest.Response != nil {
		return rest.Response.StatusCode
	}
	return 0
}

func discordFailure(err error) string {
	switch statusOf(err) {
	case 403:
		return "Discord refused that. Check my role position and permissions."
	case 404:
		return "Discord could not find that member."
	default:
		return "Discord refused that action."
	}
}

func (b *Bot) dm(userID, content string) {
	if userID == "" || content == "" {
		return
	}
	channel, err := b.session.UserChannelCreate(userID)
	if err != nil {
		return
	}
	_, _ = b.session.ChannelMessageSendComplex(channel.ID, &discordgo.MessageSend{
		Content:         content,
		AllowedMentions: noMentions,
	})
}

func (b *Bot) sendLog(channelID string, embed *discordgo.MessageEmbed) {
	if channelID == "" || embed == nil {
		return
	}
	if embed.Timestamp == "" {
		embed.Timestamp = time.Now().Format(time.RFC3339)
	}
	if embed.Footer == nil {
		embed.Footer = &discordgo.MessageEmbedFooter{Text: "SquirrelBot"}
	}
	if _, err := b.session.ChannelMessageSendEmbed(channelID, embed); err != nil {
		b.log.Warn("mod log", "channel", channelID, "err", err)
	}
}

func (b *Bot) outranks(guildID, actorID, targetID string, actorRoles, targetRoles []string) error {
	self := ""
	if b.session.State.User != nil {
		self = b.session.State.User.ID
	}
	if targetID == self {
		return errors.New("I can't do that to myself.")
	}
	if actorID != "" && actorID == targetID {
		return errors.New("You can't do that to yourself.")
	}
	guild, err := b.session.Guild(guildID)
	if err != nil {
		return errors.New("I could not read this server's roles.")
	}
	if targetID == guild.OwnerID {
		return errors.New("You can't do that to the server owner.")
	}
	positions := map[string]int{}
	for _, role := range guild.Roles {
		positions[role.ID] = role.Position
	}
	botMember, err := b.session.GuildMember(guildID, self)
	if err != nil {
		return errors.New("I could not read my own roles.")
	}
	if domain.TopRolePosition(positions, botMember.Roles) <= domain.TopRolePosition(positions, targetRoles) {
		return errors.New("Move my role above theirs, then try again.")
	}
	if actorID != "" && actorID != guild.OwnerID {
		if domain.TopRolePosition(positions, actorRoles) <= domain.TopRolePosition(positions, targetRoles) {
			return errors.New("Their role is above yours.")
		}
	}
	return nil
}

func (b *Bot) canAssign(guildID, roleID string) error {
	if roleID == guildID {
		return errors.New("That role cannot be granted.")
	}
	roles, err := b.session.GuildRoles(guildID)
	if err != nil {
		return errors.New("I could not read this server's roles.")
	}
	var target *discordgo.Role
	positions := map[string]int{}
	for _, role := range roles {
		positions[role.ID] = role.Position
		if role.ID == roleID {
			target = role
		}
	}
	if target == nil {
		return errors.New("That role is not on this server.")
	}
	if target.Managed {
		return errors.New("That role is managed by an integration.")
	}
	self := ""
	if b.session.State.User != nil {
		self = b.session.State.User.ID
	}
	botMember, err := b.session.GuildMember(guildID, self)
	if err != nil {
		return errors.New("I could not read my own roles.")
	}
	if domain.TopRolePosition(positions, botMember.Roles) <= target.Position {
		return errors.New("Move my role above that role, then try again.")
	}
	return nil
}

func memberHas(member *discordgo.Member, roleID string) bool {
	if member == nil || roleID == "" {
		return false
	}
	return domain.ContainsID(member.Roles, roleID)
}

func caseLine(number int, action, target, reason string) string {
	if reason == "" {
		reason = "No reason given."
	}
	return fmt.Sprintf("Case #%d · %s · %s\n%s", number, action, target, reason)
}
