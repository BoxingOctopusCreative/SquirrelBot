package api

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/BoxingOctopus/squirrelbot/internal/discord"
	"github.com/BoxingOctopus/squirrelbot/internal/domain"
	"github.com/BoxingOctopus/squirrelbot/internal/store"
	"github.com/bwmarrin/discordgo"
	"github.com/gofiber/fiber/v2"
)

type channelDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type int    `json:"type"`
}

type roleDTO struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Position int    `json:"position"`
	Managed  bool   `json:"managed"`
	Created  bool   `json:"created,omitempty"`
}

type metaEntry struct {
	channels []channelDTO
	roles    []roleDTO
	exp      time.Time
}

type metaCache struct {
	mu    sync.Mutex
	items map[string]metaEntry
}

func newMetaCache() *metaCache {
	return &metaCache{items: map[string]metaEntry{}}
}

func (s *Server) bundle(c *fiber.Ctx) error {
	guildID := c.Params("guildID")
	ctx := c.UserContext()
	guild, err := s.store.GetGuild(ctx, guildID)
	if err != nil {
		return fail(c, fiber.StatusInternalServerError, "Could not load this server.")
	}
	counts, err := s.store.Counts(ctx, guildID)
	if err != nil {
		return fail(c, fiber.StatusInternalServerError, "Could not load this server.")
	}
	nextKick, err := s.store.NextPendingKick(ctx, guildID)
	if err != nil {
		return fail(c, fiber.StatusInternalServerError, "Could not load this server.")
	}
	channels, roles, err := s.guildMeta(guildID)
	if err != nil {
		s.log.Warn("discord metadata", "guild", guildID, "err", err)
	}
	gate, err := s.store.GetGate(ctx, guildID)
	if err != nil {
		return fail(c, fiber.StatusInternalServerError, "Could not load the application.")
	}
	automod, err := s.store.GetAutomod(ctx, guildID)
	if err != nil {
		return fail(c, fiber.StatusInternalServerError, "Could not load automod.")
	}
	logs, err := s.store.GetLogConfig(ctx, guildID)
	if err != nil {
		return fail(c, fiber.StatusInternalServerError, "Could not load logging.")
	}
	escalationRoles, err := s.store.ListEscalationRoles(ctx, guildID)
	if err != nil {
		return fail(c, fiber.StatusInternalServerError, "Could not load escalation roles.")
	}
	starboard, err := s.store.GetStarboard(ctx, guildID)
	if err != nil {
		return fail(c, fiber.StatusInternalServerError, "Could not load the starboard.")
	}
	return c.JSON(fiber.Map{
		"guild": guild, "counts": counts, "nextKick": nextKick,
		"channels": channels, "roles": roles, "gate": gate, "automod": automod,
		"logging": logs, "escalationRoles": escalationRoles, "starboard": starboard,
	})
}

func (s *Server) guildMeta(guildID string) ([]channelDTO, []roleDTO, error) {
	s.meta.mu.Lock()
	if entry, ok := s.meta.items[guildID]; ok && time.Now().Before(entry.exp) {
		s.meta.mu.Unlock()
		return entry.channels, entry.roles, nil
	}
	s.meta.mu.Unlock()
	if s.bot == nil || !s.bot.Ready() {
		return []channelDTO{}, []roleDTO{}, errors.New("bot is not connected")
	}
	session := s.bot.Session()
	rawChannels, err := session.GuildChannels(guildID)
	if err != nil {
		return []channelDTO{}, []roleDTO{}, err
	}
	rawRoles, err := session.GuildRoles(guildID)
	if err != nil {
		return []channelDTO{}, []roleDTO{}, err
	}
	channels := []channelDTO{}
	for _, channel := range rawChannels {
		if channel.Type != discordgo.ChannelTypeGuildText && channel.Type != discordgo.ChannelTypeGuildNews {
			continue
		}
		channels = append(channels, channelDTO{ID: channel.ID, Name: channel.Name, Type: int(channel.Type)})
	}
	sort.Slice(channels, func(i, j int) bool { return channels[i].Name < channels[j].Name })
	roles := []roleDTO{}
	for _, role := range rawRoles {
		if role.ID == guildID {
			continue
		}
		roles = append(roles, roleDTO{ID: role.ID, Name: role.Name, Position: role.Position, Managed: role.Managed})
	}
	sort.Slice(roles, func(i, j int) bool { return roles[i].Position > roles[j].Position })
	s.meta.mu.Lock()
	s.meta.items[guildID] = metaEntry{channels: channels, roles: roles, exp: time.Now().Add(30 * time.Second)}
	s.meta.mu.Unlock()
	return channels, roles, nil
}

func (s *Server) forgetGuildMeta(guildID string) {
	s.meta.mu.Lock()
	delete(s.meta.items, guildID)
	s.meta.mu.Unlock()
}

func (s *Server) patchSettings(c *fiber.Ctx) error {
	var body store.Guild
	if err := c.BodyParser(&body); err != nil {
		return fail(c, fiber.StatusBadRequest, "Could not read that request.")
	}
	body.ID = c.Params("guildID")
	if !optionalID(body.ModLogChannelID) || !optionalID(body.ModRoleID) || !optionalID(body.AdminRoleID) || !optionalID(body.WelcomeChannelID) || !optionalID(body.AutoroleID) {
		return fail(c, fiber.StatusBadRequest, "One of those Discord ids is not valid.")
	}
	if !domain.ValidPrefix(body.Prefix) {
		return fail(c, fiber.StatusBadRequest, "Prefix must be 1 to 5 characters and cannot contain a space or slash.")
	}
	body.WelcomeMessage = strings.TrimSpace(body.WelcomeMessage)
	if len(body.WelcomeMessage) > domain.MaxWelcome {
		return fail(c, fiber.StatusBadRequest, "Welcome message is too long.")
	}
	if err := s.store.UpdateGuildSettings(c.UserContext(), body); err != nil {
		return fail(c, fiber.StatusInternalServerError, "Could not save settings.")
	}
	updated, err := s.store.GetGuild(c.UserContext(), body.ID)
	if err != nil {
		return fail(c, fiber.StatusInternalServerError, "Could not save settings.")
	}
	return c.JSON(updated)
}

func (s *Server) cases(c *fiber.Ctx) error {
	items, err := s.store.ListCases(c.UserContext(), c.Params("guildID"), 50)
	if err != nil {
		return fail(c, fiber.StatusInternalServerError, "Could not load cases.")
	}
	return c.JSON(fiber.Map{"cases": items})
}

func (s *Server) announcements(c *fiber.Ctx) error {
	items, err := s.store.ListAnnouncements(c.UserContext(), c.Params("guildID"))
	if err != nil {
		return fail(c, fiber.StatusInternalServerError, "Could not load announcements.")
	}
	return c.JSON(fiber.Map{"announcements": items})
}

func (s *Server) createAnnouncement(c *fiber.Ctx) error {
	var body struct {
		ChannelID   string     `json:"channelId"`
		Title       string     `json:"title"`
		Body        string     `json:"body"`
		PingRoleID  string     `json:"pingRoleId"`
		ScheduledAt *time.Time `json:"scheduledAt"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fail(c, fiber.StatusBadRequest, "Could not read that request.")
	}
	if !domain.ValidSnowflake(body.ChannelID) || !optionalID(body.PingRoleID) || body.PingRoleID == c.Params("guildID") {
		return fail(c, fiber.StatusBadRequest, "Choose a channel, and a role if you want a ping.")
	}
	now := time.Now()
	when := body.ScheduledAt
	sendNow := when == nil || !when.After(now.Add(time.Minute))
	if sendNow {
		when = nil
	}
	if err := domain.ValidateAnnouncement(body.Title, body.Body, when, now); err != nil {
		return fail(c, fiber.StatusBadRequest, err.Error())
	}
	session := c.Locals("session").(store.Session)
	item := store.Announcement{
		GuildID: c.Params("guildID"), ChannelID: body.ChannelID, AuthorID: session.UserID,
		AuthorName: session.Username, Title: strings.TrimSpace(body.Title), Body: strings.TrimSpace(body.Body),
		PingRoleID: body.PingRoleID, ScheduledAt: when, Status: "scheduled",
	}
	if sendNow {
		if err := s.requireBot(); err != nil {
			return fail(c, fiber.StatusServiceUnavailable, err.Error())
		}
		messageID, err := s.bot.DeliverAnnouncement(item)
		if err != nil {
			return fail(c, fiber.StatusBadGateway, "Discord refused the announcement.")
		}
		sent := time.Now()
		item.Status = "sent"
		item.MessageID = messageID
		item.SentAt = &sent
		item.ScheduledAt = &sent
	}
	created, err := s.store.CreateAnnouncement(c.UserContext(), item)
	if err != nil {
		s.log.Error("save announcement", "err", err)
		return fail(c, fiber.StatusInternalServerError, "Could not save the announcement.")
	}
	return c.Status(fiber.StatusCreated).JSON(created)
}

func (s *Server) deleteAnnouncement(c *fiber.Ctx) error {
	id, ok := pathID(c, "announcementID")
	if !ok {
		return fail(c, fiber.StatusNotFound, "Unknown announcement.")
	}
	if err := s.store.DeleteAnnouncement(c.UserContext(), c.Params("guildID"), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return fail(c, fiber.StatusNotFound, "That announcement was already sent, or it does not exist.")
		}
		return fail(c, fiber.StatusInternalServerError, "Could not delete the announcement.")
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (s *Server) sendAnnouncement(c *fiber.Ctx) error {
	id, ok := pathID(c, "announcementID")
	if !ok {
		return fail(c, fiber.StatusNotFound, "Unknown announcement.")
	}
	item, err := s.store.GetAnnouncement(c.UserContext(), c.Params("guildID"), id)
	if err != nil {
		return fail(c, fiber.StatusNotFound, "Unknown announcement.")
	}
	if item.Status == "sent" {
		return fail(c, fiber.StatusConflict, "That announcement was already sent.")
	}
	if err := s.requireBot(); err != nil {
		return fail(c, fiber.StatusServiceUnavailable, err.Error())
	}
	messageID, err := s.bot.DeliverAnnouncement(item)
	if err != nil {
		_ = s.store.MarkAnnouncementFailed(c.UserContext(), item.ID, "Discord refused the announcement.")
		return fail(c, fiber.StatusBadGateway, "Discord refused the announcement.")
	}
	if err := s.store.MarkAnnouncementSent(c.UserContext(), item.ID, messageID); err != nil {
		return fail(c, fiber.StatusInternalServerError, "Discord accepted it, but the record did not update.")
	}
	updated, _ := s.store.GetAnnouncement(c.UserContext(), c.Params("guildID"), id)
	return c.JSON(updated)
}

func (s *Server) putEscalationRoles(c *fiber.Ctx) error {
	var body struct {
		Roles []store.EscalationRole `json:"roles"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fail(c, fiber.StatusBadRequest, "Could not read that request.")
	}
	if len(body.Roles) > 25 {
		return fail(c, fiber.StatusBadRequest, "Keep the allowlist to 25 roles.")
	}
	seen := map[string]bool{}
	for i := range body.Roles {
		role := body.Roles[i]
		if !domain.ValidSnowflake(role.RoleID) || seen[role.RoleID] || role.RoleID == c.Params("guildID") {
			return fail(c, fiber.StatusBadRequest, "Each allowlist entry needs a distinct role.")
		}
		if role.Tier != "mod" && role.Tier != "admin" {
			return fail(c, fiber.StatusBadRequest, "Tier must be mod or admin.")
		}
		if role.MaxDurationSeconds < int(domain.MinEscalation.Seconds()) || role.MaxDurationSeconds > int(domain.MaxEscalation.Seconds()) {
			return fail(c, fiber.StatusBadRequest, "The maximum grant must be between 5 minutes and 30 days.")
		}
		seen[role.RoleID] = true
		body.Roles[i].GuildID = c.Params("guildID")
	}
	if err := s.store.ReplaceEscalationRoles(c.UserContext(), c.Params("guildID"), body.Roles); err != nil {
		return fail(c, fiber.StatusInternalServerError, "Could not save the allowlist.")
	}
	roles, err := s.store.ListEscalationRoles(c.UserContext(), c.Params("guildID"))
	if err != nil {
		return fail(c, fiber.StatusInternalServerError, "Could not save the allowlist.")
	}
	return c.JSON(fiber.Map{"roles": roles})
}

func (s *Server) escalations(c *fiber.Ctx) error {
	items, err := s.store.ListEscalations(c.UserContext(), c.Params("guildID"))
	if err != nil {
		return fail(c, fiber.StatusInternalServerError, "Could not load admin firecall access.")
	}
	return c.JSON(fiber.Map{"escalations": items})
}

func (s *Server) createEscalation(c *fiber.Ctx) error {
	var body struct {
		UserID   string `json:"userId"`
		RoleID   string `json:"roleId"`
		Duration string `json:"duration"`
		Reason   string `json:"reason"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fail(c, fiber.StatusBadRequest, "Could not read that request.")
	}
	if !domain.ValidSnowflake(body.UserID) || !domain.ValidSnowflake(body.RoleID) {
		return fail(c, fiber.StatusBadRequest, "Choose a member and a role.")
	}
	length, err := domain.ParseDuration(body.Duration)
	if err != nil {
		return fail(c, fiber.StatusBadRequest, err.Error())
	}
	reason, err := domain.CleanReason(body.Reason)
	if err != nil || reason == "" {
		return fail(c, fiber.StatusBadRequest, "Add a reason.")
	}
	if err := s.requireBot(); err != nil {
		return fail(c, fiber.StatusServiceUnavailable, err.Error())
	}
	session := c.Locals("session").(store.Session)
	actor, err := s.bot.Session().GuildMember(c.Params("guildID"), session.UserID)
	if err != nil {
		return fail(c, fiber.StatusForbidden, "Discord could not confirm your roles in that server.")
	}
	member, err := s.bot.Session().GuildMember(c.Params("guildID"), body.UserID)
	name := body.UserID
	if err == nil && member.User != nil {
		name = member.User.Username
		if member.Nick != "" {
			name = member.Nick
		}
	}
	created, err := s.bot.GrantEscalation(discord.GrantInput{
		GuildID: c.Params("guildID"), UserID: body.UserID, UserName: name, RoleID: body.RoleID,
		ActorID: session.UserID, ActorName: session.Username, ActorRoles: actor.Roles, CheckActor: true,
		Reason: reason, Duration: length,
	})
	if err != nil {
		return fail(c, fiber.StatusBadRequest, err.Error())
	}
	return c.Status(fiber.StatusCreated).JSON(created)
}

func (s *Server) revokeEscalation(c *fiber.Ctx) error {
	id, ok := pathID(c, "escalationID")
	if !ok {
		return fail(c, fiber.StatusNotFound, "Unknown grant.")
	}
	var body struct {
		Reason string `json:"reason"`
	}
	_ = c.BodyParser(&body)
	reason, err := domain.CleanReason(body.Reason)
	if err != nil {
		return fail(c, fiber.StatusBadRequest, err.Error())
	}
	if err := s.requireBot(); err != nil {
		return fail(c, fiber.StatusServiceUnavailable, err.Error())
	}
	session := c.Locals("session").(store.Session)
	if err := s.bot.RevokeEscalation(c.Params("guildID"), id, session.UserID, session.Username, reason); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return fail(c, fiber.StatusNotFound, "Unknown grant.")
		}
		return fail(c, fiber.StatusBadRequest, err.Error())
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (s *Server) putGate(c *fiber.Ctx) error {
	var gate store.RoleGate
	if err := c.BodyParser(&gate); err != nil {
		return fail(c, fiber.StatusBadRequest, "Could not read that request.")
	}
	gate.GuildID = c.Params("guildID")
	gate.KickReason = strings.TrimSpace(gate.KickReason)
	gate.DMMessage = strings.TrimSpace(gate.DMMessage)
	if len(gate.KickReason) == 0 || len(gate.KickReason) > domain.MaxReason {
		return fail(c, fiber.StatusBadRequest, "Write a kick reason.")
	}
	if len(gate.DMMessage) > domain.MaxWelcome {
		return fail(c, fiber.StatusBadRequest, "The direct message is too long.")
	}
	if !optionalID(gate.LogChannelID) {
		return fail(c, fiber.StatusBadRequest, "Log channel is not a valid id.")
	}
	if len(gate.ExemptRoleIDs) > 25 {
		return fail(c, fiber.StatusBadRequest, "Keep exemptions to 25 roles.")
	}
	for _, id := range gate.ExemptRoleIDs {
		if !domain.ValidSnowflake(id) {
			return fail(c, fiber.StatusBadRequest, "An exempt role id is not valid.")
		}
	}
	questions, err := domain.CleanGateQuestions(gate.Questions)
	if err != nil {
		return fail(c, fiber.StatusBadRequest, err.Error())
	}
	gate.Questions = questions
	if err := domain.ValidateGate(gate.Enabled, time.Duration(gate.DeadlineSeconds)*time.Second, time.Duration(gate.WarnBeforeSeconds)*time.Second); err != nil {
		return fail(c, fiber.StatusBadRequest, err.Error())
	}
	existing, err := s.store.GetGate(c.UserContext(), gate.GuildID)
	if err != nil {
		return fail(c, fiber.StatusInternalServerError, "Could not save the application.")
	}
	gate.ChannelID = existing.ChannelID
	gate.MessageID = existing.MessageID
	gate.Emoji = ""
	if err := s.store.SaveGate(c.UserContext(), gate); err != nil {
		return fail(c, fiber.StatusInternalServerError, "Could not save the application.")
	}
	saved, err := s.store.GetGate(c.UserContext(), gate.GuildID)
	if err != nil {
		return fail(c, fiber.StatusInternalServerError, "Could not save the application.")
	}
	return c.JSON(saved)
}

func (s *Server) gateMembers(c *fiber.Ctx) error {
	pending, recent, err := s.store.ListGateMembers(c.UserContext(), c.Params("guildID"))
	if err != nil {
		return fail(c, fiber.StatusInternalServerError, "Could not load the application queue.")
	}
	return c.JSON(fiber.Map{"pending": pending, "recent": recent})
}

func (s *Server) gatePanel(c *fiber.Ctx) error {
	var body struct {
		ChannelID string `json:"channelId"`
	}
	if err := c.BodyParser(&body); err != nil || !domain.ValidSnowflake(body.ChannelID) {
		return fail(c, fiber.StatusBadRequest, "Choose a channel.")
	}
	if err := s.requireBot(); err != nil {
		return fail(c, fiber.StatusServiceUnavailable, err.Error())
	}
	if err := s.bot.PostGatePanel(c.Params("guildID"), body.ChannelID); err != nil {
		return fail(c, fiber.StatusBadGateway, err.Error())
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (s *Server) putAutomod(c *fiber.Ctx) error {
	var cfg store.Automod
	if err := c.BodyParser(&cfg); err != nil {
		return fail(c, fiber.StatusBadRequest, "Could not read that request.")
	}
	cfg.GuildID = c.Params("guildID")
	switch cfg.Action {
	case "delete", "warn", "timeout", "kick":
	default:
		return fail(c, fiber.StatusBadRequest, "Choose what automod should do.")
	}
	if cfg.MentionLimit < 0 || cfg.MentionLimit > 50 || cfg.CapsPercent < 0 || cfg.CapsPercent > 100 || cfg.CapsMinLength < 0 || cfg.CapsMinLength > 200 {
		return fail(c, fiber.StatusBadRequest, "Check the mention and caps limits.")
	}
	if cfg.TimeoutSeconds < 60 || cfg.TimeoutSeconds > int(domain.MaxDiscordTimeout.Seconds()) {
		return fail(c, fiber.StatusBadRequest, "Automod timeouts must be between 1 minute and 28 days.")
	}
	words := make([]string, 0, len(cfg.BadWords))
	for _, word := range cfg.BadWords {
		word = strings.ToLower(strings.TrimSpace(word))
		if word == "" {
			continue
		}
		if len(word) > domain.MaxBadWordLen || strings.ContainsAny(word, "\n\t") {
			return fail(c, fiber.StatusBadRequest, "Each blocked word must be 40 characters or fewer.")
		}
		words = append(words, word)
	}
	if len(words) > domain.MaxBadWords {
		return fail(c, fiber.StatusBadRequest, "Keep the blocked word list to 100 entries.")
	}
	cfg.BadWords = words
	if !idList(cfg.IgnoredChannelIDs, 50) || !idList(cfg.IgnoredRoleIDs, 50) {
		return fail(c, fiber.StatusBadRequest, "Ignored channels or roles contain an invalid id.")
	}
	if err := s.store.SaveAutomod(c.UserContext(), cfg); err != nil {
		return fail(c, fiber.StatusInternalServerError, "Could not save automod.")
	}
	saved, err := s.store.GetAutomod(c.UserContext(), cfg.GuildID)
	if err != nil {
		return fail(c, fiber.StatusInternalServerError, "Could not save automod.")
	}
	return c.JSON(saved)
}

func (s *Server) putLogging(c *fiber.Ctx) error {
	var cfg store.LogConfig
	if err := c.BodyParser(&cfg); err != nil {
		return fail(c, fiber.StatusBadRequest, "Could not read that request.")
	}
	cfg.GuildID = c.Params("guildID")
	if !optionalID(cfg.ChannelID) {
		return fail(c, fiber.StatusBadRequest, "Choose a log channel.")
	}
	if err := s.store.SaveLogConfig(c.UserContext(), cfg); err != nil {
		return fail(c, fiber.StatusInternalServerError, "Could not save logging.")
	}
	saved, err := s.store.GetLogConfig(c.UserContext(), cfg.GuildID)
	if err != nil {
		return fail(c, fiber.StatusInternalServerError, "Could not save logging.")
	}
	return c.JSON(saved)
}

func (s *Server) putStarboard(c *fiber.Ctx) error {
	var board store.Starboard
	if err := c.BodyParser(&board); err != nil {
		return fail(c, fiber.StatusBadRequest, "Could not read that request.")
	}
	board.GuildID = c.Params("guildID")
	if board.IgnoredChannelIDs == nil {
		board.IgnoredChannelIDs = []string{}
	}
	if err := domain.ValidateStarboard(board.Enabled, board.ChannelID, board.Threshold, board.IgnoredChannelIDs); err != nil {
		return fail(c, fiber.StatusBadRequest, err.Error())
	}
	if err := s.store.SaveStarboard(c.UserContext(), board); err != nil {
		return fail(c, fiber.StatusInternalServerError, "Could not save the starboard.")
	}
	saved, err := s.store.GetStarboard(c.UserContext(), board.GuildID)
	if err != nil {
		return fail(c, fiber.StatusInternalServerError, "Could not save the starboard.")
	}
	return c.JSON(saved)
}

func (s *Server) createSetupChannel(c *fiber.Ctx) error {
	var body struct {
		Name    string `json:"name"`
		Purpose string `json:"purpose"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fail(c, fiber.StatusBadRequest, "Could not read that request.")
	}
	name, err := domain.NormalizeChannelName(body.Name)
	if err != nil {
		return fail(c, fiber.StatusBadRequest, err.Error())
	}
	topic, private, ok := domain.SetupPurpose(body.Purpose)
	if !ok {
		return fail(c, fiber.StatusBadRequest, "Choose logging, announcements, or roles.")
	}
	spec := domain.SetupChannel{Name: name, Topic: topic, Private: private}
	if err := s.requireBot(); err != nil {
		return fail(c, fiber.StatusServiceUnavailable, err.Error())
	}
	var staff []string
	var userID string
	if spec.Private {
		session := c.Locals("session").(store.Session)
		userID = session.UserID
		guild, err := s.store.GetGuild(c.UserContext(), c.Params("guildID"))
		if err == nil {
			staff = []string{guild.ModRoleID, guild.AdminRoleID}
		}
	}
	id, name, created, err := s.bot.EnsureTextChannel(c.Params("guildID"), spec, staff, userID)
	if err != nil {
		return fail(c, fiber.StatusBadGateway, err.Error())
	}
	s.forgetGuildMeta(c.Params("guildID"))
	status := fiber.StatusOK
	if created {
		status = fiber.StatusCreated
	}
	return c.Status(status).JSON(fiber.Map{"id": id, "name": name, "created": created})
}

func (s *Server) createRole(c *fiber.Ctx) error {
	var body struct {
		Name string `json:"name"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fail(c, fiber.StatusBadRequest, "Could not read that request.")
	}
	name, err := domain.CleanRoleName(body.Name)
	if err != nil {
		return fail(c, fiber.StatusBadRequest, err.Error())
	}
	if err := s.requireBot(); err != nil {
		return fail(c, fiber.StatusServiceUnavailable, err.Error())
	}
	id, name, position, created, err := s.bot.EnsureRole(c.Params("guildID"), name)
	if err != nil {
		return fail(c, fiber.StatusBadGateway, err.Error())
	}
	s.forgetGuildMeta(c.Params("guildID"))
	status := fiber.StatusOK
	if created {
		status = fiber.StatusCreated
	}
	return c.Status(status).JSON(roleDTO{ID: id, Name: name, Position: position, Managed: false, Created: created})
}

func (s *Server) reactionRoles(c *fiber.Ctx) error {
	items, err := s.store.ListReactionRoles(c.UserContext(), c.Params("guildID"))
	if err != nil {
		return fail(c, fiber.StatusInternalServerError, "Could not load reaction roles.")
	}
	return c.JSON(fiber.Map{"reactionRoles": items})
}

func (s *Server) setReactionChannel(c *fiber.Ctx) error {
	var body struct {
		ChannelID string `json:"channelId"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fail(c, fiber.StatusBadRequest, "Could not read that request.")
	}
	if !domain.ValidSnowflake(body.ChannelID) {
		return fail(c, fiber.StatusBadRequest, "Pick a roles channel.")
	}
	if err := s.requireBot(); err != nil {
		return fail(c, fiber.StatusServiceUnavailable, err.Error())
	}
	guild, err := s.store.GetGuild(c.UserContext(), c.Params("guildID"))
	if errors.Is(err, store.ErrNotFound) {
		return fail(c, fiber.StatusNotFound, "Unknown server.")
	} else if err != nil {
		return fail(c, fiber.StatusInternalServerError, "Could not save the roles channel.")
	}
	if guild.ReactionChannelID == body.ChannelID {
		return c.JSON(fiber.Map{"channelId": body.ChannelID, "messageId": guild.ReactionMessageID})
	}
	items, err := s.store.ListReactionRoles(c.UserContext(), guild.ID)
	if err != nil {
		return fail(c, fiber.StatusInternalServerError, "Could not save the roles channel.")
	}
	if len(items) > domain.MaxReactionRoles {
		return fail(c, fiber.StatusBadRequest, "That message can hold 20 reactions. Delete some before moving the channel.")
	}
	if len(items) == 0 {
		if guild.ReactionMessageID != "" {
			s.bot.RemoveReactionPanel(guild.ReactionChannelID, guild.ReactionMessageID)
		}
		if err := s.store.SetReactionPanel(c.UserContext(), guild.ID, body.ChannelID, ""); err != nil {
			return fail(c, fiber.StatusInternalServerError, "Could not save the roles channel.")
		}
		return c.JSON(fiber.Map{"channelId": body.ChannelID, "messageId": ""})
	}
	if duplicateEmoji(items) {
		return fail(c, fiber.StatusBadRequest, "Two reactions use the same emoji. Delete one before moving the channel.")
	}
	messageID, err := s.bot.PublishReactionPanel(body.ChannelID, "", s.reactionLines(guild.ID, items))
	if err != nil {
		return fail(c, fiber.StatusBadGateway, err.Error())
	}
	for _, item := range items {
		if err := s.bot.PlaceReaction(guild.ID, body.ChannelID, messageID, item.Emoji, item.RoleID); err != nil {
			s.bot.RemoveReactionPanel(body.ChannelID, messageID)
			return fail(c, fiber.StatusBadGateway, err.Error())
		}
	}
	if err := s.store.RetargetReactionRoles(c.UserContext(), guild.ID, body.ChannelID, messageID); err != nil {
		s.bot.RemoveReactionPanel(body.ChannelID, messageID)
		return fail(c, fiber.StatusInternalServerError, "Could not save the roles channel.")
	}
	if err := s.store.SetReactionPanel(c.UserContext(), guild.ID, body.ChannelID, messageID); err != nil {
		return fail(c, fiber.StatusInternalServerError, "Could not save the roles channel.")
	}
	if guild.ReactionMessageID != "" {
		s.bot.RemoveReactionPanel(guild.ReactionChannelID, guild.ReactionMessageID)
	} else {
		for _, item := range items {
			s.bot.LiftReaction(item.ChannelID, item.MessageID, item.Emoji)
		}
	}
	return c.JSON(fiber.Map{"channelId": body.ChannelID, "messageId": messageID})
}

func (s *Server) createReactionRole(c *fiber.Ctx) error {
	var body struct {
		Emoji  string `json:"emoji"`
		RoleID string `json:"roleId"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fail(c, fiber.StatusBadRequest, "Could not read that request.")
	}
	body.Emoji = strings.TrimSpace(body.Emoji)
	if !validEmoji(body.Emoji) || !domain.ValidSnowflake(body.RoleID) {
		return fail(c, fiber.StatusBadRequest, "Choose an emoji and a role.")
	}
	if err := s.requireBot(); err != nil {
		return fail(c, fiber.StatusServiceUnavailable, err.Error())
	}
	guild, err := s.store.GetGuild(c.UserContext(), c.Params("guildID"))
	if errors.Is(err, store.ErrNotFound) {
		return fail(c, fiber.StatusNotFound, "Unknown server.")
	} else if err != nil {
		return fail(c, fiber.StatusInternalServerError, "Could not save that reaction role.")
	}
	if !domain.ValidSnowflake(guild.ReactionChannelID) {
		return fail(c, fiber.StatusBadRequest, "Pick a roles channel first.")
	}
	items, err := s.store.ListReactionRoles(c.UserContext(), guild.ID)
	if err != nil {
		return fail(c, fiber.StatusInternalServerError, "Could not save that reaction role.")
	}
	existed := false
	for _, item := range items {
		if item.Emoji == body.Emoji {
			existed = true
			break
		}
	}
	if !existed && len(items) >= domain.MaxReactionRoles {
		return fail(c, fiber.StatusBadRequest, "That message can hold 20 reactions. Delete one before adding another.")
	}
	messageID, err := s.ensureReactionMessage(c.UserContext(), guild, items)
	if err != nil {
		return fail(c, fiber.StatusBadGateway, err.Error())
	}
	if !existed {
		if err := s.bot.PlaceReaction(guild.ID, guild.ReactionChannelID, messageID, body.Emoji, body.RoleID); err != nil {
			if len(items) == 0 {
				s.bot.RemoveReactionPanel(guild.ReactionChannelID, messageID)
				_ = s.store.SetReactionPanel(c.UserContext(), guild.ID, guild.ReactionChannelID, "")
			}
			return fail(c, fiber.StatusBadGateway, err.Error())
		}
	}
	saved, err := s.store.AddReactionRole(c.UserContext(), store.ReactionRole{
		GuildID: guild.ID, ChannelID: guild.ReactionChannelID, MessageID: messageID,
		Emoji: body.Emoji, RoleID: body.RoleID,
	})
	if err != nil {
		if !existed {
			s.bot.LiftReaction(guild.ReactionChannelID, messageID, body.Emoji)
			if len(items) == 0 {
				s.bot.RemoveReactionPanel(guild.ReactionChannelID, messageID)
				_ = s.store.SetReactionPanel(c.UserContext(), guild.ID, guild.ReactionChannelID, "")
			}
		}
		return fail(c, fiber.StatusInternalServerError, "Could not save that reaction role.")
	}
	next := make([]store.ReactionRole, 0, len(items)+1)
	replaced := false
	for _, item := range items {
		if item.Emoji == body.Emoji {
			item.RoleID = body.RoleID
			item.ChannelID = guild.ReactionChannelID
			item.MessageID = messageID
			replaced = true
		}
		next = append(next, item)
	}
	if !replaced {
		next = append(next, saved)
	}
	if _, err := s.bot.PublishReactionPanel(guild.ReactionChannelID, messageID, s.reactionLines(guild.ID, next)); err != nil {
		s.log.Warn("reaction panel", "guild", guild.ID, "err", err)
	}
	return c.Status(fiber.StatusCreated).JSON(saved)
}

func (s *Server) deleteReactionRole(c *fiber.Ctx) error {
	id, ok := pathID(c, "reactionID")
	if !ok {
		return fail(c, fiber.StatusNotFound, "Unknown reaction role.")
	}
	item, err := s.store.DeleteReactionRole(c.UserContext(), c.Params("guildID"), id)
	if errors.Is(err, store.ErrNotFound) {
		return fail(c, fiber.StatusNotFound, "Unknown reaction role.")
	} else if err != nil {
		return fail(c, fiber.StatusInternalServerError, "Could not delete that reaction role.")
	}
	if s.bot != nil && s.bot.Ready() {
		s.bot.LiftReaction(item.ChannelID, item.MessageID, item.Emoji)
		s.refreshReactionPanel(c, item.GuildID, item.ChannelID, item.MessageID)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (s *Server) ensureReactionMessage(ctx context.Context, guild store.Guild, items []store.ReactionRole) (string, error) {
	messageID, err := s.bot.PublishReactionPanel(guild.ReactionChannelID, guild.ReactionMessageID, s.reactionLines(guild.ID, items))
	if err != nil {
		return "", err
	}
	if messageID == guild.ReactionMessageID {
		return messageID, nil
	}
	if duplicateEmoji(items) {
		s.bot.RemoveReactionPanel(guild.ReactionChannelID, messageID)
		return "", errors.New("Two reactions use the same emoji. Delete one before continuing.")
	}
	for _, item := range items {
		if err := s.bot.PlaceReaction(guild.ID, guild.ReactionChannelID, messageID, item.Emoji, item.RoleID); err != nil {
			s.bot.RemoveReactionPanel(guild.ReactionChannelID, messageID)
			return "", err
		}
	}
	if len(items) > 0 {
		if err := s.store.RetargetReactionRoles(ctx, guild.ID, guild.ReactionChannelID, messageID); err != nil {
			s.bot.RemoveReactionPanel(guild.ReactionChannelID, messageID)
			return "", err
		}
	}
	if err := s.store.SetReactionPanel(ctx, guild.ID, guild.ReactionChannelID, messageID); err != nil {
		return "", err
	}
	if guild.ReactionMessageID != "" {
		s.bot.RemoveReactionPanel(guild.ReactionChannelID, guild.ReactionMessageID)
	} else {
		for _, item := range items {
			s.bot.LiftReaction(item.ChannelID, item.MessageID, item.Emoji)
		}
	}
	return messageID, nil
}

func (s *Server) refreshReactionPanel(c *fiber.Ctx, guildID, channelID, messageID string) {
	items, err := s.store.ListReactionRoles(c.UserContext(), guildID)
	if err != nil {
		s.log.Warn("reaction panel", "guild", guildID, "err", err)
		return
	}
	onMessage := make([]store.ReactionRole, 0, len(items))
	for _, item := range items {
		if item.MessageID == messageID {
			onMessage = append(onMessage, item)
		}
	}
	guild, err := s.store.GetGuild(c.UserContext(), guildID)
	if err != nil {
		s.log.Warn("reaction panel", "guild", guildID, "err", err)
		return
	}
	if len(onMessage) == 0 {
		s.bot.RemoveReactionPanel(channelID, messageID)
		if guild.ReactionMessageID == messageID && guild.ReactionChannelID != "" {
			_ = s.store.SetReactionPanel(c.UserContext(), guildID, guild.ReactionChannelID, "")
		}
		return
	}
	if _, err := s.bot.PublishReactionPanel(channelID, messageID, s.reactionLines(guildID, onMessage)); err != nil {
		s.log.Warn("reaction panel", "guild", guildID, "err", err)
	}
}

func (s *Server) reactionLines(guildID string, items []store.ReactionRole) []domain.ReactionLine {
	names, err := s.bot.RoleNames(guildID)
	if err != nil {
		s.log.Warn("reaction role names", "guild", guildID, "err", err)
		names = map[string]string{}
	}
	ordered := append([]store.ReactionRole{}, items...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].ID == 0 {
			return false
		}
		if ordered[j].ID == 0 {
			return true
		}
		return ordered[i].ID < ordered[j].ID
	})
	lines := make([]domain.ReactionLine, len(ordered))
	for i, item := range ordered {
		name := names[item.RoleID]
		if name == "" {
			name = "role"
		}
		lines[i] = domain.ReactionLine{Emoji: item.Emoji, RoleName: name}
	}
	return lines
}

func duplicateEmoji(items []store.ReactionRole) bool {
	seen := map[string]bool{}
	for _, item := range items {
		if seen[item.Emoji] {
			return true
		}
		seen[item.Emoji] = true
	}
	return false
}

func (s *Server) commands(c *fiber.Ctx) error {
	items, err := s.store.ListCommands(c.UserContext(), c.Params("guildID"))
	if err != nil {
		return fail(c, fiber.StatusInternalServerError, "Could not load commands.")
	}
	return c.JSON(fiber.Map{"commands": items})
}

func (s *Server) saveCommand(c *fiber.Ctx) error {
	var body store.CustomCommand
	if err := c.BodyParser(&body); err != nil {
		return fail(c, fiber.StatusBadRequest, "Could not read that request.")
	}
	body.GuildID = c.Params("guildID")
	body.Name = strings.ToLower(strings.TrimSpace(body.Name))
	body.Response = strings.TrimSpace(body.Response)
	if !domain.ValidCommandName(body.Name) {
		return fail(c, fiber.StatusBadRequest, "Command names use lowercase letters, numbers, _ or -.")
	}
	if body.Response == "" || len(body.Response) > domain.MaxCustomResponse {
		return fail(c, fiber.StatusBadRequest, "Write a response, up to 1500 characters.")
	}
	if err := s.store.SaveCommand(c.UserContext(), body); err != nil {
		return fail(c, fiber.StatusInternalServerError, "Could not save that command.")
	}
	return c.Status(fiber.StatusCreated).JSON(body)
}

func (s *Server) deleteCommand(c *fiber.Ctx) error {
	name := strings.ToLower(c.Params("name"))
	if !domain.ValidCommandName(name) {
		return fail(c, fiber.StatusNotFound, "Unknown command.")
	}
	if err := s.store.DeleteCommand(c.UserContext(), c.Params("guildID"), name); errors.Is(err, store.ErrNotFound) {
		return fail(c, fiber.StatusNotFound, "Unknown command.")
	} else if err != nil {
		return fail(c, fiber.StatusInternalServerError, "Could not delete that command.")
	}
	return c.SendStatus(fiber.StatusNoContent)
}

const (
	memberPageSize = 1000
	memberPageCap  = 10
)

func (s *Server) searchMembers(c *fiber.Ctx) error {
	query := strings.TrimSpace(c.Query("q"))
	if len(query) > 32 || strings.ContainsAny(query, "\n\t") {
		return fail(c, fiber.StatusBadRequest, "That name is too long.")
	}
	if err := s.requireBot(); err != nil {
		return fail(c, fiber.StatusServiceUnavailable, err.Error())
	}
	members, err := s.guildMembers(c.Params("guildID"))
	if err != nil {
		var rest *discordgo.RESTError
		if errors.As(err, &rest) && rest.Response != nil && rest.Response.StatusCode == fiber.StatusForbidden {
			return fail(c, fiber.StatusForbidden, "Turn on the Server Members Intent, then restart the bot.")
		}
		return fail(c, fiber.StatusBadGateway, "Discord did not return members.")
	}
	type result struct {
		ID          string `json:"id"`
		Username    string `json:"username"`
		DisplayName string `json:"displayName"`
	}
	out := []result{}
	for _, member := range members {
		username, display, ok := memberLabel(member)
		if !ok || !memberMatches(display, username, query) {
			continue
		}
		out = append(out, result{ID: member.User.ID, Username: username, DisplayName: display})
	}
	sort.Slice(out, func(i, j int) bool {
		left := strings.ToLower(out[i].DisplayName)
		right := strings.ToLower(out[j].DisplayName)
		if left == right {
			return out[i].Username < out[j].Username
		}
		return left < right
	})
	return c.JSON(fiber.Map{"members": out})
}

func (s *Server) guildMembers(guildID string) ([]*discordgo.Member, error) {
	var all []*discordgo.Member
	after := ""
	for page := 0; page < memberPageCap; page++ {
		batch, err := s.bot.Session().GuildMembers(guildID, after, memberPageSize)
		if err != nil {
			return nil, err
		}
		all = append(all, batch...)
		if len(batch) < memberPageSize {
			break
		}
		last := batch[len(batch)-1].User
		if last == nil || last.ID == "" || last.ID == after {
			break
		}
		after = last.ID
	}
	return all, nil
}

func memberLabel(member *discordgo.Member) (username, display string, ok bool) {
	if member == nil || member.User == nil || member.User.ID == "" {
		return "", "", false
	}
	username = member.User.Username
	display = username
	if member.Nick != "" {
		display = member.Nick
	} else if member.User.GlobalName != "" {
		display = member.User.GlobalName
	}
	return username, display, true
}

func memberMatches(display, username, query string) bool {
	needle := strings.ToLower(strings.TrimSpace(query))
	if needle == "" {
		return true
	}
	return strings.Contains(strings.ToLower(display), needle) || strings.Contains(strings.ToLower(username), needle)
}

func (s *Server) requireBot() error {
	if s.bot == nil || !s.bot.Ready() {
		return errors.New("The bot is still connecting to Discord.")
	}
	return nil
}

func optionalID(id string) bool {
	return domain.OptionalSnowflake(id)
}

func idList(ids []string, max int) bool {
	if len(ids) > max {
		return false
	}
	for _, id := range ids {
		if !domain.ValidSnowflake(id) {
			return false
		}
	}
	return true
}

func pathID(c *fiber.Ctx, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.Params(name), 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func validEmoji(emoji string) bool {
	if emoji == "" || len(emoji) > 64 || strings.ContainsAny(emoji, " \n\t") {
		return false
	}
	if strings.Contains(emoji, ":") {
		name, id, ok := strings.Cut(emoji, ":")
		return ok && name != "" && !strings.Contains(id, ":") && domain.ValidSnowflake(id)
	}
	return true
}
