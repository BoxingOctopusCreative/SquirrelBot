package discord

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/BoxingOctopus/squirrelbot/internal/domain"
	"github.com/BoxingOctopus/squirrelbot/internal/store"
)

func TestPublicConfessionHidesAuthor(t *testing.T) {
	authorID := "123456789012345678"
	review := reviewConfessionEmbed(store.Confession{
		AuthorID: authorID, AuthorName: "Acorn", Body: "I ate the last acorn", Status: domain.ConfessionPending,
	})
	raw, err := json.Marshal(review)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), authorID) || !strings.Contains(string(raw), "Acorn") {
		t.Fatalf("review embed should name the author: %s", raw)
	}
	public := publicConfessionEmbed("I ate the last acorn")
	raw, err = json.Marshal(public)
	if err != nil {
		t.Fatal(err)
	}
	if public.Author != nil || len(public.Fields) != 0 {
		t.Fatal("public embed should have no author field")
	}
	if strings.Contains(string(raw), authorID) || strings.Contains(string(raw), "Acorn") {
		t.Fatalf("public embed names the author: %s", raw)
	}
	if public.Description != "I ate the last acorn" {
		t.Fatalf("description %q", public.Description)
	}
}

func TestConfessionButtons(t *testing.T) {
	components := confessionButtons(15)
	if len(components) != 1 {
		t.Fatalf("rows %d", len(components))
	}
	encoded, err := json.Marshal(components)
	if err != nil {
		t.Fatal(err)
	}
	body := string(encoded)
	if !strings.Contains(body, domain.ConfessionCustomID("approve", 15)) || !strings.Contains(body, domain.ConfessionCustomID("reject", 15)) {
		t.Fatalf("buttons %s", body)
	}
}

func TestReviewMentionTagsOnlyTheModRole(t *testing.T) {
	roleID := "888888888888888882"
	content, mentions := reviewMention(roleID)
	if content != "<@&"+roleID+">" {
		t.Fatalf("content %q", content)
	}
	raw, err := json.Marshal(mentions)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"parse":[]`) || !strings.Contains(string(raw), roleID) {
		t.Fatalf("mentions %s", raw)
	}
	if strings.Contains(string(raw), "everyone") {
		t.Fatalf("mentions allow everyone: %s", raw)
	}
	content, mentions = reviewMention("")
	if content != "" || mentions != noMentions {
		t.Fatal("missing mod role should not ping")
	}
}

func TestSubmitConfessionButton(t *testing.T) {
	raw, err := json.Marshal(submitConfessionButton())
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	if !strings.Contains(body, "Submit a Confession") || !strings.Contains(body, confessionOpenID) {
		t.Fatalf("button %s", body)
	}
	if strings.Contains(body, "123456789012345678") {
		t.Fatal("the public button should not carry an author id")
	}
}

func TestConfessionFormComponents(t *testing.T) {
	raw, err := json.Marshal(confessionFormComponents())
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	for _, piece := range []string{`"type":18`, `"type":19`, `"type":4`, `"custom_id":"text"`, `"custom_id":"file"`, `"required":false`, `"min_values":0`, `"max_values":1`} {
		if !strings.Contains(body, piece) {
			t.Fatalf("missing %s in %s", piece, body)
		}
	}
	if strings.Contains(body, `"label":""`) {
		t.Fatalf("empty text label %s", body)
	}
}

func TestConfessionFormFromEvent(t *testing.T) {
	raw := []byte(`{
		"id": "123456789012345678",
		"application_id": "555555555555555555",
		"type": 5,
		"guild_id": "111111111111111111",
		"channel_id": "777777777777777771",
		"token": "token",
		"member": {"user": {"id": "222222222222222222", "username": "acorn"}},
		"data": {
			"custom_id": "sb:confess:form",
			"components": [
				{"type": 18, "component": {"type": 4, "custom_id": "text", "value": "I ate the last acorn"}},
				{"type": 18, "component": {"type": 19, "custom_id": "file", "values": ["999999999999999999"]}}
			],
			"resolved": {
				"attachments": {
					"999999999999999999": {
						"filename": "note.txt",
						"url": "https://cdn.discordapp.com/ephemeral-attachments/1/2/note.txt",
						"content_type": "text/plain",
						"size": 12
					}
				}
			}
		}
	}`)
	interaction, form, ok := confessionFormFromEvent(raw)
	if !ok || interaction.GuildID != "111111111111111111" || interaction.Member.User.ID != "222222222222222222" {
		t.Fatalf("interaction %+v ok %v", interaction, ok)
	}
	if form.Text != "I ate the last acorn" || form.FileName != "note.txt" || form.FileSize != 12 || form.MissingFile {
		t.Fatalf("form %+v", form)
	}
	if _, _, ok := confessionFormFromEvent([]byte(`{"type":2,"data":{"custom_id":"sb:confess:form"}}`)); ok {
		t.Fatal("a slash command is not the confession form")
	}
}

func TestAllowedFileURL(t *testing.T) {
	if _, err := allowedFileURL("https://cdn.discordapp.com/ephemeral-attachments/1/2/note.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := allowedFileURL("https://media.discordapp.net/attachments/1/2/note.txt"); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{
		"http://cdn.discordapp.com/a.txt",
		"https://evil.example/a.txt",
		"https://cdn.discordapp.com.evil.example/a.txt",
		"https://127.0.0.1/a.txt",
	} {
		if _, err := allowedFileURL(raw); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
}

func TestCleanAttachmentName(t *testing.T) {
	if got := cleanAttachmentName(`..\secret\note.txt`); got != "note.txt" {
		t.Fatalf("path %q", got)
	}
	if got := cleanAttachmentName(".."); got != "attachment" {
		t.Fatalf("dots %q", got)
	}
	if got := cleanContentType("Image/PNG; charset=utf-8"); got != "image/png" {
		t.Fatalf("type %q", got)
	}
}
