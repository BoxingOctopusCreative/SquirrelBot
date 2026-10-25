package harness

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"time"

	"github.com/jackc/pgx/v5"
)

const defaultAdminURL = "postgres://squirrelbot:squirrelbot@127.0.0.1:5432/postgres?sslmode=disable"

var dbNamePattern = regexp.MustCompile(`^squirrelbot_e2e_[a-f0-9]{12}$`)

func adminURL() string {
	if value := os.Getenv("SQUIRRELBOT_TEST_DATABASE_URL"); value != "" {
		return value
	}
	return defaultAdminURL
}

func openDatabase(ctx context.Context) (name, appURL, admin string, err error) {
	admin = adminURL()
	if err = ensurePostgres(ctx, admin); err != nil {
		return "", "", "", err
	}
	raw := make([]byte, 6)
	if _, err = rand.Read(raw); err != nil {
		return "", "", "", err
	}
	name = "squirrelbot_e2e_" + hex.EncodeToString(raw)
	if err = execAdmin(ctx, admin, fmt.Sprintf("CREATE DATABASE %s", name)); err != nil {
		return "", "", "", err
	}
	appURL, err = withDatabase(admin, name)
	if err != nil {
		_ = dropDatabase(context.Background(), admin, name)
		return "", "", "", err
	}
	return name, appURL, admin, nil
}

func dropDatabase(ctx context.Context, admin, name string) error {
	if !dbNamePattern.MatchString(name) {
		return fmt.Errorf("refusing to drop %q", name)
	}
	conn, err := pgx.Connect(ctx, admin)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	_, _ = conn.Exec(ctx, `
		SELECT pg_terminate_backend(pid)
		FROM pg_stat_activity
		WHERE datname = $1 AND pid <> pg_backend_pid()`, name)
	_, err = conn.Exec(ctx, "DROP DATABASE IF EXISTS "+name)
	return err
}

func execAdmin(ctx context.Context, admin, sql string) error {
	conn, err := pgx.Connect(ctx, admin)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	_, err = conn.Exec(ctx, sql)
	return err
}

func withDatabase(raw, name string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	parsed.Path = "/" + name
	return parsed.String(), nil
}

func ensurePostgres(ctx context.Context, admin string) error {
	if err := ping(ctx, admin); err == nil {
		return nil
	}
	root := moduleRoot()
	if root == "" {
		return errors.New("could not find the module root to start Postgres")
	}
	bin, err := composeBin()
	if err != nil {
		return fmt.Errorf("postgres is not accepting connections, and no container runtime is available: %w", err)
	}
	cmd := exec.CommandContext(ctx, bin, "compose", "up", "postgres", "-d")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("start postgres: %w: %s", err, out)
	}
	deadline := time.Now().Add(40 * time.Second)
	var last error
	for time.Now().Before(deadline) {
		last = ping(ctx, admin)
		if last == nil {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("postgres did not accept connections: %w", last)
}

func ping(ctx context.Context, admin string) error {
	conn, err := pgx.Connect(ctx, admin)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	return conn.Ping(ctx)
}

func composeBin() (string, error) {
	if _, err := exec.LookPath("docker"); err == nil {
		return "docker", nil
	}
	if _, err := exec.LookPath("podman"); err == nil {
		return "podman", nil
	}
	return "", errors.New("install Docker or Podman, or set SQUIRRELBOT_TEST_DATABASE_URL")
}

func moduleRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}
