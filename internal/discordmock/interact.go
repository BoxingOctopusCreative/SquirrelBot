package discordmock

import (
	"net/http"
	"time"
)

func (s *Server) interactionRoute(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) != 3 || parts[2] != "callback" || r.Method != http.MethodPost {
		s.miss(w, r)
		return
	}
	var body struct {
		Type int `json:"type"`
		Data struct {
			Content string `json:"content"`
		} `json:"data"`
	}
	_ = readJSON(r, &body)
	s.mu.Lock()
	s.interactions[parts[0]] = &interactionState{id: parts[0], content: body.Data.Content}
	s.tokensByID[parts[1]] = parts[0]
	s.mu.Unlock()
	writeJSON(w, http.StatusNoContent, nil)
}

func (s *Server) webhookRoute(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) < 2 {
		s.miss(w, r)
		return
	}
	token := parts[1]
	if len(parts) == 2 && r.Method == http.MethodPost {
		id := s.allocID()
		writeJSON(w, http.StatusOK, (&messageState{id: id, content: "ok", timestamp: time.Now().UTC()}).json())
		return
	}
	if len(parts) == 4 && parts[2] == "messages" && parts[3] == "@original" && r.Method == http.MethodPatch {
		var body struct {
			Content *string `json:"content"`
		}
		_ = readJSON(r, &body)
		text := ""
		if body.Content != nil {
			text = *body.Content
		}
		s.mu.Lock()
		id := s.tokensByID[token]
		item := s.interactions[id]
		if item == nil {
			item = &interactionState{id: id}
			if id != "" {
				s.interactions[id] = item
			}
		}
		if item != nil {
			item.edits = append(item.edits, text)
		}
		s.mu.Unlock()
		messageID := s.allocID()
		writeJSON(w, http.StatusOK, (&messageState{id: messageID, content: text, timestamp: time.Now().UTC()}).json())
		return
	}
	s.miss(w, r)
}

func (s *Server) applicationRoute(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 2 && parts[1] == "commands" && r.Method == http.MethodPut {
		writeJSON(w, http.StatusOK, []any{})
		return
	}
	s.miss(w, r)
}
