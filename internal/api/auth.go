package api

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/BoxingOctopus/squirrelbot/internal/domain"
	"github.com/BoxingOctopus/squirrelbot/internal/secretbox"
	"github.com/BoxingOctopus/squirrelbot/internal/store"
	"github.com/gofiber/fiber/v2"
)

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Error        string `json:"error"`
}

type discordUser struct {
	ID         string `json:"id"`
	Username   string `json:"username"`
	GlobalName string `json:"global_name"`
	Avatar     string `json:"avatar"`
}

type discordGuild struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Icon        string        `json:"icon"`
	Owner       bool          `json:"owner"`
	Permissions jsonIntString `json:"permissions"`
}

// jsonIntString reads a JSON string or number as text.
// Discord sends guild permission bits as a number in some responses and a string in others.
type jsonIntString string

func (v *jsonIntString) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*v = ""
		return nil
	}
	if data[0] == '"' {
		var text string
		if err := json.Unmarshal(data, &text); err != nil {
			return err
		}
		*v = jsonIntString(text)
		return nil
	}
	if data[0] == '-' || (data[0] >= '0' && data[0] <= '9') {
		*v = jsonIntString(data)
		return nil
	}
	return errors.New("expected a string or number")
}

const botInvitePermissions = "1099847265366"

const oauthScopes = "identify guilds bot applications.commands"

func discordAuthorizeURL(clientID, redirectURI, state string) string {
	query := url.Values{}
	query.Set("client_id", clientID)
	query.Set("redirect_uri", redirectURI)
	query.Set("response_type", "code")
	query.Set("permissions", botInvitePermissions)
	query.Set("integration_type", "0")
	query.Set("state", state)
	query.Set("prompt", "consent")
	// Discord's authorize page splits scopes on %20. url.Values encodes a space as +.
	scope := strings.ReplaceAll(url.QueryEscape(oauthScopes), "+", "%20")
	return "https://discord.com/oauth2/authorize?" + query.Encode() + "&scope=" + scope
}

func (s *Server) startLogin(c *fiber.Ctx) error {
	state, err := randomID()
	if err != nil {
		return fail(c, fiber.StatusInternalServerError, "Could not start sign-in.")
	}
	s.setCookie(c, stateCookie, state, time.Now().Add(10*time.Minute))
	return c.Redirect(discordAuthorizeURL(s.cfg.DiscordClientID, s.cfg.PublicURL+"/auth/discord/callback", state), fiber.StatusFound)
}

func (s *Server) finishLogin(c *fiber.Ctx) error {
	if c.Query("error") != "" {
		return c.Redirect("/?error=denied", fiber.StatusFound)
	}
	if c.Cookies(stateCookie) == "" || c.Cookies(stateCookie) != c.Query("state") || c.Query("code") == "" {
		return c.Redirect("/?error=state", fiber.StatusFound)
	}
	s.clearCookie(c, stateCookie)
	tok, err := s.exchange(c.UserContext(), url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {c.Query("code")},
		"redirect_uri": {s.cfg.PublicURL + "/auth/discord/callback"},
	})
	if err != nil {
		s.log.Warn("oauth exchange", "err", err)
		return c.Redirect("/?error=oauth", fiber.StatusFound)
	}
	user, err := s.discordMe(c.UserContext(), tok.AccessToken)
	if err != nil {
		return c.Redirect("/?error=oauth", fiber.StatusFound)
	}
	session, err := s.newSession(user, tok)
	if err != nil {
		return c.Redirect("/?error=oauth", fiber.StatusFound)
	}
	if err := s.store.SaveSession(c.UserContext(), session); err != nil {
		s.log.Error("save session", "err", err)
		return c.Redirect("/?error=oauth", fiber.StatusFound)
	}
	s.setCookie(c, sessionCookie, session.ID, session.ExpiresAt)
	if guildID := c.Query("guild_id"); s.bot != nil && domain.ValidSnowflake(guildID) {
		if err := s.bot.RememberGuild(c.UserContext(), guildID); err != nil {
			s.log.Warn("invited guild", "err", err)
		}
	}
	return c.Redirect("/app", fiber.StatusFound)
}

func (s *Server) logout(c *fiber.Ctx) error {
	if id := c.Cookies(sessionCookie); id != "" {
		_ = s.store.DeleteSession(c.UserContext(), id)
		s.access.drop(id)
	}
	s.clearCookie(c, sessionCookie)
	return c.SendStatus(fiber.StatusNoContent)
}

func (s *Server) exchange(ctx context.Context, values url.Values) (tokenResponse, error) {
	values.Set("client_id", s.cfg.DiscordClientID)
	values.Set("client_secret", s.cfg.DiscordClientSecret)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://discord.com/api/oauth2/token", strings.NewReader(values.Encode()))
	if err != nil {
		return tokenResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "SquirrelBot (https://github.com/BoxingOctopus/squirrelbot)")
	res, err := s.client.Do(req)
	if err != nil {
		return tokenResponse{}, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return tokenResponse{}, err
	}
	var tok tokenResponse
	if err := json.Unmarshal(body, &tok); err != nil {
		return tokenResponse{}, err
	}
	if res.StatusCode >= 300 || tok.AccessToken == "" {
		return tokenResponse{}, fmt.Errorf("discord token status %d", res.StatusCode)
	}
	return tok, nil
}

func (s *Server) discordMe(ctx context.Context, access string) (discordUser, error) {
	var user discordUser
	err := s.discordGet(ctx, access, "https://discord.com/api/users/@me", &user)
	return user, err
}

func (s *Server) discordGuilds(ctx context.Context, access string) ([]discordGuild, error) {
	var guilds []discordGuild
	err := s.discordGet(ctx, access, "https://discord.com/api/users/@me/guilds", &guilds)
	return guilds, err
}

func (s *Server) discordGet(ctx context.Context, access, endpoint string, dest any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+access)
	req.Header.Set("User-Agent", "SquirrelBot (https://github.com/BoxingOctopus/squirrelbot)")
	res, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	if err != nil {
		return err
	}
	if res.StatusCode == http.StatusUnauthorized {
		return errUnauthorized
	}
	if res.StatusCode >= 300 {
		return fmt.Errorf("discord status %d", res.StatusCode)
	}
	return json.Unmarshal(body, dest)
}

var errUnauthorized = errors.New("unauthorized")

func (s *Server) newSession(user discordUser, tok tokenResponse) (store.Session, error) {
	id, err := randomID()
	if err != nil {
		return store.Session{}, err
	}
	access, err := secretbox.Seal(s.key, tok.AccessToken)
	if err != nil {
		return store.Session{}, err
	}
	refresh, err := secretbox.Seal(s.key, tok.RefreshToken)
	if err != nil {
		return store.Session{}, err
	}
	name := user.GlobalName
	if name == "" {
		name = user.Username
	}
	expires := 7 * 24 * time.Hour
	if tok.ExpiresIn > 0 {
		expires = time.Duration(tok.ExpiresIn) * time.Second
	}
	return store.Session{
		ID:             id,
		UserID:         user.ID,
		Username:       name,
		Avatar:         user.Avatar,
		AccessToken:    access,
		RefreshToken:   refresh,
		TokenExpiresAt: time.Now().Add(expires),
		ExpiresAt:      time.Now().Add(14 * 24 * time.Hour),
	}, nil
}

func (s *Server) requireSession(c *fiber.Ctx) error {
	session, err := s.currentSession(c)
	if err != nil {
		return fail(c, fiber.StatusUnauthorized, "Sign in again.")
	}
	c.Locals("session", session)
	return c.Next()
}

func (s *Server) currentSession(c *fiber.Ctx) (store.Session, error) {
	id := c.Cookies(sessionCookie)
	if id == "" {
		return store.Session{}, errUnauthorized
	}
	session, err := s.store.GetSession(c.UserContext(), id)
	if err != nil {
		return store.Session{}, errUnauthorized
	}
	access, err := secretbox.Open(s.key, session.AccessToken)
	if err != nil {
		return store.Session{}, errUnauthorized
	}
	refresh := ""
	if session.RefreshToken != "" {
		refresh, err = secretbox.Open(s.key, session.RefreshToken)
		if err != nil {
			return store.Session{}, errUnauthorized
		}
	}
	if time.Until(session.TokenExpiresAt) < 2*time.Minute && refresh != "" {
		tok, err := s.exchange(c.UserContext(), url.Values{
			"grant_type":    {"refresh_token"},
			"refresh_token": {refresh},
		})
		if err != nil {
			_ = s.store.DeleteSession(c.UserContext(), session.ID)
			return store.Session{}, errUnauthorized
		}
		if tok.RefreshToken == "" {
			tok.RefreshToken = refresh
		}
		updated, err := s.newSession(discordUser{ID: session.UserID, Username: session.Username, Avatar: session.Avatar}, tok)
		if err != nil {
			return store.Session{}, err
		}
		updated.ID = session.ID
		updated.ExpiresAt = session.ExpiresAt
		if err := s.store.SaveSession(c.UserContext(), updated); err != nil {
			return store.Session{}, err
		}
		session = updated
		access = tok.AccessToken
	}
	session.AccessToken = access
	session.RefreshToken = refresh
	return session, nil
}

func (s *Server) requireSameOrigin(c *fiber.Ctx) error {
	switch c.Method() {
	case fiber.MethodGet, fiber.MethodHead, fiber.MethodOptions:
		return c.Next()
	}
	if c.Get("Origin") != s.cfg.Origin() {
		return fail(c, fiber.StatusForbidden, "Cross-site request blocked.")
	}
	return c.Next()
}

func (s *Server) requireGuild(c *fiber.Ctx) error {
	guildID := c.Params("guildID")
	if !domain.ValidSnowflake(guildID) {
		return fail(c, fiber.StatusNotFound, "Unknown server.")
	}
	session := c.Locals("session").(store.Session)
	perms, ok := s.permissions(c, session, guildID)
	if !ok || !domain.CanManageGuild(perms) {
		return fail(c, fiber.StatusForbidden, "You need Manage Server on that Discord server.")
	}
	if _, err := s.store.GetGuild(c.UserContext(), guildID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return fail(c, fiber.StatusNotFound, "SquirrelBot is not in that server.")
		}
		s.log.Error("load guild", "err", err)
		return fail(c, fiber.StatusInternalServerError, "Could not load that server.")
	}
	c.Locals("guildID", guildID)
	return c.Next()
}

func (s *Server) permissions(c *fiber.Ctx, session store.Session, guildID string) (int64, bool) {
	guilds, ok := s.access.get(session.ID)
	if !ok {
		list, err := s.discordGuilds(c.UserContext(), session.AccessToken)
		if err != nil {
			return 0, false
		}
		guilds = map[string]int64{}
		for _, guild := range list {
			bits, _ := strconv.ParseInt(string(guild.Permissions), 10, 64)
			if guild.Owner {
				bits |= domain.PermAdministrator
			}
			guilds[guild.ID] = bits
		}
		s.access.set(session.ID, guilds)
	}
	bits, found := guilds[guildID]
	return bits, found
}

func (s *Server) me(c *fiber.Ctx) error {
	session := c.Locals("session").(store.Session)
	return c.JSON(fiber.Map{
		"id":        session.UserID,
		"username":  session.Username,
		"avatarUrl": avatarURL(session.UserID, session.Avatar),
	})
}

func (s *Server) guilds(c *fiber.Ctx) error {
	session := c.Locals("session").(store.Session)
	list, err := s.discordGuilds(c.UserContext(), session.AccessToken)
	if err != nil {
		if errors.Is(err, errUnauthorized) {
			return fail(c, fiber.StatusUnauthorized, "Sign in again.")
		}
		s.log.Warn("discord guilds", "err", err)
		return fail(c, fiber.StatusBadGateway, "Discord did not return your servers.")
	}
	type item struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		IconURL string `json:"iconUrl"`
	}
	out := []item{}
	cached := map[string]int64{}
	for _, guild := range list {
		bits, _ := strconv.ParseInt(string(guild.Permissions), 10, 64)
		if guild.Owner {
			bits |= domain.PermAdministrator
		}
		cached[guild.ID] = bits
		if !domain.CanManageGuild(bits) {
			continue
		}
		if _, err := s.store.GetGuild(c.UserContext(), guild.ID); err != nil {
			continue
		}
		out = append(out, item{ID: guild.ID, Name: guild.Name, IconURL: iconURL(guild.ID, guild.Icon)})
	}
	s.access.set(session.ID, cached)
	return c.JSON(fiber.Map{"guilds": out})
}

func avatarURL(userID, hash string) string {
	if hash == "" || !domain.ValidSnowflake(userID) {
		return ""
	}
	return "https://cdn.discordapp.com/avatars/" + userID + "/" + hash + ".png"
}

func iconURL(guildID, hash string) string {
	if hash == "" || !domain.ValidSnowflake(guildID) {
		return ""
	}
	return "https://cdn.discordapp.com/icons/" + guildID + "/" + hash + ".png"
}

func (s *Server) setCookie(c *fiber.Ctx, name, value string, expires time.Time) {
	c.Cookie(&fiber.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		Expires:  expires,
		HTTPOnly: true,
		Secure:   s.cfg.SecureCookies,
		SameSite: "Lax",
	})
}

func (s *Server) clearCookie(c *fiber.Ctx, name string) {
	c.Cookie(&fiber.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HTTPOnly: true,
		Secure:   s.cfg.SecureCookies,
		SameSite: "Lax",
	})
}

func randomID() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

type accessCache struct {
	mu    sync.Mutex
	items map[string]accessEntry
}

type accessEntry struct {
	guilds map[string]int64
	exp    time.Time
}

func newAccessCache() *accessCache {
	return &accessCache{items: map[string]accessEntry{}}
}

func (c *accessCache) get(sessionID string) (map[string]int64, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.items[sessionID]
	if !ok || time.Now().After(entry.exp) {
		return nil, false
	}
	return entry.guilds, true
}

func (c *accessCache) set(sessionID string, guilds map[string]int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.items) > 500 {
		c.items = map[string]accessEntry{}
	}
	c.items[sessionID] = accessEntry{guilds: guilds, exp: time.Now().Add(45 * time.Second)}
}

func (c *accessCache) drop(sessionID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, sessionID)
}
