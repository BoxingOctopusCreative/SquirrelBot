package domain

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/osteele/liquid"
)

const (
	MaxTemplateName     = 40
	MaxTemplateVars     = 20
	MaxTemplateVarKey   = 32
	MaxTemplateVarValue = 200
)

var templateNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 _.-]{0,39}$`)
var templateVarPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{0,31}$`)

var templateEngine = func() *liquid.Engine {
	engine := liquid.NewEngine()
	engine.StrictVariables()
	engine.UnregisterTag("include")
	engine.UnregisterTag("render")
	return engine
}()

type AnnouncementScope struct {
	Server string
	Author string
	Vars   map[string]string
}

func ValidateAnnouncementTemplate(name, title, body string) (string, string, string, error) {
	name = strings.TrimSpace(name)
	title = strings.TrimSpace(title)
	body = strings.TrimSpace(body)
	if !templateNamePattern.MatchString(name) || utf8.RuneCountInString(name) > MaxTemplateName {
		return "", "", "", fmt.Errorf("name the template with letters, numbers, spaces, or ._-")
	}
	if title == "" || utf8.RuneCountInString(title) > MaxAnnounceTitle {
		return "", "", "", fmt.Errorf("template title must be 1 to %d characters", MaxAnnounceTitle)
	}
	if body == "" || utf8.RuneCountInString(body) > MaxAnnounceBody {
		return "", "", "", fmt.Errorf("template body must be 1 to %d characters", MaxAnnounceBody)
	}
	if _, err := templateEngine.ParseString(title); err != nil {
		return "", "", "", fmt.Errorf("title: %s", liquidMessage(err))
	}
	if _, err := templateEngine.ParseString(body); err != nil {
		return "", "", "", fmt.Errorf("body: %s", liquidMessage(err))
	}
	return name, title, body, nil
}

func NormalizeTemplateVars(vars map[string]string) (map[string]string, error) {
	if len(vars) > MaxTemplateVars {
		return nil, fmt.Errorf("use at most %d values", MaxTemplateVars)
	}
	out := make(map[string]string, len(vars))
	for key, value := range vars {
		key = strings.TrimSpace(key)
		if key == "server" || key == "author" {
			continue
		}
		if !templateVarPattern.MatchString(key) {
			return nil, fmt.Errorf("value names must start with a letter")
		}
		value = strings.TrimSpace(value)
		if value == "" || utf8.RuneCountInString(value) > MaxTemplateVarValue {
			return nil, fmt.Errorf("each value must be 1 to %d characters", MaxTemplateVarValue)
		}
		out[key] = value
	}
	return out, nil
}

func RenderAnnouncementTemplate(title, body string, scope AnnouncementScope) (string, string, error) {
	vars, err := NormalizeTemplateVars(scope.Vars)
	if err != nil {
		return "", "", err
	}
	bindings := map[string]any{
		"server": scope.Server,
		"author": scope.Author,
	}
	for key, value := range vars {
		bindings[key] = value
	}
	renderedTitle, err := renderTemplate(title, bindings)
	if err != nil {
		return "", "", fmt.Errorf("title: %s", liquidMessage(err))
	}
	renderedBody, err := renderTemplate(body, bindings)
	if err != nil {
		return "", "", fmt.Errorf("body: %s", liquidMessage(err))
	}
	renderedTitle = strings.TrimSpace(renderedTitle)
	renderedBody = strings.TrimSpace(renderedBody)
	if err := ValidateAnnouncement(renderedTitle, renderedBody, nil, time.Now()); err != nil {
		return "", "", err
	}
	return renderedTitle, renderedBody, nil
}

func renderTemplate(source string, bindings map[string]any) (string, error) {
	return templateEngine.ParseAndRenderString(source, bindings)
}

func liquidMessage(err error) string {
	message := strings.TrimSpace(err.Error())
	message = strings.ReplaceAll(message, "\n", " ")
	if utf8.RuneCountInString(message) > 180 {
		message = string([]rune(message)[:180])
	}
	return message
}
