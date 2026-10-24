package api

import (
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BoxingOctopus/squirrelbot/internal/config"
	"github.com/gofiber/fiber/v2"
)

func TestHealthAndOrigin(t *testing.T) {
	server := New(config.Config{
		PublicURL:     "http://localhost:8080",
		SessionSecret: "0123456789abcdef0123456789abcdef",
		WebDir:        "missing",
	}, nil, nil, nil)
	res, err := server.app.Test(httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("health status %d", res.StatusCode)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/guilds/123/announcements", nil)
	req.Header.Set("Origin", "https://evil.example")
	res, err = server.app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusUnauthorized {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("status %d body %s", res.StatusCode, body)
	}

	app := fiber.New()
	app.Use(server.requireSameOrigin)
	app.Post("/x", func(c *fiber.Ctx) error { return c.SendStatus(204) })
	blocked := httptest.NewRequest(http.MethodPost, "/x", nil)
	blocked.Header.Set("Origin", "https://evil.example")
	res, err = app.Test(blocked)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("origin status %d", res.StatusCode)
	}
	allowed := httptest.NewRequest(http.MethodPost, "/x", nil)
	allowed.Header.Set("Origin", "http://localhost:8080")
	res, err = app.Test(allowed)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("same origin status %d", res.StatusCode)
	}
}

func TestInlineScriptCSP(t *testing.T) {
	dir := t.TempDir()
	const page = "<html><body><script>\nready()\n</script></body></html>"
	if err := os.WriteFile(filepath.Join(dir, "200.html"), []byte(page), 0o644); err != nil {
		t.Fatal(err)
	}
	server := New(config.Config{
		PublicURL:     "http://localhost:8080",
		SessionSecret: "0123456789abcdef0123456789abcdef",
		WebDir:        dir,
	}, nil, nil, nil)
	res, err := server.app.Test(httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatal(err)
	}
	policy := res.Header.Get("Content-Security-Policy")
	sum := sha256.Sum256([]byte("\nready()\n"))
	want := "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
	if !strings.Contains(policy, want) {
		t.Fatalf("policy %s missing %s", policy, want)
	}
}
