package config

import (
	"fmt"
	"os"
	"strings"
)

// Config is process configuration. Secrets come from the environment, never from source.
type Config struct {
	DatabaseURL         string
	DiscordToken        string
	DiscordClientID     string
	DiscordClientSecret string
	SessionSecret       string
	PublicURL           string
	HTTPAddr            string
	WebDir              string
	SecureCookies       bool
}

func Load() (Config, error) {
	if err := loadEnvFile(); err != nil {
		return Config{}, err
	}
	cfg := Config{
		DatabaseURL:         strings.TrimSpace(os.Getenv("DATABASE_URL")),
		DiscordToken:        strings.TrimSpace(os.Getenv("DISCORD_TOKEN")),
		DiscordClientID:     strings.TrimSpace(os.Getenv("DISCORD_CLIENT_ID")),
		DiscordClientSecret: strings.TrimSpace(os.Getenv("DISCORD_CLIENT_SECRET")),
		SessionSecret:       os.Getenv("SESSION_SECRET"),
		PublicURL:           strings.TrimRight(strings.TrimSpace(os.Getenv("PUBLIC_URL")), "/"),
		HTTPAddr:            strings.TrimSpace(os.Getenv("HTTP_ADDR")),
		WebDir:              strings.TrimSpace(os.Getenv("WEB_DIR")),
	}
	if cfg.HTTPAddr == "" {
		cfg.HTTPAddr = ":8080"
	}
	if cfg.WebDir == "" {
		cfg.WebDir = "web/build"
	}
	if cfg.PublicURL == "" {
		cfg.PublicURL = "http://localhost:8080"
	}
	var missing []string
	if cfg.DatabaseURL == "" {
		missing = append(missing, "DATABASE_URL")
	}
	if cfg.DiscordToken == "" {
		missing = append(missing, "DISCORD_TOKEN")
	}
	if cfg.DiscordClientID == "" {
		missing = append(missing, "DISCORD_CLIENT_ID")
	}
	if cfg.DiscordClientSecret == "" {
		missing = append(missing, "DISCORD_CLIENT_SECRET")
	}
	if strings.TrimSpace(cfg.SessionSecret) == "" {
		missing = append(missing, "SESSION_SECRET")
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required environment: %s", strings.Join(missing, ", "))
	}
	if len(cfg.SessionSecret) < 32 {
		return Config{}, fmt.Errorf("SESSION_SECRET must be at least 32 characters")
	}
	if !strings.HasPrefix(cfg.PublicURL, "http://") && !strings.HasPrefix(cfg.PublicURL, "https://") {
		return Config{}, fmt.Errorf("PUBLIC_URL must start with http:// or https://")
	}
	cfg.SecureCookies = strings.HasPrefix(cfg.PublicURL, "https://")
	return cfg, nil
}

// Origin is the scheme and host of the public URL, used for OAuth and same-origin checks.
func (c Config) Origin() string {
	u := c.PublicURL
	if i := strings.Index(u, "://"); i >= 0 {
		rest := u[i+3:]
		if slash := strings.IndexByte(rest, '/'); slash >= 0 {
			return u[:i+3+slash]
		}
	}
	return u
}
