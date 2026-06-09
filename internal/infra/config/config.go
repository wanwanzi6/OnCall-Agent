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
	RAG       RAGConfig       `yaml:"rag"`
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

type RAGConfig struct {
	ChunkSize    int `yaml:"chunk_size"`
	ChunkOverlap int `yaml:"chunk_overlap"`
	EmbeddingDim int `yaml:"embedding_dim"`
	DefaultTopK  int `yaml:"default_top_k"`
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
			AllowedExts:      []string{".md", ".markdown", ".txt"},
		},
		RAG: RAGConfig{
			ChunkSize:    800,
			ChunkOverlap: 100,
			EmbeddingDim: 64,
			DefaultTopK:  3,
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
	if chunkSize := os.Getenv("RAG_CHUNK_SIZE"); chunkSize != "" {
		if parsed, err := strconv.Atoi(chunkSize); err == nil {
			cfg.RAG.ChunkSize = parsed
		}
	}
	if chunkOverlap := os.Getenv("RAG_CHUNK_OVERLAP"); chunkOverlap != "" {
		if parsed, err := strconv.Atoi(chunkOverlap); err == nil {
			cfg.RAG.ChunkOverlap = parsed
		}
	}
	if embeddingDim := os.Getenv("RAG_EMBEDDING_DIM"); embeddingDim != "" {
		if parsed, err := strconv.Atoi(embeddingDim); err == nil {
			cfg.RAG.EmbeddingDim = parsed
		}
	}
	if topK := os.Getenv("RAG_DEFAULT_TOP_K"); topK != "" {
		if parsed, err := strconv.Atoi(topK); err == nil {
			cfg.RAG.DefaultTopK = parsed
		}
	}
}
