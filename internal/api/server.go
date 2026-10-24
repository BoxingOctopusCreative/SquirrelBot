package api

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BoxingOctopus/squirrelbot/internal/config"
	"github.com/BoxingOctopus/squirrelbot/internal/discord"
	"github.com/BoxingOctopus/squirrelbot/internal/secretbox"
	"github.com/BoxingOctopus/squirrelbot/internal/store"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
)

const (
	sessionCookie = "sb_session"
	stateCookie   = "sb_oauth_state"
)

type Server struct {
	app    *fiber.App
	cfg    config.Config
	store  *store.Store
	bot    *discord.Bot
	key    []byte
	log    *slog.Logger
	client *http.Client
	access *accessCache
	meta   *metaCache
	csp    string
}

func New(cfg config.Config, st *store.Store, bot *discord.Bot, log *slog.Logger) *Server {
	s := &Server{
		cfg:   cfg,
		store: st,
		bot:   bot,
		key:   secretbox.Key(cfg.SessionSecret),
		log:   log,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		access: newAccessCache(),
		meta:   newMetaCache(),
	}
	s.app = fiber.New(fiber.Config{
		BodyLimit:             1 << 20,
		DisableStartupMessage: true,
		ServerHeader:          "squirrelbot",
		ErrorHandler:          s.handleError,
	})
	s.csp = contentSecurityPolicy(cfg.WebDir)
	s.routes()
	return s
}

func (s *Server) Listen() error { return s.app.Listen(s.cfg.HTTPAddr) }

func (s *Server) Shutdown() error {
	return s.app.ShutdownWithTimeout(10 * time.Second)
}

func (s *Server) handleError(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		code = fiberErr.Code
	}
	if code >= 500 {
		s.log.Error("request", "path", c.Path(), "err", err)
		return c.Status(code).JSON(fiber.Map{"error": "Something went wrong."})
	}
	return c.Status(code).JSON(fiber.Map{"error": err.Error()})
}

func (s *Server) routes() {
	s.app.Use(func(c *fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("Referrer-Policy", "no-referrer")
		c.Set("Content-Security-Policy", s.csp)
		return c.Next()
	})
	s.app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"ok": true})
	})
	s.app.Get("/readyz", s.ready)
	s.app.Get("/auth/discord", s.startLogin)
	s.app.Get("/auth/discord/callback", s.finishLogin)
	s.app.Post("/auth/logout", s.requireSameOrigin, s.logout)

	api := s.app.Group("/api", s.requireSession, s.requireSameOrigin)
	api.Get("/me", s.me)
	api.Get("/guilds", s.guilds)
	guild := api.Group("/guilds/:guildID", s.requireGuild)
	guild.Get("/", s.bundle)
	guild.Patch("/", s.patchSettings)
	guild.Get("/cases", s.cases)
	guild.Get("/announcements", s.announcements)
	guild.Post("/announcements", s.createAnnouncement)
	guild.Delete("/announcements/:announcementID", s.deleteAnnouncement)
	guild.Post("/announcements/:announcementID/send", s.sendAnnouncement)
	guild.Get("/announcement-templates", s.announcementTemplates)
	guild.Post("/announcement-templates", s.createAnnouncementTemplate)
	guild.Put("/announcement-templates/:templateID", s.updateAnnouncementTemplate)
	guild.Delete("/announcement-templates/:templateID", s.deleteAnnouncementTemplate)
	guild.Post("/announcement-templates/:templateID/render", s.renderAnnouncementTemplate)
	guild.Put("/escalation-roles", s.putEscalationRoles)
	guild.Get("/escalations", s.escalations)
	guild.Post("/escalations", s.createEscalation)
	guild.Post("/escalations/:escalationID/revoke", s.revokeEscalation)
	guild.Put("/gate", s.putGate)
	guild.Get("/gate/members", s.gateMembers)
	guild.Post("/gate/panel", s.gatePanel)
	guild.Put("/automod", s.putAutomod)
	guild.Put("/logging", s.putLogging)
	guild.Put("/starboard", s.putStarboard)
	guild.Post("/channels", s.createSetupChannel)
	guild.Post("/roles", s.createRole)
	guild.Get("/reaction-roles", s.reactionRoles)
	guild.Put("/reaction-roles/channel", s.setReactionChannel)
	guild.Post("/reaction-roles", s.createReactionRole)
	guild.Delete("/reaction-roles/:reactionID", s.deleteReactionRole)
	guild.Get("/commands", s.commands)
	guild.Post("/commands", s.saveCommand)
	guild.Delete("/commands/:name", s.deleteCommand)
	guild.Get("/members", s.searchMembers)

	if info, err := os.Stat(s.cfg.WebDir); err == nil && info.IsDir() {
		handler := filesystem.New(filesystem.Config{
			Root:         http.Dir(s.cfg.WebDir),
			Browse:       false,
			Index:        "200.html",
			NotFoundFile: "200.html",
			MaxAge:       3600,
		})
		s.app.Use(func(c *fiber.Ctx) error {
			path := c.Path()
			if strings.HasPrefix(path, "/api") || strings.HasPrefix(path, "/auth") || path == "/healthz" || path == "/readyz" {
				return c.Next()
			}
			if c.Method() != fiber.MethodGet && c.Method() != fiber.MethodHead {
				return c.SendStatus(fiber.StatusNotFound)
			}
			return handler(c)
		})
	}
}

func (s *Server) ready(c *fiber.Ctx) error {
	if s.store == nil || s.bot == nil || !s.bot.Ready() {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"ok": false})
	}
	if err := s.store.Ping(c.UserContext()); err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"ok": false})
	}
	return c.JSON(fiber.Map{"ok": true})
}

func fail(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(fiber.Map{"error": message})
}

func contentSecurityPolicy(webDir string) string {
	policy := "default-src 'self'; img-src 'self' https://cdn.discordapp.com data:; style-src 'self' 'unsafe-inline'; font-src 'self'; connect-src 'self'; script-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'"
	body, err := os.ReadFile(filepath.Join(webDir, "200.html"))
	if err != nil {
		return policy
	}
	var hashes []string
	for _, script := range inlineScripts(body) {
		sum := sha256.Sum256(script)
		hashes = append(hashes, "'sha256-"+base64.StdEncoding.EncodeToString(sum[:])+"'")
	}
	if len(hashes) == 0 {
		return policy
	}
	return strings.Replace(policy, "script-src 'self'", "script-src 'self' "+strings.Join(hashes, " "), 1)
}

func inlineScripts(html []byte) [][]byte {
	const open = "<script>"
	const close = "</script>"
	var scripts [][]byte
	rest := html
	for {
		start := strings.Index(string(rest), open)
		if start < 0 {
			return scripts
		}
		rest = rest[start+len(open):]
		end := strings.Index(string(rest), close)
		if end < 0 {
			return scripts
		}
		scripts = append(scripts, rest[:end])
		rest = rest[end+len(close):]
	}
}
