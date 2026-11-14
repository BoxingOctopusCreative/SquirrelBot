package discord

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/BoxingOctopus/squirrelbot/internal/domain"
	"github.com/bwmarrin/discordgo"
)

const (
	confessionOpenID   = "sb:confess:open"
	confessionFormID   = "sb:confess:form"
	maxConfessionBytes = 8 << 20
)

type confessionFile struct {
	Name        string
	ContentType string
	Data        []byte
}

type confessionForm struct {
	Text        string
	FileID      string
	FileName    string
	FileURL     string
	FileType    string
	FileSize    int
	MissingFile bool
}

type modalLabel struct {
	Label       string
	Description string
	Component   discordgo.MessageComponent
}

func (modalLabel) Type() discordgo.ComponentType { return 18 }

func (m modalLabel) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type        int                        `json:"type"`
		Label       string                     `json:"label"`
		Description string                     `json:"description,omitempty"`
		Component   discordgo.MessageComponent `json:"component"`
	}{
		Type:        18,
		Label:       m.Label,
		Description: m.Description,
		Component:   m.Component,
	})
}

type modalParagraph struct {
	CustomID    string
	Placeholder string
	MinLength   int
	MaxLength   int
}

func (modalParagraph) Type() discordgo.ComponentType { return discordgo.TextInputComponent }

func (m modalParagraph) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type        int    `json:"type"`
		CustomID    string `json:"custom_id"`
		Style       int    `json:"style"`
		Placeholder string `json:"placeholder,omitempty"`
		Required    bool   `json:"required"`
		MinLength   int    `json:"min_length,omitempty"`
		MaxLength   int    `json:"max_length,omitempty"`
	}{
		Type:        int(discordgo.TextInputComponent),
		CustomID:    m.CustomID,
		Style:       int(discordgo.TextInputParagraph),
		Placeholder: m.Placeholder,
		Required:    true,
		MinLength:   m.MinLength,
		MaxLength:   m.MaxLength,
	})
}

type modalFileUpload struct {
	CustomID string
}

func (modalFileUpload) Type() discordgo.ComponentType { return 19 }

func (m modalFileUpload) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type      int    `json:"type"`
		CustomID  string `json:"custom_id"`
		MinValues int    `json:"min_values"`
		MaxValues int    `json:"max_values"`
		Required  bool   `json:"required"`
	}{
		Type:      19,
		CustomID:  m.CustomID,
		MinValues: 0,
		MaxValues: 1,
		Required:  false,
	})
}

func confessionFormComponents() []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		modalLabel{
			Label: "Confession",
			Component: modalParagraph{
				CustomID:    "text",
				Placeholder: "What do you want to say?",
				MinLength:   1,
				MaxLength:   domain.MaxConfession,
			},
		},
		modalLabel{
			Label:       "File",
			Description: "Optional. One file, up to 8 MB.",
			Component:   modalFileUpload{CustomID: "file"},
		},
	}
}

func submitConfessionButton() []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Submit a Confession",
				Style:    discordgo.PrimaryButton,
				CustomID: confessionOpenID,
			},
		}},
	}
}

func (b *Bot) openConfessionForm(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.GuildID == "" || i.Member == nil || i.Member.User == nil || i.Member.User.Bot {
		_ = respondEphemeral(s, i, "Use this button in a server.")
		return
	}
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID:   confessionFormID,
			Title:      "Submit a confession",
			Components: confessionFormComponents(),
		},
	})
	if err != nil {
		b.log.Warn("confession form", "guild", i.GuildID, "err", err)
	}
}

func (b *Bot) onGatewayEvent(s *discordgo.Session, event *discordgo.Event) {
	if event == nil || event.Type != "INTERACTION_CREATE" || len(event.RawData) == 0 {
		return
	}
	interaction, form, ok := confessionFormFromEvent(event.RawData)
	if !ok {
		return
	}
	b.submitConfessionForm(s, interaction, form)
}

func (b *Bot) submitConfessionForm(s *discordgo.Session, interaction *discordgo.Interaction, form confessionForm) {
	i := &discordgo.InteractionCreate{Interaction: interaction}
	if form.MissingFile {
		_ = respondEphemeral(s, i, "I could not read that file.")
		return
	}
	body, err := domain.CleanConfession(form.Text)
	if err != nil {
		_ = respondEphemeral(s, i, err.Error())
		return
	}
	if !b.beginConfession(s, i) {
		return
	}
	var file *confessionFile
	if form.FileURL != "" {
		downloaded, err := b.downloadConfessionFile(fileRef{
			URL: form.FileURL, Name: form.FileName, ContentType: form.FileType, Size: form.FileSize, Authorized: true,
		})
		if err != nil {
			editReply(s, i, err.Error())
			return
		}
		file = downloaded
	}
	b.postConfession(s, i, body, file)
}

func confessionFormFromEvent(raw []byte) (*discordgo.Interaction, confessionForm, bool) {
	var payload struct {
		ID        string                    `json:"id"`
		AppID     string                    `json:"application_id"`
		Type      discordgo.InteractionType `json:"type"`
		GuildID   string                    `json:"guild_id"`
		ChannelID string                    `json:"channel_id"`
		Token     string                    `json:"token"`
		Member    *discordgo.Member         `json:"member"`
		User      *discordgo.User           `json:"user"`
		Data      struct {
			CustomID   string          `json:"custom_id"`
			Components json.RawMessage `json:"components"`
			Resolved   struct {
				Attachments map[string]struct {
					Filename    string `json:"filename"`
					URL         string `json:"url"`
					ContentType string `json:"content_type"`
					Size        int    `json:"size"`
				} `json:"attachments"`
			} `json:"resolved"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, confessionForm{}, false
	}
	if payload.Type != discordgo.InteractionModalSubmit || payload.Data.CustomID != confessionFormID {
		return nil, confessionForm{}, false
	}
	var form confessionForm
	collectConfessionFields(payload.Data.Components, &form)
	if form.FileID != "" {
		attachment, ok := payload.Data.Resolved.Attachments[form.FileID]
		if !ok || attachment.URL == "" {
			form.MissingFile = true
		} else {
			form.FileName = attachment.Filename
			form.FileURL = attachment.URL
			form.FileType = attachment.ContentType
			form.FileSize = attachment.Size
		}
	}
	return &discordgo.Interaction{
		ID:        payload.ID,
		AppID:     payload.AppID,
		Type:      payload.Type,
		GuildID:   payload.GuildID,
		ChannelID: payload.ChannelID,
		Token:     payload.Token,
		Member:    payload.Member,
		User:      payload.User,
	}, form, true
}

func collectConfessionFields(raw json.RawMessage, form *confessionForm) {
	if len(raw) == 0 || string(raw) == "null" {
		return
	}
	if raw[0] == '[' {
		var items []json.RawMessage
		if json.Unmarshal(raw, &items) != nil {
			return
		}
		for _, item := range items {
			collectConfessionFields(item, form)
		}
		return
	}
	var part struct {
		CustomID   string          `json:"custom_id"`
		Value      string          `json:"value"`
		Values     []string        `json:"values"`
		Component  json.RawMessage `json:"component"`
		Components json.RawMessage `json:"components"`
	}
	if json.Unmarshal(raw, &part) != nil {
		return
	}
	switch part.CustomID {
	case "text":
		form.Text = part.Value
	case "file":
		if len(part.Values) > 0 {
			form.FileID = part.Values[0]
		}
	}
	collectConfessionFields(part.Component, form)
	collectConfessionFields(part.Components, form)
}

type fileRef struct {
	URL         string
	Name        string
	ContentType string
	Size        int
	Authorized  bool
}

func (b *Bot) downloadConfessionFile(ref fileRef) (*confessionFile, error) {
	if ref.Size > maxConfessionBytes {
		return nil, errors.New("That file is too large. Keep it under 8 MB.")
	}
	parsed, err := allowedFileURL(ref.URL)
	if err != nil {
		return nil, err
	}
	base := http.DefaultClient
	if b.session != nil && b.session.Client != nil {
		base = b.session.Client
	}
	token := ""
	if b.session != nil {
		token = b.session.Token
	}
	client := &http.Client{
		Timeout:   20 * time.Second,
		Transport: base.Transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return errors.New("I could not read that file.")
			}
			if _, err := allowedFileURL(req.URL.String()); err != nil {
				return err
			}
			if auth := via[0].Header.Get("Authorization"); auth != "" {
				req.Header.Set("Authorization", auth)
			}
			return nil
		},
	}
	data, err := readDiscordFile(client, token, parsed, ref.Authorized)
	if err != nil {
		return nil, err
	}
	return &confessionFile{
		Name:        cleanAttachmentName(ref.Name),
		ContentType: cleanContentType(ref.ContentType),
		Data:        data,
	}, nil
}

func readDiscordFile(client *http.Client, token string, parsed *url.URL, authorized bool) ([]byte, error) {
	data, status, err := requestDiscordFile(client, token, parsed, authorized)
	if err != nil {
		return nil, err
	}
	if (status == http.StatusUnauthorized || status == http.StatusForbidden) && !authorized && token != "" {
		data, status, err = requestDiscordFile(client, token, parsed, true)
		if err != nil {
			return nil, err
		}
	}
	if status != http.StatusOK {
		return nil, errors.New("I could not read that file.")
	}
	if len(data) == 0 {
		return nil, errors.New("That file is empty.")
	}
	if len(data) > maxConfessionBytes {
		return nil, errors.New("That file is too large. Keep it under 8 MB.")
	}
	return data, nil
}

func requestDiscordFile(client *http.Client, token string, parsed *url.URL, authorized bool) ([]byte, int, error) {
	req, err := http.NewRequest(http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, 0, errors.New("I could not read that file.")
	}
	if authorized && token != "" {
		req.Header.Set("Authorization", token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, errors.New("I could not read that file.")
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxConfessionBytes+1))
	if err != nil {
		return nil, resp.StatusCode, errors.New("I could not read that file.")
	}
	return data, resp.StatusCode, nil
}

func allowedFileURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Hostname() == "" {
		return nil, errors.New("I could not read that file.")
	}
	if parsed.Scheme != "https" {
		return nil, errors.New("I could not read that file.")
	}
	switch strings.ToLower(parsed.Hostname()) {
	case "cdn.discordapp.com", "media.discordapp.net":
		return parsed, nil
	default:
		return nil, errors.New("I could not read that file.")
	}
}

func cleanAttachmentName(name string) string {
	name = strings.TrimSpace(strings.ReplaceAll(name, "\\", "/"))
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}
	var b strings.Builder
	for _, r := range name {
		if r < 32 || r == 127 {
			continue
		}
		b.WriteRune(r)
	}
	out := strings.Trim(b.String(), ". ")
	if out == "" {
		return "attachment"
	}
	if utf8.RuneCountInString(out) > 80 {
		out = string([]rune(out)[:80])
	}
	return out
}

func cleanContentType(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if i := strings.IndexByte(value, ';'); i >= 0 {
		value = strings.TrimSpace(value[:i])
	}
	if value == "" || strings.Count(value, "/") != 1 || strings.ContainsAny(value, " \r\n") {
		return "application/octet-stream"
	}
	return value
}

func firstAttachment(msg *discordgo.Message) (fileRef, bool) {
	if msg == nil || len(msg.Attachments) == 0 || msg.Attachments[0] == nil || msg.Attachments[0].URL == "" {
		return fileRef{}, false
	}
	item := msg.Attachments[0]
	return fileRef{
		URL:         item.URL,
		Name:        item.Filename,
		ContentType: item.ContentType,
		Size:        item.Size,
	}, true
}
