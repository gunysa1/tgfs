package config_test

import (
	"os"
	"testing"

	"github.com/gunysa1/tgfs/internal/config"
)

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("write config: %v", err)
	}
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })
	return f.Name()
}

func TestLoadConfig(t *testing.T) {
	path := writeTempConfig(t, `
telegram:
  bot_token: "test-token"
  channels:
    - id: -1001234567890
      name: movies
mount:
  path: /mnt/tgfs
db:
  path: /tmp/test.db
cache:
  max_size_gb: 2
chunk:
  size_mb: 1900
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Telegram.BotToken != "test-token" {
		t.Errorf("expected bot_token 'test-token', got %q", cfg.Telegram.BotToken)
	}
	if len(cfg.Telegram.Channels) != 1 {
		t.Errorf("expected 1 channel, got %d", len(cfg.Telegram.Channels))
	}
	if cfg.Telegram.Channels[0].Name != "movies" {
		t.Errorf("expected channel name 'movies', got %q", cfg.Telegram.Channels[0].Name)
	}
	if cfg.Mount.Path != "/mnt/tgfs" {
		t.Errorf("expected mount path '/mnt/tgfs', got %q", cfg.Mount.Path)
	}
	if cfg.Chunk.SizeMB != 1900 {
		t.Errorf("expected chunk size 1900, got %d", cfg.Chunk.SizeMB)
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	path := writeTempConfig(t, `
telegram:
  bot_token: "tok"
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Mount.Path != "/mnt/tgfs" {
		t.Errorf("expected default mount path, got %q", cfg.Mount.Path)
	}
	if cfg.DB.Path != "/var/lib/tgfs/tgfs.db" {
		t.Errorf("expected default db path, got %q", cfg.DB.Path)
	}
	if cfg.Cache.MaxSizeGB != 2 {
		t.Errorf("expected default cache 2, got %d", cfg.Cache.MaxSizeGB)
	}
	if cfg.Chunk.SizeMB != 1900 {
		t.Errorf("expected default chunk 1900, got %d", cfg.Chunk.SizeMB)
	}
}

func TestLoadConfig_MissingBotToken(t *testing.T) {
	path := writeTempConfig(t, `
telegram:
  bot_token: ""
`)
	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for missing bot_token, got nil")
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := config.Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}
