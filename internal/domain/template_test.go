package domain

import "testing"

func TestRenderAnnouncementTemplate(t *testing.T) {
	title, body, err := RenderAnnouncementTemplate(
		"{{ server | upcase }} downtime",
		"{% if when %}{{ server }} pauses at {{ when }}.{% else %}{{ server }} is open.{% endif %} Posted by {{ author }}.",
		AnnouncementScope{Server: "Canopy", Author: "Ada", Vars: map[string]string{"when": "Tonight", "server": "Other"}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if title != "CANOPY downtime" {
		t.Fatalf("title %q", title)
	}
	if body != "Canopy pauses at Tonight. Posted by Ada." {
		t.Fatalf("body %q", body)
	}
}

func TestRenderAnnouncementTemplateRejectsMissingValue(t *testing.T) {
	_, _, err := RenderAnnouncementTemplate("{{ server }}", "See you {{ when }}.", AnnouncementScope{Server: "Canopy", Author: "Ada"})
	if err == nil {
		t.Fatal("expected a missing value to fail")
	}
}

func TestRenderAnnouncementTemplateRejectsInclude(t *testing.T) {
	_, _, _, err := ValidateAnnouncementTemplate("Note", "Hello", `{% include "secret.liquid" %}`)
	if err == nil {
		t.Fatal("include should be rejected")
	}
}

func TestValidateAnnouncementTemplate(t *testing.T) {
	name, title, body, err := ValidateAnnouncementTemplate("  Maintenance window ", "Down at {{ when }}", "We will be back.")
	if err != nil {
		t.Fatal(err)
	}
	if name != "Maintenance window" || title != "Down at {{ when }}" || body != "We will be back." {
		t.Fatalf("stored %q %q %q", name, title, body)
	}
	if _, _, _, err := ValidateAnnouncementTemplate("", "Hi", "There"); err == nil {
		t.Fatal("expected a name")
	}
	if _, _, _, err := ValidateAnnouncementTemplate("Note", "{% if %}", "There"); err == nil {
		t.Fatal("expected a liquid error")
	}
}
