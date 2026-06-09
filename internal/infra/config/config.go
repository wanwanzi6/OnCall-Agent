package config

import (
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server    ServerConfig    `yaml:"server"`
	App       AppConfig       `yaml:"app"`
	Mock      MockConfig      `yaml:"mock"`
	Knowledge KnowledgeConfig `yaml:"knowledge"`
}

type ServerConfig struct {
	Port int `yaml:"port"`
}

type AppConfig struct {
	Env string `yaml:"env"`
}

type MockConfig struct {
	Enabled bool `yaml:"enabled"`
}

type KnowledgeConfig struct {
	UploadDir        string   `yaml:"upload_dir"`
	MaxFileSizeBytes int64    `yaml:"max_file_size_bytes"`
	AllowedExts      []string `yaml:"allowed_exts"`
}

func Load(path string) (*Config, error) {
	cfg := defaultConfig()

	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read config file: %w", err)
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parse config file: %w", err)
		}
	}

	applyEnv(cfg)
	return cfg, nil
}

func defaultConfig() *Config {
	return &Config{
		Server: ServerConfig{Port: 8080},
		App:    AppConfig{Env: "dev"},
		Mock:   MockConfig{Enabled: true},
		Knowledge: KnowledgeConfig{
			UploadDir:        "data/uploads",
			MaxFileSizeBytes: 2 * 1024 * 1024,
			AllowedExts:      []string{".md", ".txt"},
		},
	}
}

func applyEnv(cfg *Config) {
	if env := os.Getenv("APP_ENV"); env != "" {
		cfg.App.Env = env
	}
	if port := os.Getenv("SERVER_PORT"); port != "" {
		if parsed, err := strconv.Atoi(port); err == nil {
			cfg.Server.Port = parsed
		}
	}
	if enabled := os.Getenv("MOCK_ENABLED"); enabled != "" {
		if parsed, err := strconv.ParseBool(enabled); err == nil {
			cfg.Mock.Enabled = parsed
		}
	}
	if uploadDir := os.Getenv("KNOWLEDGE_UPLOAD_DIR"); uploadDir != "" {
		cfg.Knowledge.UploadDir = uploadDir
	}
	if maxSize := os.Getenv("KNOWLEDGE_MAX_FILE_SIZE_BYTES"); maxSize != "" {
		if parsed, err := strconv.ParseInt(maxSize, 10, 64); err == nil {
			cfg.Knowledge.MaxFileSizeBytes = parsed
		}
	}
}
