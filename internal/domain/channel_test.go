package domain

import "testing"

func TestNormalizeChannelName(t *testing.T) {
	name, err := NormalizeChannelName("  Mod Log  ")
	if err != nil || name != "mod-log" {
		t.Fatalf("mod log: %q %v", name, err)
	}
	name, err = NormalizeChannelName("role_picks")
	if err != nil || name != "role_picks" {
		t.Fatalf("underscore: %q %v", name, err)
	}
	name, err = NormalizeChannelName("Hello!! World")
	if err != nil || name != "hello-world" {
		t.Fatalf("punctuation: %q %v", name, err)
	}
	if _, err := NormalizeChannelName("a"); err == nil {
		t.Fatal("one character should fail")
	}
	if _, err := NormalizeChannelName("!!"); err == nil {
		t.Fatal("punctuation only should fail")
	}
}

func TestSetupPurpose(t *testing.T) {
	topic, private, ok := SetupPurpose("logging")
	if !ok || !private || topic == "" {
		t.Fatalf("logging %q %v %v", topic, private, ok)
	}
	_, private, ok = SetupPurpose("announcements")
	if !ok || private {
		t.Fatal("announcements should be public")
	}
	_, private, ok = SetupPurpose("roles")
	if !ok || private {
		t.Fatal("roles should be public")
	}
	_, private, ok = SetupPurpose("starboard")
	if !ok || private {
		t.Fatal("starboard should be public")
	}
	_, private, ok = SetupPurpose("confession-review")
	if !ok || !private {
		t.Fatal("confession review should be private")
	}
	_, private, ok = SetupPurpose("confessions")
	if !ok || private {
		t.Fatal("confessions should be public")
	}
	_, private, ok = SetupPurpose("ticket-panel")
	if !ok || private {
		t.Fatal("ticket panel should be public")
	}
	_, private, ok = SetupPurpose("ticket-log")
	if !ok || !private {
		t.Fatal("ticket transcripts should be private")
	}
	if _, _, ok := SetupPurpose("general"); ok {
		t.Fatal("unknown purpose")
	}
	kind, _, _, ok := ClassifySetup("ticket-category")
	if !ok || kind != "category" {
		t.Fatal("ticket category")
	}
	kind, _, private, ok = ClassifySetup("ticket-log")
	if !ok || kind != "text" || !private {
		t.Fatal("ticket log kind")
	}
}
