package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnvFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("# comment\nexport PUBLIC_URL=\"http://localhost:5173\"\nDISCORD_TOKEN=from-file\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(root, "cmd", "squirrelbot")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(sub)
	t.Setenv("DISCORD_TOKEN", "from-shell")
	previous, had := os.LookupEnv("PUBLIC_URL")
	os.Unsetenv("PUBLIC_URL")
	t.Cleanup(func() {
		if had {
			os.Setenv("PUBLIC_URL", previous)
		} else {
			os.Unsetenv("PUBLIC_URL")
		}
	})

	if err := loadEnvFile(); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("DISCORD_TOKEN"); got != "from-shell" {
		t.Fatalf("shell value was replaced")
	}
	if got := os.Getenv("PUBLIC_URL"); got != "http://localhost:5173" {
		t.Fatalf("PUBLIC_URL = %q", got)
	}
}

func TestLoadEnvFileMissing(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	if err := loadEnvFile(); err != nil {
		t.Fatal(err)
	}
}
