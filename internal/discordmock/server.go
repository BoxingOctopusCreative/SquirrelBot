// Package discordmock is a Discord REST stand-in for tests.
// Point a discordgo session's HTTP client at it with NewTransport.
package discordmock

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// User is a Discord account the OAuth routes can return.
type User struct {
	ID         string
	Username   string
	GlobalName string
}

// UserGuild is one server listed by users/@me/guilds.
type UserGuild struct {
	ID          string
	Name        string
	Icon        string
	Owner       bool
	Permissions string
}

// Member is a guild member stored by the mock.
type Member struct {
	ID         string
	Username   string
	GlobalName string
	Nick       string
	Roles      []string
	Bot        bool
	Timeout    *time.Time
}

// Channel is a guild or DM channel stored by the mock.
type Channel struct {
	ID               string
	GuildID          string
	Name             string
	Type             int
	RateLimitPerUser int
}

// Message is a channel message stored by the mock.
type Message struct {
	ID        string
	ChannelID string
	AuthorID  string
	Content   string
}

// Server answers the Discord REST and OAuth calls the bot and dashboard make.
type Server struct {
	mu           sync.Mutex
	access       string
	refresh      string
	tokens       map[string]tokenUser
	guilds       map[string]*guildState
	channels     map[string]*channelState
	messages     map[string]*messageState
	interactions map[string]*interactionState
	tokensByID   map[string]string
	nextID       int64
}

type tokenUser struct {
	user   User
	guilds []UserGuild
}

type guildState struct {
	id      string
	name    string
	ownerID string
	roles   map[string]*roleState
	members map[string]*memberState
	emojis  []emojiState
}

type roleState struct {
	id          string
	name        string
	position    int
	managed     bool
	permissions string
}

type memberState struct {
	id         string
	username   string
	globalName string
	nick       string
	roles      []string
	bot        bool
	timeout    *time.Time
}

type channelState struct {
	id        string
	guildID   string
	name      string
	kind      int
	rateLimit int
}

type messageState struct {
	id        string
	channelID string
	authorID  string
	content   string
	timestamp time.Time
}

type interactionState struct {
	id      string
	content string
	edits   []string
}

type emojiState struct {
	id       string
	name     string
	animated bool
}

// New returns an empty Discord stand-in.
func New() *Server {
	return &Server{
		tokens:       map[string]tokenUser{},
		guilds:       map[string]*guildState{},
		channels:     map[string]*channelState{},
		messages:     map[string]*messageState{},
		interactions: map[string]*interactionState{},
		tokensByID:   map[string]string{},
		nextID:       130000000000000000,
	}
}

// SetOAuth chooses the tokens returned by the authorization-code exchange.
func (s *Server) SetOAuth(access, refresh string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.access = access
	s.refresh = refresh
}

// AddToken registers a bearer token for users/@me and users/@me/guilds.
func (s *Server) AddToken(access string, user User, guilds []UserGuild) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[access] = tokenUser{user: user, guilds: append([]UserGuild{}, guilds...)}
}

// AddGuild stores a server. Later calls replace the same id.
func (s *Server) AddGuild(id, name, ownerID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.guilds[id] = &guildState{
		id: id, name: name, ownerID: ownerID,
		roles: map[string]*roleState{}, members: map[string]*memberState{},
	}
}

// AddRole stores a role on a guild that AddGuild already created.
func (s *Server) AddRole(guildID, id, name string, position int, managed bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	guild := s.guilds[guildID]
	if guild == nil {
		return
	}
	guild.roles[id] = &roleState{id: id, name: name, position: position, managed: managed, permissions: "0"}
}

// AddChannel stores a channel and returns its id.
func (s *Server) AddChannel(guildID, id, name string, kind int) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id == "" {
		id = s.allocID()
	}
	s.channels[id] = &channelState{id: id, guildID: guildID, name: name, kind: kind}
	return id
}

// AddMember stores a guild member.
func (s *Server) AddMember(guildID string, member Member) {
	s.mu.Lock()
	defer s.mu.Unlock()
	guild := s.guilds[guildID]
	if guild == nil {
		return
	}
	roles := append([]string{}, member.Roles...)
	var timeout *time.Time
	if member.Timeout != nil {
		copied := *member.Timeout
		timeout = &copied
	}
	guild.members[member.ID] = &memberState{
		id: member.ID, username: member.Username, globalName: member.GlobalName,
		nick: member.Nick, roles: roles, bot: member.Bot, timeout: timeout,
	}
}

// AddEmoji stores a custom emoji.
func (s *Server) AddEmoji(guildID, id, name string, animated bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	guild := s.guilds[guildID]
	if guild == nil {
		return
	}
	guild.emojis = append(guild.emojis, emojiState{id: id, name: name, animated: animated})
}

// AddMessage stores a message and returns its id.
func (s *Server) AddMessage(channelID, authorID, content string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.allocID()
	s.messages[id] = &messageState{
		id: id, channelID: channelID, authorID: authorID, content: content, timestamp: time.Now().UTC(),
	}
	return id
}

// Member returns a copy of a guild member.
func (s *Server) Member(guildID, userID string) (Member, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	guild := s.guilds[guildID]
	if guild == nil {
		return Member{}, false
	}
	member := guild.members[userID]
	if member == nil {
		return Member{}, false
	}
	return member.copy(), true
}

// Channel returns a copy of a channel.
func (s *Server) Channel(id string) (Channel, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	channel := s.channels[id]
	if channel == nil {
		return Channel{}, false
	}
	return channel.copy(), true
}

// Messages returns the messages in a channel, newest first.
func (s *Server) Messages(channelID string) []Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.messageCopies(channelID)
}

// InteractionContent returns the latest reply for an interaction.
// An edit replaces the original callback text.
func (s *Server) InteractionContent(id string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item := s.interactions[id]
	if item == nil {
		return "", false
	}
	if len(item.edits) > 0 {
		return item.edits[len(item.edits)-1], true
	}
	return item.content, true
}

// ServeHTTP implements the Discord routes the app calls.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 || parts[0] != "api" {
		s.miss(w, r)
		return
	}
	if parts[1] == "v9" || parts[1] == "v10" {
		s.rest(w, r, parts[2:])
		return
	}
	s.oauth(w, r, parts[1:])
}

func (s *Server) oauth(w http.ResponseWriter, r *http.Request, parts []string) {
	switch strings.Join(parts, "/") {
	case "oauth2/token":
		s.mu.Lock()
		access, refresh := s.access, s.refresh
		s.mu.Unlock()
		writeJSON(w, http.StatusOK, map[string]any{
			"access_token":  access,
			"refresh_token": refresh,
			"token_type":    "Bearer",
			"expires_in":    3600,
		})
	case "users/@me":
		user, ok := s.userFor(bearer(r))
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"message": "401: Unauthorized"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"id": user.ID, "username": user.Username, "global_name": user.GlobalName, "avatar": "",
		})
	case "users/@me/guilds":
		token, ok := s.tokenFor(bearer(r))
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"message": "401: Unauthorized"})
			return
		}
		out := make([]any, 0, len(token.guilds))
		for _, guild := range token.guilds {
			out = append(out, map[string]any{
				"id": guild.ID, "name": guild.Name, "icon": guild.Icon,
				"owner": guild.Owner, "permissions": guild.Permissions,
			})
		}
		writeJSON(w, http.StatusOK, out)
	default:
		s.miss(w, r)
	}
}

func (s *Server) rest(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 0 {
		s.miss(w, r)
		return
	}
	switch parts[0] {
	case "guilds":
		s.guildRoute(w, r, parts[1:])
	case "channels":
		s.channelRoute(w, r, parts[1:])
	case "users":
		s.userRoute(w, r, parts[1:])
	case "interactions":
		s.interactionRoute(w, r, parts[1:])
	case "webhooks":
		s.webhookRoute(w, r, parts[1:])
	case "applications":
		s.applicationRoute(w, r, parts[1:])
	default:
		s.miss(w, r)
	}
}

func (s *Server) userFor(access string) (User, bool) {
	token, ok := s.tokenFor(access)
	if !ok {
		return User{}, false
	}
	return token.user, true
}

func (s *Server) tokenFor(access string) (tokenUser, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	token, ok := s.tokens[access]
	return token, ok
}

func (s *Server) allocID() string {
	return strconv.FormatInt(atomic.AddInt64(&s.nextID, 1), 10)
}

func (m memberState) copy() Member {
	var timeout *time.Time
	if m.timeout != nil {
		copied := *m.timeout
		timeout = &copied
	}
	return Member{
		ID: m.id, Username: m.username, GlobalName: m.globalName, Nick: m.nick,
		Roles: append([]string{}, m.roles...), Bot: m.bot, Timeout: timeout,
	}
}

func (c channelState) copy() Channel {
	return Channel{ID: c.id, GuildID: c.guildID, Name: c.name, Type: c.kind, RateLimitPerUser: c.rateLimit}
}

func bearer(r *http.Request) string {
	value := r.Header.Get("Authorization")
	value = strings.TrimPrefix(value, "Bearer ")
	return strings.TrimPrefix(value, "Bot ")
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	if body == nil {
		w.WriteHeader(status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func (s *Server) miss(w http.ResponseWriter, r *http.Request) {
	log.Printf("discord mock: unhandled %s %s", r.Method, r.URL.RequestURI())
	writeJSON(w, http.StatusNotFound, map[string]any{"message": "Unknown", "code": 0})
}

func readJSON(r *http.Request, dest any) bool {
	media := r.Header.Get("Content-Type")
	if media != "" && !strings.Contains(media, "json") {
		_, _ = io.Copy(io.Discard, r.Body)
		return false
	}
	return json.NewDecoder(r.Body).Decode(dest) == nil
}
