package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// loadEnvFile reads KEY=VALUE lines from .env in the working directory, or from the
// module root when the process was started in a subdirectory. Variables already set
// in the environment are left alone.
func loadEnvFile() error {
	path, ok, err := findEnvFile()
	if err != nil || !ok {
		return err
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", filepath.Base(path), err)
	}
	text := strings.TrimPrefix(string(body), "\uFEFF")
	for i, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(strings.TrimRight(line, "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		key, val, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(key) == "" || strings.ContainsAny(strings.TrimSpace(key), " \t") {
			return fmt.Errorf(".env:%d: expected KEY=VALUE", i+1)
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if len(val) >= 2 {
			if (val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'') {
				val = val[1 : len(val)-1]
			}
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		if err := os.Setenv(key, val); err != nil {
			return fmt.Errorf(".env: %s: %w", key, err)
		}
	}
	return nil
}

func findEnvFile() (string, bool, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", false, err
	}
	for range 8 {
		envPath := filepath.Join(dir, ".env")
		if info, err := os.Stat(envPath); err == nil && !info.IsDir() {
			return envPath, true, nil
		}
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return "", false, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false, nil
		}
		dir = parent
	}
	return "", false, nil
}
