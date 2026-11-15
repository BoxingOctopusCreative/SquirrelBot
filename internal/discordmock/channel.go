package discordmock

import (
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"time"
)

func (s *Server) channelRoute(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 0 {
		s.miss(w, r)
		return
	}
	channelID := parts[0]
	if len(parts) == 1 {
		s.oneChannel(w, r, channelID)
		return
	}
	if parts[1] != "messages" {
		s.miss(w, r)
		return
	}
	if len(parts) == 2 && r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, s.channelMessages(channelID, r.URL.Query()))
		return
	}
	if len(parts) == 2 && r.Method == http.MethodPost {
		s.createMessage(w, r, channelID)
		return
	}
	if len(parts) == 3 && parts[2] == "bulk-delete" && r.Method == http.MethodPost {
		s.bulkDelete(w, r, channelID)
		return
	}
	if len(parts) < 3 {
		s.miss(w, r)
		return
	}
	messageID := parts[2]
	if len(parts) == 3 {
		s.oneMessage(w, r, channelID, messageID)
		return
	}
	if parts[3] == "reactions" {
		s.mu.Lock()
		_, ok := s.messages[messageID]
		s.mu.Unlock()
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{"message": "Unknown Message", "code": 10008})
			return
		}
		writeJSON(w, http.StatusNoContent, nil)
		return
	}
	s.miss(w, r)
}

func (s *Server) oneChannel(w http.ResponseWriter, r *http.Request, channelID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	channel := s.channels[channelID]
	if channel == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"message": "Unknown Channel", "code": 10003})
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, channel.json())
	case http.MethodPatch:
		var body struct {
			Name             string `json:"name"`
			RateLimitPerUser *int   `json:"rate_limit_per_user"`
		}
		if readJSON(r, &body) {
			if body.Name != "" {
				channel.name = body.Name
			}
			if body.RateLimitPerUser != nil {
				channel.rateLimit = *body.RateLimitPerUser
			}
		}
		writeJSON(w, http.StatusOK, channel.json())
	case http.MethodDelete:
		item := channel.json()
		delete(s.channels, channelID)
		writeJSON(w, http.StatusOK, item)
	default:
		s.miss(w, r)
	}
}

func (s *Server) createMessage(w http.ResponseWriter, r *http.Request, channelID string) {
	var body struct {
		Content string `json:"content"`
		Embeds  []struct {
			Title       string `json:"title"`
			Description string `json:"description"`
		} `json:"embeds"`
	}
	if !readJSON(r, &body) {
		body.Content = ""
	}
	text := body.Content
	if text == "" {
		for _, embed := range body.Embeds {
			if embed.Title != "" {
				if text != "" {
					text += "\n"
				}
				text += embed.Title
			}
			if embed.Description != "" {
				if text != "" {
					text += "\n"
				}
				text += embed.Description
			}
		}
	}
	id := s.allocID()
	s.mu.Lock()
	s.messages[id] = &messageState{
		id: id, channelID: channelID, content: text, timestamp: time.Now().UTC(),
	}
	item := s.messages[id].json()
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) oneMessage(w http.ResponseWriter, r *http.Request, channelID, messageID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item := s.messages[messageID]
	if item == nil || item.channelID != channelID {
		writeJSON(w, http.StatusNotFound, map[string]any{"message": "Unknown Message", "code": 10008})
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, item.json())
	case http.MethodPatch:
		var body struct {
			Content *string `json:"content"`
		}
		if readJSON(r, &body) && body.Content != nil {
			item.content = *body.Content
		}
		writeJSON(w, http.StatusOK, item.json())
	case http.MethodDelete:
		delete(s.messages, messageID)
		writeJSON(w, http.StatusNoContent, nil)
	default:
		s.miss(w, r)
	}
}

func (s *Server) bulkDelete(w http.ResponseWriter, r *http.Request, channelID string) {
	var body struct {
		Messages []string `json:"messages"`
	}
	_ = readJSON(r, &body)
	s.mu.Lock()
	for _, id := range body.Messages {
		item := s.messages[id]
		if item != nil && item.channelID == channelID {
			delete(s.messages, id)
		}
	}
	s.mu.Unlock()
	writeJSON(w, http.StatusNoContent, nil)
}

func (s *Server) channelMessages(channelID string, query url.Values) []any {
	limit := 50
	if raw := query.Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > 100 {
		limit = 100
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	items := make([]*messageState, 0)
	for _, item := range s.messages {
		if item.channelID == channelID {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].id > items[j].id })
	if len(items) > limit {
		items = items[:limit]
	}
	out := make([]any, 0, len(items))
	for _, item := range items {
		out = append(out, item.json())
	}
	return out
}

func (s *Server) messageCopies(channelID string) []Message {
	items := make([]*messageState, 0)
	for _, item := range s.messages {
		if item.channelID == channelID {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].id > items[j].id })
	out := make([]Message, 0, len(items))
	for _, item := range items {
		out = append(out, Message{ID: item.id, ChannelID: item.channelID, AuthorID: item.authorID, Content: item.content})
	}
	return out
}

func (s *Server) userRoute(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 2 && parts[0] == "@me" && parts[1] == "channels" && r.Method == http.MethodPost {
		id := s.allocID()
		s.mu.Lock()
		s.channels[id] = &channelState{id: id, name: "dm", kind: 1}
		item := s.channels[id].json()
		s.mu.Unlock()
		writeJSON(w, http.StatusOK, item)
		return
	}
	s.miss(w, r)
}

func (c *channelState) json() map[string]any {
	return map[string]any{
		"id": c.id, "guild_id": c.guildID, "name": c.name, "type": c.kind,
		"rate_limit_per_user": c.rateLimit,
	}
}

func (m *messageState) json() map[string]any {
	return map[string]any{
		"id": m.id, "channel_id": m.channelID, "content": m.content,
		"timestamp": m.timestamp.Format(time.RFC3339Nano),
		"author": map[string]any{
			"id": m.authorID, "username": "member", "bot": false,
		},
	}
}
