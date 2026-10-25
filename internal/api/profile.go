package api

import (
	"encoding/base64"
	"errors"
	"io"
	"strings"

	"github.com/BoxingOctopus/squirrelbot/internal/domain"
	"github.com/BoxingOctopus/squirrelbot/internal/store"
	"github.com/bwmarrin/discordgo"
	"github.com/gofiber/fiber/v2"
)

type botProfileDTO struct {
	Nick             string `json:"nick"`
	Username         string `json:"username"`
	AvatarURL        string `json:"avatarUrl"`
	DefaultAvatarURL string `json:"defaultAvatarUrl"`
	CustomAvatar     bool   `json:"customAvatar"`
}

func (s *Server) guildProfile(c *fiber.Ctx) error {
	profile, err := s.store.GetBotProfile(c.UserContext(), c.Params("guildID"))
	if err != nil {
		return fail(c, fiber.StatusInternalServerError, "Could not load this profile.")
	}
	return c.JSON(s.profileView(profile))
}

func (s *Server) putGuildProfile(c *fiber.Ctx) error {
	if !strings.HasPrefix(c.Get("Content-Type"), "multipart/form-data") {
		return fail(c, fiber.StatusBadRequest, "Could not read that request.")
	}
	nick, err := domain.CleanNick(c.FormValue("nick"))
	if err != nil {
		return fail(c, fiber.StatusBadRequest, err.Error())
	}
	guildID := c.Params("guildID")
	current, err := s.store.GetBotProfile(c.UserContext(), guildID)
	if err != nil {
		return fail(c, fiber.StatusInternalServerError, "Could not load this profile.")
	}
	avatar, avatarType, avatarChanged, err := s.readProfileAvatar(c, current)
	if err != nil {
		return fail(c, fiber.StatusBadRequest, err.Error())
	}
	nickChanged := nick != current.Nick
	if !nickChanged && !avatarChanged {
		return c.JSON(s.profileView(current))
	}
	if err := s.requireBot(); err != nil {
		return fail(c, fiber.StatusServiceUnavailable, err.Error())
	}
	if err := s.bot.ApplyGuildProfile(c.UserContext(), guildID, nick, nickChanged, avatar, avatarType, avatarChanged); err != nil {
		s.log.Warn("guild profile", "guild", guildID, "status", discordStatus(err))
		return fail(c, fiber.StatusBadGateway, profileDiscordMessage(err, nickChanged))
	}
	next := current
	next.Nick = nick
	if avatarChanged {
		next.Avatar = avatar
		next.AvatarType = avatarType
	}
	if err := s.store.SaveBotProfile(c.UserContext(), next); err != nil {
		return fail(c, fiber.StatusInternalServerError, "Discord updated, but the profile could not be saved.")
	}
	saved, err := s.store.GetBotProfile(c.UserContext(), guildID)
	if err != nil {
		return fail(c, fiber.StatusInternalServerError, "Could not load this profile.")
	}
	return c.JSON(s.profileView(saved))
}

func (s *Server) readProfileAvatar(c *fiber.Ctx, current store.BotProfile) (data []byte, contentType string, changed bool, err error) {
	file, fileErr := c.FormFile("avatar")
	if fileErr == nil && file != nil && file.Size > 0 {
		if file.Size > domain.MaxAvatarBytes {
			return nil, "", false, errors.New("pictures must be 1 MB or smaller")
		}
		opened, err := file.Open()
		if err != nil {
			return nil, "", false, errors.New("Could not read that picture.")
		}
		defer opened.Close()
		raw, err := io.ReadAll(io.LimitReader(opened, domain.MaxAvatarBytes+1))
		if err != nil {
			return nil, "", false, errors.New("Could not read that picture.")
		}
		data, contentType, err = domain.NormalizeAvatar(raw)
		if err != nil {
			return nil, "", false, err
		}
		return data, contentType, true, nil
	}
	if c.FormValue("clearAvatar") == "1" && len(current.Avatar) > 0 {
		return nil, "", true, nil
	}
	return current.Avatar, current.AvatarType, false, nil
}

func (s *Server) profileView(profile store.BotProfile) botProfileDTO {
	username, defaultURL := "", ""
	if s.bot != nil && s.bot.Ready() {
		username, defaultURL = s.bot.Account()
	}
	customURL := avatarDataURL(profile.AvatarType, profile.Avatar)
	view := botProfileDTO{
		Nick:             profile.Nick,
		Username:         username,
		AvatarURL:        defaultURL,
		DefaultAvatarURL: defaultURL,
		CustomAvatar:     customURL != "",
	}
	if customURL != "" {
		view.AvatarURL = customURL
	}
	return view
}

func avatarDataURL(contentType string, data []byte) string {
	var prefix string
	switch contentType {
	case "image/png":
		prefix = "data:image/png;base64,"
	case "image/jpeg":
		prefix = "data:image/jpeg;base64,"
	case "image/gif":
		prefix = "data:image/gif;base64,"
	default:
		return ""
	}
	if len(data) == 0 {
		return ""
	}
	return prefix + base64.StdEncoding.EncodeToString(data)
}

func discordStatus(err error) int {
	var rest *discordgo.RESTError
	if errors.As(err, &rest) && rest.Response != nil {
		return rest.Response.StatusCode
	}
	return 0
}

func profileDiscordMessage(err error, nickChanged bool) string {
	switch discordStatus(err) {
	case 403:
		if nickChanged {
			return "The bot needs the Change Nickname permission in this server. Add the server again to grant it."
		}
		return "Discord refused that picture."
	case 400:
		return "Discord rejected that name or picture. Use a PNG, JPEG, or GIF under 1 MB, and a name of up to 32 characters."
	default:
		return "Discord refused that profile."
	}
}
