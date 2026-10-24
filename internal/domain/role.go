package domain

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// CleanRoleName trims a role name and checks Discord's length limit.
func CleanRoleName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if strings.EqualFold(name, "@everyone") || strings.EqualFold(name, "@here") {
		return "", fmt.Errorf("that name is reserved")
	}
	count := utf8.RuneCountInString(name)
	if count < 1 || count > 100 {
		return "", fmt.Errorf("role names are 1 to 100 characters")
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return "", fmt.Errorf("role names cannot include a line break")
		}
	}
	return name, nil
}
