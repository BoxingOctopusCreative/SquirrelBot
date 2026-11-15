package discordmock

import (
	"encoding/json"
	"net/http"
	"sort"
	"time"
)

func (s *Server) guildRoute(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 0 {
		s.miss(w, r)
		return
	}
	guildID := parts[0]
	if len(parts) == 1 && r.Method == http.MethodGet {
		body, ok := s.guildBody(guildID)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{"message": "Unknown Guild", "code": 10004})
			return
		}
		writeJSON(w, http.StatusOK, body)
		return
	}
	if len(parts) < 2 {
		s.miss(w, r)
		return
	}
	switch parts[1] {
	case "channels":
		s.guildChannels(w, r, guildID)
	case "roles":
		s.guildRoles(w, r, guildID)
	case "emojis":
		s.guildEmojis(w, r, guildID)
	case "members":
		s.memberRoute(w, r, guildID, parts[2:])
	case "bans":
		s.banRoute(w, r, guildID, parts[2:])
	default:
		s.miss(w, r)
	}
}

func (s *Server) guildChannels(w http.ResponseWriter, r *http.Request, guildID string) {
	if !s.hasGuild(guildID) {
		writeJSON(w, http.StatusNotFound, map[string]any{"message": "Unknown Guild", "code": 10004})
		return
	}
	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, s.channelList(guildID))
		return
	}
	if r.Method == http.MethodPost {
		var body struct {
			Name string `json:"name"`
			Type int    `json:"type"`
		}
		_ = readJSON(r, &body)
		id := s.allocID()
		s.mu.Lock()
		s.channels[id] = &channelState{id: id, guildID: guildID, name: body.Name, kind: body.Type}
		item := s.channels[id].json()
		s.mu.Unlock()
		writeJSON(w, http.StatusOK, item)
		return
	}
	s.miss(w, r)
}

func (s *Server) guildRoles(w http.ResponseWriter, r *http.Request, guildID string) {
	if !s.hasGuild(guildID) {
		writeJSON(w, http.StatusNotFound, map[string]any{"message": "Unknown Guild", "code": 10004})
		return
	}
	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, s.roleList(guildID))
		return
	}
	if r.Method == http.MethodPost {
		var body struct {
			Name string `json:"name"`
		}
		_ = readJSON(r, &body)
		id := s.allocID()
		s.mu.Lock()
		guild := s.guilds[guildID]
		guild.roles[id] = &roleState{id: id, name: body.Name, position: 15, permissions: "0"}
		item := guild.roles[id].json()
		s.mu.Unlock()
		writeJSON(w, http.StatusOK, item)
		return
	}
	s.miss(w, r)
}

func (s *Server) guildEmojis(w http.ResponseWriter, r *http.Request, guildID string) {
	if r.Method != http.MethodGet {
		s.miss(w, r)
		return
	}
	s.mu.Lock()
	guild := s.guilds[guildID]
	if guild == nil {
		s.mu.Unlock()
		writeJSON(w, http.StatusNotFound, map[string]any{"message": "Unknown Guild", "code": 10004})
		return
	}
	out := make([]any, 0, len(guild.emojis))
	for _, emoji := range guild.emojis {
		out = append(out, map[string]any{
			"id": emoji.id, "name": emoji.name, "animated": emoji.animated, "roles": []string{},
		})
	}
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) memberRoute(w http.ResponseWriter, r *http.Request, guildID string, parts []string) {
	if !s.hasGuild(guildID) {
		writeJSON(w, http.StatusNotFound, map[string]any{"message": "Unknown Guild", "code": 10004})
		return
	}
	if len(parts) == 0 && r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, s.memberPage(guildID, r.URL.Query().Get("after")))
		return
	}
	if len(parts) == 0 {
		s.miss(w, r)
		return
	}
	userID := parts[0]
	if userID == "@me" {
		userID = s.botMemberID(guildID)
	}
	if len(parts) == 1 && r.Method == http.MethodDelete {
		s.mu.Lock()
		guild := s.guilds[guildID]
		if guild.members[userID] == nil {
			s.mu.Unlock()
			writeJSON(w, http.StatusNotFound, map[string]any{"message": "Unknown Member", "code": 10007})
			return
		}
		delete(guild.members, userID)
		s.mu.Unlock()
		writeJSON(w, http.StatusNoContent, nil)
		return
	}
	if len(parts) == 1 && (r.Method == http.MethodGet || r.Method == http.MethodPatch) {
		s.memberReadOrPatch(w, r, guildID, userID)
		return
	}
	if len(parts) == 3 && parts[1] == "roles" && (r.Method == http.MethodPut || r.Method == http.MethodDelete) {
		s.memberRole(w, r, guildID, userID, parts[2])
		return
	}
	s.miss(w, r)
}

func (s *Server) memberReadOrPatch(w http.ResponseWriter, r *http.Request, guildID, userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	member := s.guilds[guildID].members[userID]
	if member == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"message": "Unknown Member", "code": 10007})
		return
	}
	if r.Method == http.MethodPatch {
		var raw map[string]json.RawMessage
		if readJSON(r, &raw) {
			if nick, ok := raw["nick"]; ok {
				if string(nick) == "null" {
					member.nick = ""
				} else {
					_ = json.Unmarshal(nick, &member.nick)
				}
			}
			if until, ok := raw["communication_disabled_until"]; ok {
				if string(until) == "null" {
					member.timeout = nil
				} else {
					var at time.Time
					if json.Unmarshal(until, &at) == nil {
						member.timeout = &at
					}
				}
			}
		}
	}
	writeJSON(w, http.StatusOK, member.json())
}

func (s *Server) memberRole(w http.ResponseWriter, r *http.Request, guildID, userID, roleID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	member := s.guilds[guildID].members[userID]
	if member == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"message": "Unknown Member", "code": 10007})
		return
	}
	if r.Method == http.MethodPut {
		if !contains(member.roles, roleID) {
			member.roles = append(member.roles, roleID)
		}
	} else {
		next := make([]string, 0, len(member.roles))
		for _, id := range member.roles {
			if id != roleID {
				next = append(next, id)
			}
		}
		member.roles = next
	}
	writeJSON(w, http.StatusNoContent, nil)
}

func (s *Server) banRoute(w http.ResponseWriter, r *http.Request, guildID string, parts []string) {
	if !s.hasGuild(guildID) {
		writeJSON(w, http.StatusNotFound, map[string]any{"message": "Unknown Guild", "code": 10004})
		return
	}
	if len(parts) != 1 || r.Method != http.MethodPut {
		s.miss(w, r)
		return
	}
	userID := parts[0]
	s.mu.Lock()
	delete(s.guilds[guildID].members, userID)
	s.mu.Unlock()
	writeJSON(w, http.StatusNoContent, nil)
}

func (s *Server) botMemberID(guildID string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	guild := s.guilds[guildID]
	if guild == nil {
		return ""
	}
	for _, member := range guild.members {
		if member.bot {
			return member.id
		}
	}
	return ""
}

func (s *Server) hasGuild(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.guilds[id] != nil
}

func (s *Server) guildBody(id string) (map[string]any, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	guild := s.guilds[id]
	if guild == nil {
		return nil, false
	}
	return map[string]any{
		"id": guild.id, "name": guild.name, "owner_id": guild.ownerID, "roles": guild.roleJSON(),
	}, true
}

func (s *Server) channelList(guildID string) []any {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []any{}
	for _, channel := range s.channels {
		if channel.guildID == guildID {
			out = append(out, channel.json())
		}
	}
	return out
}

func (s *Server) roleList(guildID string) []any {
	s.mu.Lock()
	defer s.mu.Unlock()
	guild := s.guilds[guildID]
	if guild == nil {
		return []any{}
	}
	return guild.roleJSON()
}

func (g *guildState) roleJSON() []any {
	out := make([]any, 0, len(g.roles))
	for _, role := range g.roles {
		out = append(out, role.json())
	}
	return out
}

func (role *roleState) json() map[string]any {
	return map[string]any{
		"id": role.id, "name": role.name, "position": role.position,
		"managed": role.managed, "permissions": role.permissions,
	}
}

func (s *Server) memberPage(guildID, after string) []any {
	s.mu.Lock()
	defer s.mu.Unlock()
	guild := s.guilds[guildID]
	ids := make([]string, 0, len(guild.members))
	for id := range guild.members {
		if after == "" || id > after {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	out := make([]any, 0, len(ids))
	for _, id := range ids {
		out = append(out, guild.members[id].json())
	}
	return out
}

func (m *memberState) json() map[string]any {
	roles := m.roles
	if roles == nil {
		roles = []string{}
	}
	body := map[string]any{
		"nick": m.nick, "roles": roles,
		"user": map[string]any{
			"id": m.id, "username": m.username, "global_name": m.globalName, "bot": m.bot,
		},
	}
	if m.timeout != nil {
		body["communication_disabled_until"] = m.timeout.UTC().Format(time.RFC3339Nano)
	}
	return body
}

func contains(ids []string, id string) bool {
	for _, item := range ids {
		if item == id {
			return true
		}
	}
	return false
}
