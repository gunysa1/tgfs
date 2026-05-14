package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type ChannelConfig struct {
	ID   int64  `yaml:"id"`
	Name string `yaml:"name"`
}

type Config struct {
	Telegram struct {
		BotToken string          `yaml:"bot_token"`
		Channels []ChannelConfig `yaml:"channels"`
	} `yaml:"telegram"`
	Mount struct {
		Path string `yaml:"path"`
	} `yaml:"mount"`
	DB struct {
		Path string `yaml:"path"`
	} `yaml:"db"`
	Cache struct {
		MaxSizeGB int `yaml:"max_size_gb"`
	} `yaml:"cache"`
	Chunk struct {
		SizeMB int `yaml:"size_mb"`
	} `yaml:"chunk"`
	Migrate struct {
		FileWorkers  int `yaml:"file_workers"`
		ChunkWorkers int `yaml:"chunk_workers"`
	} `yaml:"migrate"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg.Telegram.BotToken == "" {
		return nil, fmt.Errorf("telegram.bot_token is required")
	}
	if cfg.Mount.Path == "" {
		cfg.Mount.Path = "/mnt/tgfs"
	}
	if cfg.DB.Path == "" {
		cfg.DB.Path = "/var/lib/tgfs/tgfs.db"
	}
	if cfg.Cache.MaxSizeGB == 0 {
		cfg.Cache.MaxSizeGB = 2
	}
	if cfg.Chunk.SizeMB == 0 {
		cfg.Chunk.SizeMB = 1900
	}
	if cfg.Migrate.FileWorkers == 0 {
		cfg.Migrate.FileWorkers = 4
	}
	if cfg.Migrate.ChunkWorkers == 0 {
		cfg.Migrate.ChunkWorkers = 4
	}
	return &cfg, nil
}
