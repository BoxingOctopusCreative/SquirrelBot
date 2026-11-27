package discord

import "testing"

func TestVoiceRoster(t *testing.T) {
	roster := newVoiceRoster()
	roster.set("g", "a", "lobby", "Acorn", []string{"role"}, true)
	roster.set("g", "b", "lobby", "Fern", nil, true)
	roster.set("g", "c", "afk", "Quiet", nil, true)
	channels := roster.channels("g")
	if len(channels["lobby"]) != 2 || len(channels["afk"]) != 1 {
		t.Fatalf("channels %#v", channels)
	}
	roster.set("g", "c", "", "", nil, false)
	if _, ok := roster.channels("g")["afk"]; ok {
		t.Fatal("leave should drop the seat")
	}
	roster.rememberRoles("g", "a", "Acorn", []string{"role", "other"})
	found := false
	for _, person := range roster.channels("g")["lobby"] {
		if person.userID == "a" && person.known && len(person.roles) == 2 {
			found = true
		}
	}
	if !found {
		t.Fatal("roles were not stored")
	}
}

func TestGrouped(t *testing.T) {
	if grouped(0) != "0" || grouped(999) != "999" || grouped(1000) != "1,000" || grouped(1899250) != "1,899,250" {
		t.Fatalf("%s %s %s %s", grouped(0), grouped(999), grouped(1000), grouped(1899250))
	}
}
