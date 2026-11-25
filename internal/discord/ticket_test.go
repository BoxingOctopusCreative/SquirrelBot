package discord

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestTicketOpenModalUsesLabels(t *testing.T) {
	raw, err := json.Marshal(ticketOpenModal().Data.Components)
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	for _, piece := range []string{`"type":18`, `"custom_id":"subject"`, `"custom_id":"details"`, `"label":"Subject"`} {
		if !strings.Contains(body, piece) {
			t.Fatalf("missing %s in %s", piece, body)
		}
	}
	if strings.Contains(body, `"type":1,`) || strings.Contains(body, `"type":1}`) {
		t.Fatalf("text inputs are still in an action row: %s", body)
	}
}

func TestLabelModalSubmitIsReadFromTheRawEvent(t *testing.T) {
	raw := []byte(`{
		"id": "123456789012345678",
		"application_id": "555555555555555555",
		"type": 5,
		"guild_id": "111111111111111111",
		"channel_id": "777777777777777771",
		"token": "token",
		"member": {"user": {"id": "222222222222222222", "username": "acorn"}},
		"data": {
			"custom_id": "sb:ticket:form",
			"components": [
				{"type": 18, "component": {"type": 4, "custom_id": "subject", "value": "Map pin"}},
				{"type": 18, "component": {"type": 4, "custom_id": "details", "value": "The pin is missing."}}
			]
		}
	}`)
	var created discordgo.InteractionCreate
	if err := json.Unmarshal(raw, &created); err == nil && created.Interaction != nil && created.Data != nil {
		t.Fatal("discordgo parsed a label modal, so the raw parser would open a second ticket")
	}
	interaction, customID, values, ok := ticketFormFromEvent(raw)
	if !ok || customID != "sb:ticket:form" || interaction.Member.User.ID != "222222222222222222" {
		t.Fatalf("interaction %+v id %s ok %v", interaction, customID, ok)
	}
	if values["subject"] != "Map pin" || values["details"] != "The pin is missing." {
		t.Fatalf("values %v", values)
	}
	actionRow := []byte(`{"type":5,"data":{"custom_id":"sb:ticket:form","components":[{"type":1,"components":[{"type":4,"custom_id":"subject","value":"Map pin"}]}]}}`)
	if _, _, _, ok := ticketFormFromEvent(actionRow); ok {
		t.Fatal("an action row submit would be handled twice")
	}
}
