package api

import (
	"errors"
	"time"

	"github.com/BoxingOctopus/squirrelbot/internal/domain"
	"github.com/BoxingOctopus/squirrelbot/internal/store"
	"github.com/gofiber/fiber/v2"
)

func (s *Server) purges(c *fiber.Ctx) error {
	guild, err := s.store.GetGuild(c.UserContext(), c.Params("guildID"))
	if err != nil {
		return fail(c, fiber.StatusInternalServerError, "Could not load timed purge.")
	}
	items, err := s.store.ListChannelPurges(c.UserContext(), guild.ID)
	if err != nil {
		return fail(c, fiber.StatusInternalServerError, "Could not load timed purge.")
	}
	schedules := make([]fiber.Map, 0, len(items))
	for _, item := range items {
		schedules = append(schedules, viewPurge(item, guild.PurgeEnabled))
	}
	return c.JSON(fiber.Map{"enabled": guild.PurgeEnabled, "schedules": schedules})
}

func (s *Server) putPurgeEnabled(c *fiber.Ctx) error {
	if !flagPresent(c.Body(), "enabled") {
		return fail(c, fiber.StatusBadRequest, "Choose on or off.")
	}
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fail(c, fiber.StatusBadRequest, "Could not read that request.")
	}
	if err := s.store.SetPurgeEnabled(c.UserContext(), c.Params("guildID"), body.Enabled); err != nil {
		return fail(c, fiber.StatusInternalServerError, "Could not save that setting.")
	}
	return c.JSON(fiber.Map{"enabled": body.Enabled})
}

func (s *Server) savePurge(c *fiber.Ctx) error {
	if err := s.requireBot(); err != nil {
		return fail(c, fiber.StatusServiceUnavailable, err.Error())
	}
	var body struct {
		ChannelID string `json:"channelId"`
		Preset    string `json:"preset"`
		Minutes   *int   `json:"minutes"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fail(c, fiber.StatusBadRequest, "Could not read that request.")
	}
	interval, err := domain.ParsePurgeInterval(body.Preset, body.Minutes)
	if err != nil {
		return fail(c, fiber.StatusBadRequest, err.Error())
	}
	guildID := c.Params("guildID")
	if !domain.ValidSnowflake(body.ChannelID) {
		return fail(c, fiber.StatusBadRequest, "Choose a text channel.")
	}
	channel, err := s.bot.Session().Channel(body.ChannelID)
	if err != nil || channel.GuildID != guildID || !domain.PurgeChannelAllowed(int(channel.Type)) {
		return fail(c, fiber.StatusBadRequest, "Choose a text channel.")
	}
	saved, err := s.store.SaveChannelPurge(c.UserContext(), guildID, body.ChannelID, interval.Preset, interval.Seconds)
	if err != nil {
		s.log.Error("save purge", "err", err)
		return fail(c, fiber.StatusInternalServerError, "Could not save that schedule.")
	}
	guild, err := s.store.GetGuild(c.UserContext(), guildID)
	if err != nil {
		return fail(c, fiber.StatusInternalServerError, "Could not load timed purge.")
	}
	return c.JSON(viewPurge(saved, guild.PurgeEnabled))
}

func (s *Server) deletePurge(c *fiber.Ctx) error {
	channelID := c.Params("channelID")
	if !domain.ValidSnowflake(channelID) {
		return fail(c, fiber.StatusNotFound, "That channel has no schedule.")
	}
	err := s.store.DeleteChannelPurge(c.UserContext(), c.Params("guildID"), channelID)
	if errors.Is(err, store.ErrNotFound) {
		return fail(c, fiber.StatusNotFound, "That channel has no schedule.")
	}
	if err != nil {
		return fail(c, fiber.StatusInternalServerError, "Could not delete that schedule.")
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func viewPurge(item store.ChannelPurge, enabled bool) fiber.Map {
	state := "scheduled"
	switch item.Status {
	case "running":
		state = "running"
	case "missing":
		state = "missing"
	default:
		if !enabled {
			state = "paused"
		}
	}
	var next *time.Time
	if state == "scheduled" {
		next = item.NextRunAt
	}
	return fiber.Map{
		"channelId":       item.ChannelID,
		"preset":          item.Preset,
		"intervalSeconds": item.IntervalSeconds,
		"state":           state,
		"nextRunAt":       next,
		"lastRunAt":       item.LastRunAt,
		"removed":         item.LastRemoved,
		"skipped":         item.LastSkipped,
		"error":           item.Error,
	}
}
