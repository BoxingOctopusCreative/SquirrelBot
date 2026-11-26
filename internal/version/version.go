package version

import (
	"runtime/debug"
	"strings"
)

// Version is the release stamped into this binary.
// Release and image builds set it with
// -ldflags "-X github.com/BoxingOctopus/squirrelbot/internal/version.Version=0.7.1".
var Version = "dev"

// String returns the stamped release. A leading v is removed.
// An unstamped binary reports a short commit, or dev when that is unavailable.
func String() string {
	if stamped := clean(Version); stamped != "" && stamped != "dev" {
		return stamped
	}
	if revision := commit(); revision != "" {
		return revision
	}
	if stamped := clean(Version); stamped != "" {
		return stamped
	}
	return "dev"
}

func clean(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 && (value[0] == 'v' || value[0] == 'V') && value[1] >= '0' && value[1] <= '9' {
		return value[1:]
	}
	return value
}

func commit() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	revision := ""
	dirty := false
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			dirty = setting.Value == "true"
		}
	}
	if revision == "" {
		return ""
	}
	if len(revision) > 12 {
		revision = revision[:12]
	}
	if dirty {
		return revision + "-dirty"
	}
	return revision
}
