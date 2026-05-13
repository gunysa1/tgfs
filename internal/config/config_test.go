package config_test

import (
	"os"
	"testing"

	"github.com/gunysa1/tgfs/internal/config"
)

func TestLoadConfig(t *testing.T) {
	yaml := `
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
`
	f, _ := os.CreateTemp("", "config-*.yaml")
	f.WriteString(yaml)
	f.Close()
	defer os.Remove(f.Name())

	cfg, err := config.Load(f.Name())
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

func TestLoadConfig_MissingBotToken(t *testing.T) {
	yaml := `
telegram:
  bot_token: ""
mount:
  path: /mnt/tgfs
db:
  path: /tmp/test.db
`
	f, _ := os.CreateTemp("", "config-*.yaml")
	f.WriteString(yaml)
	f.Close()
	defer os.Remove(f.Name())

	_, err := config.Load(f.Name())
	if err == nil {
		t.Fatal("expected error for missing bot_token, got nil")
	}
}
