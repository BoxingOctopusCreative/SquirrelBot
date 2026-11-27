package discord

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/BoxingOctopus/squirrelbot/internal/domain"
	"github.com/BoxingOctopus/squirrelbot/internal/store"
	"github.com/bwmarrin/discordgo"
)

func (b *Bot) rank(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.GuildID == "" || i.Member == nil || i.Member.User == nil {
		_ = respondEphemeral(s, i, "Use this command in a server.")
		return
	}
	ctx, cancel := contextTimeout()
	defer cancel()
	cfg, err := b.store.GetLevels(ctx, i.GuildID)
	if err != nil {
		b.log.Warn("rank", "guild", i.GuildID, "err", err)
		_ = respondEphemeral(s, i, "I could not load levels.")
		return
	}
	if !cfg.Enabled {
		_ = respondEphemeral(s, i, "Levels are off in this server.")
		return
	}
	user := optionUser(i, "user")
	member := i.Member
	if user == nil || user.ID == "" || user.ID == i.Member.User.ID {
		user = i.Member.User
	} else {
		member = resolvedMember(i, user.ID)
	}
	if user.Bot {
		_ = respondEphemeral(s, i, "Bots do not earn XP.")
		return
	}
	row, found, err := b.store.MemberLevel(ctx, i.GuildID, user.ID)
	if err != nil {
		b.log.Warn("rank", "guild", i.GuildID, "err", err)
		_ = respondEphemeral(s, i, "I could not load that rank.")
		return
	}
	if err := respondEmbed(s, i, rankEmbed(display(user, member), row, found), false); err != nil {
		b.log.Warn("rank reply", "guild", i.GuildID, "err", err)
	}
}

func (b *Bot) leaderboard(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.GuildID == "" || i.Member == nil || i.Member.User == nil {
		_ = respondEphemeral(s, i, "Use this command in a server.")
		return
	}
	ctx, cancel := contextTimeout()
	defer cancel()
	cfg, err := b.store.GetLevels(ctx, i.GuildID)
	if err != nil {
		b.log.Warn("leaderboard", "guild", i.GuildID, "err", err)
		_ = respondEphemeral(s, i, "I could not load levels.")
		return
	}
	if !cfg.Enabled {
		_ = respondEphemeral(s, i, "Levels are off in this server.")
		return
	}
	page := optionInt(i, "page", 1)
	if page < 1 {
		page = 1
	}
	if page > domain.MaxLevelPage {
		page = domain.MaxLevelPage
	}
	rows, total, err := b.store.Leaderboard(ctx, i.GuildID, page)
	if err != nil {
		b.log.Warn("leaderboard", "guild", i.GuildID, "err", err)
		_ = respondEphemeral(s, i, "I could not load the leaderboard.")
		return
	}
	if err := respondEmbed(s, i, leaderboardEmbed(page, total, rows), false); err != nil {
		b.log.Warn("leaderboard reply", "guild", i.GuildID, "err", err)
	}
}

func (b *Bot) levelCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if _, ok := b.permit(i, access{perm: domain.PermManageGuild, admin: true}); !ok {
		return
	}
	sub, opts := commandSub(i)
	if sub == nil {
		_ = respondEphemeral(s, i, "Choose an action.")
		return
	}
	switch sub.Name {
	case "add":
		b.levelAdd(s, i, opts)
	case "set":
		b.levelSet(s, i, opts)
	case "reset":
		b.levelReset(s, i, opts)
	case "reset-all":
		b.levelResetAll(s, i)
	case "restore":
		b.levelRestore(s, i)
	case "log":
		b.levelLog(s, i)
	default:
		_ = respondEphemeral(s, i, "I don't know that action.")
	}
}

func (b *Bot) levelAdd(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user, name, roles, ok := b.levelTarget(i, opts)
	if !ok {
		return
	}
	amount := subInt(opts, "amount", 0)
	if amount < 1 || amount > 1000 {
		_ = respondEphemeral(s, i, "Add from 1 to 1000 XP.")
		return
	}
	ctx, cancel := contextTimeout()
	defer cancel()
	cfg, err := b.enabledLevels(ctx, i)
	if err != nil || cfg.GuildID == "" {
		return
	}
	oldXP, newXP, err := b.store.GainMemberXP(ctx, i.GuildID, user.ID, name, amount)
	if err != nil {
		b.log.Warn("add xp", "guild", i.GuildID, "err", err)
		_ = respondEphemeral(s, i, "I could not add that XP.")
		return
	}
	if newXP == oldXP {
		_ = respondEphemeral(s, i, "They are already at the highest XP.")
		return
	}
	b.finishLevelChange(ctx, cfg, i.GuildID, user.ID, name, "", roles, oldXP, newXP, true)
	level, _, _ := domain.Progress(newXP)
	_ = respondEphemeral(s, i, fmt.Sprintf("Added %d XP. %s is level %d with %s XP.", newXP-oldXP, name, level, grouped(newXP)))
}

func (b *Bot) levelSet(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user, name, roles, ok := b.levelTarget(i, opts)
	if !ok {
		return
	}
	level := subInt(opts, "level", -1)
	if level < 0 || level > domain.MaxLevel {
		_ = respondEphemeral(s, i, fmt.Sprintf("Set a level from 0 to %d.", domain.MaxLevel))
		return
	}
	ctx, cancel := contextTimeout()
	defer cancel()
	cfg, err := b.enabledLevels(ctx, i)
	if err != nil || cfg.GuildID == "" {
		return
	}
	oldXP, newXP, err := b.store.SetMemberXP(ctx, i.GuildID, user.ID, name, domain.TotalXP(level))
	if err != nil {
		b.log.Warn("set level", "guild", i.GuildID, "err", err)
		_ = respondEphemeral(s, i, "I could not set that level.")
		return
	}
	if newXP == oldXP {
		_ = respondEphemeral(s, i, fmt.Sprintf("They are already level %d.", domain.LevelForXP(oldXP)))
		return
	}
	b.finishLevelChange(ctx, cfg, i.GuildID, user.ID, name, "", roles, oldXP, newXP, true)
	_ = respondEphemeral(s, i, fmt.Sprintf("%s is level %d with %s XP.", name, domain.LevelForXP(newXP), grouped(newXP)))
}

func (b *Bot) levelReset(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	user, name, roles, ok := b.levelTarget(i, opts)
	if !ok {
		return
	}
	ctx, cancel := contextTimeout()
	defer cancel()
	cfg, err := b.store.GetLevels(ctx, i.GuildID)
	if err != nil {
		b.log.Warn("reset level", "guild", i.GuildID, "err", err)
		_ = respondEphemeral(s, i, "I could not load levels.")
		return
	}
	oldXP, err := b.store.ClearMemberXP(ctx, i.GuildID, user.ID)
	if err != nil {
		b.log.Warn("reset level", "guild", i.GuildID, "err", err)
		_ = respondEphemeral(s, i, "I could not reset that member.")
		return
	}
	if oldXP == 0 {
		_ = respondEphemeral(s, i, name+" had no XP.")
		return
	}
	b.finishLevelChange(ctx, cfg, i.GuildID, user.ID, name, "", roles, oldXP, 0, false)
	_ = respondEphemeral(s, i, name+" is back to level 0. This reset was not backed up.")
}

func (b *Bot) levelResetAll(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx, cancel := contextTimeout()
	defer cancel()
	members, err := b.store.BackupAndClearLevels(ctx, i.GuildID)
	if err != nil {
		b.log.Warn("reset levels", "guild", i.GuildID, "err", err)
		_ = respondEphemeral(s, i, "I could not reset levels.")
		return
	}
	cfg, err := b.store.GetLevels(ctx, i.GuildID)
	if err != nil {
		b.log.Warn("reset levels", "guild", i.GuildID, "err", err)
	} else {
		b.SyncLevelMembers(i.GuildID, members, cfg.Rewards, cfg.ReplaceRewards, false)
	}
	_ = respondEphemeral(s, i, fmt.Sprintf("Cleared XP for %d members. Restore brings this backup back. A new reset replaces it.", len(members)))
}

func (b *Bot) levelRestore(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx, cancel := contextTimeout()
	defer cancel()
	members, err := b.store.RestoreLevels(ctx, i.GuildID)
	if errors.Is(err, store.ErrNoLevelBackup) {
		_ = respondEphemeral(s, i, err.Error())
		return
	}
	if err != nil {
		b.log.Warn("restore levels", "guild", i.GuildID, "err", err)
		_ = respondEphemeral(s, i, "I could not restore levels.")
		return
	}
	cfg, err := b.store.GetLevels(ctx, i.GuildID)
	if err != nil {
		b.log.Warn("restore levels", "guild", i.GuildID, "err", err)
	} else {
		b.SyncLevelMembers(i.GuildID, members, cfg.Rewards, cfg.ReplaceRewards, true)
	}
	_ = respondEphemeral(s, i, fmt.Sprintf("Restored XP for %d members.", len(members)))
}

func (b *Bot) levelLog(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx, cancel := contextTimeout()
	defer cancel()
	events, err := b.store.ListLevelEvents(ctx, i.GuildID)
	if err != nil {
		b.log.Warn("level log", "guild", i.GuildID, "err", err)
		_ = respondEphemeral(s, i, "I could not load the level log.")
		return
	}
	if err := respondEmbed(s, i, levelLogEmbed(events), true); err != nil {
		b.log.Warn("level log reply", "guild", i.GuildID, "err", err)
	}
}

func (b *Bot) enabledLevels(ctx context.Context, i *discordgo.InteractionCreate) (store.LevelSettings, error) {
	cfg, err := b.store.GetLevels(ctx, i.GuildID)
	if err != nil {
		b.log.Warn("levels", "guild", i.GuildID, "err", err)
		_ = respondEphemeral(b.session, i, "I could not load levels.")
		return store.LevelSettings{}, err
	}
	if !cfg.Enabled {
		_ = respondEphemeral(b.session, i, "Turn levels on in the dashboard first.")
		return store.LevelSettings{}, nil
	}
	return cfg, nil
}

func (b *Bot) levelTarget(i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) (*discordgo.User, string, []string, bool) {
	user := subUser(i, opts, "user")
	if user == nil || user.ID == "" {
		_ = respondEphemeral(b.session, i, "Choose a member.")
		return nil, "", nil, false
	}
	if user.Bot || (b.session != nil && b.session.State != nil && b.session.State.User != nil && user.ID == b.session.State.User.ID) {
		_ = respondEphemeral(b.session, i, "Bots do not earn XP.")
		return nil, "", nil, false
	}
	member := resolvedMember(i, user.ID)
	name := display(user, member)
	var roles []string
	if member != nil {
		roles = member.Roles
		if roles == nil {
			roles = []string{}
		}
	}
	return user, name, roles, true
}

func rankEmbed(name string, row store.LevelStanding, found bool) *discordgo.MessageEmbed {
	if name == "" {
		name = row.UserID
	}
	level, into, span := domain.Progress(row.XP)
	desc := "Level 0\nNo XP yet."
	if found && row.XP > 0 {
		if level >= domain.MaxLevel {
			desc = fmt.Sprintf("Level %d · #%d\n%s XP\n`%s`", level, row.Rank, grouped(row.XP), domain.XPBar(into, span, 16))
		} else {
			desc = fmt.Sprintf("Level %d · #%d\n%s XP\n%s/%s XP to level %d\n`%s`",
				level, row.Rank, grouped(row.XP), grouped(into), grouped(span), level+1, domain.XPBar(into, span, 16))
		}
	}
	return &discordgo.MessageEmbed{Title: "Rank · " + name, Description: desc, Color: colorCopper}
}

func leaderboardEmbed(page, total int, rows []store.LevelStanding) *discordgo.MessageEmbed {
	if total == 0 {
		return &discordgo.MessageEmbed{Title: "Leaderboard", Description: "No one has XP yet.", Color: colorCopper}
	}
	if len(rows) == 0 {
		return &discordgo.MessageEmbed{Title: "Leaderboard", Description: "That page is empty.", Color: colorCopper}
	}
	var body strings.Builder
	for _, row := range rows {
		name := row.UserName
		if name == "" {
			name = row.UserID
		}
		fmt.Fprintf(&body, "%d. %s · level %d · %s XP\n", row.Rank, name, row.Level, grouped(row.XP))
	}
	pages := (total + domain.LevelPageSize - 1) / domain.LevelPageSize
	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Leaderboard · page %d of %d", page, pages),
		Description: strings.TrimRight(body.String(), "\n"),
		Color:       colorCopper,
	}
}

func levelLogEmbed(events []store.LevelEvent) *discordgo.MessageEmbed {
	if len(events) == 0 {
		return &discordgo.MessageEmbed{Title: "Level ups", Description: "No level ups yet.", Color: colorMist}
	}
	var body strings.Builder
	for _, event := range events {
		name := event.UserName
		if name == "" {
			name = event.UserID
		}
		fmt.Fprintf(&body, "%s · %s reached level %d\n", event.CreatedAt.UTC().Format("Jan 2 15:04"), name, event.Level)
	}
	return &discordgo.MessageEmbed{
		Title:       "Level ups",
		Description: strings.TrimRight(body.String(), "\n"),
		Color:       colorMist,
	}
}

func grouped(n int) string {
	if n < 0 {
		return "-" + grouped(-n)
	}
	raw := strconv.Itoa(n)
	if len(raw) <= 3 {
		return raw
	}
	lead := len(raw) % 3
	if lead == 0 {
		lead = 3
	}
	var body strings.Builder
	body.WriteString(raw[:lead])
	for i := lead; i < len(raw); i += 3 {
		body.WriteByte(',')
		body.WriteString(raw[i : i+3])
	}
	return body.String()
}

func commandSub(i *discordgo.InteractionCreate) (*discordgo.ApplicationCommandInteractionDataOption, []*discordgo.ApplicationCommandInteractionDataOption) {
	data := i.ApplicationCommandData()
	if len(data.Options) == 0 || data.Options[0] == nil || data.Options[0].Type != discordgo.ApplicationCommandOptionSubCommand {
		return nil, nil
	}
	return data.Options[0], data.Options[0].Options
}

func findOpt(opts []*discordgo.ApplicationCommandInteractionDataOption, name string) *discordgo.ApplicationCommandInteractionDataOption {
	for _, opt := range opts {
		if opt != nil && opt.Name == name {
			return opt
		}
	}
	return nil
}

func subInt(opts []*discordgo.ApplicationCommandInteractionDataOption, name string, fallback int) int {
	opt := findOpt(opts, name)
	if opt == nil || opt.Type != discordgo.ApplicationCommandOptionInteger {
		return fallback
	}
	return int(opt.IntValue())
}

func subUser(i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption, name string) *discordgo.User {
	opt := findOpt(opts, name)
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

func resolvedMember(i *discordgo.InteractionCreate, userID string) *discordgo.Member {
	data := i.ApplicationCommandData()
	if data.Resolved == nil || userID == "" {
		return nil
	}
	return data.Resolved.Members[userID]
}
