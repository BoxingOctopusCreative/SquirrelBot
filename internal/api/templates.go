package api

import (
	"errors"
	"strings"

	"github.com/BoxingOctopus/squirrelbot/internal/domain"
	"github.com/BoxingOctopus/squirrelbot/internal/store"
	"github.com/gofiber/fiber/v2"
)

func (s *Server) announcementTemplates(c *fiber.Ctx) error {
	items, err := s.store.ListAnnouncementTemplates(c.UserContext(), c.Params("guildID"))
	if err != nil {
		return fail(c, fiber.StatusInternalServerError, "Could not load templates.")
	}
	return c.JSON(fiber.Map{"templates": items})
}

func (s *Server) createAnnouncementTemplate(c *fiber.Ctx) error {
	name, title, body, err := readAnnouncementTemplate(c)
	if err != nil {
		return err
	}
	created, err := s.store.CreateAnnouncementTemplate(c.UserContext(), store.AnnouncementTemplate{
		GuildID: c.Params("guildID"), Name: name, Title: title, Body: body,
	})
	if err != nil {
		return templateStoreError(c, err, "Could not save the template.")
	}
	return c.Status(fiber.StatusCreated).JSON(created)
}

func (s *Server) updateAnnouncementTemplate(c *fiber.Ctx) error {
	id, ok := pathID(c, "templateID")
	if !ok {
		return fail(c, fiber.StatusNotFound, "Unknown template.")
	}
	name, title, body, err := readAnnouncementTemplate(c)
	if err != nil {
		return err
	}
	updated, err := s.store.UpdateAnnouncementTemplate(c.UserContext(), store.AnnouncementTemplate{
		ID: id, GuildID: c.Params("guildID"), Name: name, Title: title, Body: body,
	})
	if err != nil {
		return templateStoreError(c, err, "Could not save the template.")
	}
	return c.JSON(updated)
}

func (s *Server) deleteAnnouncementTemplate(c *fiber.Ctx) error {
	id, ok := pathID(c, "templateID")
	if !ok {
		return fail(c, fiber.StatusNotFound, "Unknown template.")
	}
	if err := s.store.DeleteAnnouncementTemplate(c.UserContext(), c.Params("guildID"), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return fail(c, fiber.StatusNotFound, "Unknown template.")
		}
		return fail(c, fiber.StatusInternalServerError, "Could not delete the template.")
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (s *Server) renderAnnouncementTemplate(c *fiber.Ctx) error {
	id, ok := pathID(c, "templateID")
	if !ok {
		return fail(c, fiber.StatusNotFound, "Unknown template.")
	}
	vars := map[string]string{}
	if len(bytesTrim(c.Body())) > 0 {
		var body struct {
			Vars map[string]string `json:"vars"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fail(c, fiber.StatusBadRequest, "Could not read that request.")
		}
		vars = body.Vars
	}
	item, err := s.store.GetAnnouncementTemplate(c.UserContext(), c.Params("guildID"), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return fail(c, fiber.StatusNotFound, "Unknown template.")
		}
		return fail(c, fiber.StatusInternalServerError, "Could not load that template.")
	}
	guild, err := s.store.GetGuild(c.UserContext(), c.Params("guildID"))
	if err != nil {
		return fail(c, fiber.StatusInternalServerError, "Could not load that server.")
	}
	serverName := guild.Name
	if serverName == "" {
		serverName = "this server"
	}
	session := c.Locals("session").(store.Session)
	title, body, err := domain.RenderAnnouncementTemplate(item.Title, item.Body, domain.AnnouncementScope{
		Server: serverName,
		Author: session.Username,
		Vars:   vars,
	})
	if err != nil {
		return fail(c, fiber.StatusBadRequest, err.Error())
	}
	return c.JSON(fiber.Map{"title": title, "body": body})
}

func readAnnouncementTemplate(c *fiber.Ctx) (string, string, string, error) {
	var body struct {
		Name  string `json:"name"`
		Title string `json:"title"`
		Body  string `json:"body"`
	}
	if err := c.BodyParser(&body); err != nil {
		return "", "", "", fail(c, fiber.StatusBadRequest, "Could not read that request.")
	}
	name, title, text, err := domain.ValidateAnnouncementTemplate(body.Name, body.Title, body.Body)
	if err != nil {
		return "", "", "", fail(c, fiber.StatusBadRequest, err.Error())
	}
	return name, title, text, nil
}

func templateStoreError(c *fiber.Ctx, err error, fallback string) error {
	if errors.Is(err, store.ErrDuplicateName) {
		return fail(c, fiber.StatusConflict, "A template with that name already exists.")
	}
	if errors.Is(err, store.ErrNotFound) {
		return fail(c, fiber.StatusNotFound, "Unknown template.")
	}
	return fail(c, fiber.StatusInternalServerError, fallback)
}

func bytesTrim(body []byte) []byte {
	return []byte(strings.TrimSpace(string(body)))
}
