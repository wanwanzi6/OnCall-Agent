package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server     ServerConfig     `yaml:"server"`
	App        AppConfig        `yaml:"app"`
	Mock       MockConfig       `yaml:"mock"`
	Knowledge  KnowledgeConfig  `yaml:"knowledge"`
	AIOps      AIOpsConfig      `yaml:"aiops"`
	RAG        RAGConfig        `yaml:"rag"`
	Embedding  EmbeddingConfig  `yaml:"embedding"`
	Milvus     MilvusConfig     `yaml:"milvus"`
	Prometheus PrometheusConfig `yaml:"prometheus"`
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

type AIOpsConfig struct {
	AlertProvider  string        `yaml:"alert_provider"`
	LogProvider    string        `yaml:"log_provider"`
	MetricProvider string        `yaml:"metric_provider"`
	Timeout        time.Duration `yaml:"timeout"`
	SOPTopK        int           `yaml:"sop_top_k"`
}

type RAGConfig struct {
	ChunkSize           int    `yaml:"chunk_size"`
	ChunkOverlap        int    `yaml:"chunk_overlap"`
	EmbeddingDim        int    `yaml:"embedding_dim"`
	DefaultTopK         int    `yaml:"default_top_k"`
	EmbedderProvider    string `yaml:"embedder_provider"`
	VectorStoreProvider string `yaml:"vector_store_provider"`
}

type EmbeddingConfig struct {
	DashScope DashScopeEmbeddingConfig `yaml:"dashscope"`
}

type DashScopeEmbeddingConfig struct {
	APIKey     string        `yaml:"api_key"`
	Model      string        `yaml:"model"`
	Dimensions int           `yaml:"dimensions"`
	Timeout    time.Duration `yaml:"timeout"`
}

type MilvusConfig struct {
	Address     string        `yaml:"address"`
	Database    string        `yaml:"database"`
	Collection  string        `yaml:"collection"`
	VectorField string        `yaml:"vector_field"`
	Timeout     time.Duration `yaml:"timeout"`
}

type PrometheusConfig struct {
	BaseURL string        `yaml:"base_url"`
	Timeout time.Duration `yaml:"timeout"`
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
		AIOps: AIOpsConfig{
			AlertProvider:  "mock",
			LogProvider:    "mock",
			MetricProvider: "mock",
			Timeout:        10 * time.Second,
			SOPTopK:        3,
		},
		RAG: RAGConfig{
			ChunkSize:           800,
			ChunkOverlap:        100,
			EmbeddingDim:        64,
			DefaultTopK:         3,
			EmbedderProvider:    "mock",
			VectorStoreProvider: "memory",
		},
		Embedding: EmbeddingConfig{
			DashScope: DashScopeEmbeddingConfig{
				Model:      "text-embedding-v4",
				Dimensions: 1024,
				Timeout:    30 * time.Second,
			},
		},
		Milvus: MilvusConfig{
			Address:     "localhost:19530",
			Database:    "agent",
			Collection:  "oncall_knowledge",
			VectorField: "vector",
			Timeout:     10 * time.Second,
		},
		Prometheus: PrometheusConfig{
			BaseURL: "http://localhost:9090",
			Timeout: 5 * time.Second,
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
	if provider := os.Getenv("RAG_EMBEDDER_PROVIDER"); provider != "" {
		cfg.RAG.EmbedderProvider = provider
	}
	if provider := os.Getenv("RAG_VECTOR_STORE_PROVIDER"); provider != "" {
		cfg.RAG.VectorStoreProvider = provider
	}
	if provider := os.Getenv("AIOPS_ALERT_PROVIDER"); provider != "" {
		cfg.AIOps.AlertProvider = provider
	}
	if provider := os.Getenv("AIOPS_LOG_PROVIDER"); provider != "" {
		cfg.AIOps.LogProvider = provider
	}
	if provider := os.Getenv("AIOPS_METRIC_PROVIDER"); provider != "" {
		cfg.AIOps.MetricProvider = provider
	}
	if timeout := os.Getenv("AIOPS_TIMEOUT"); timeout != "" {
		if parsed, err := time.ParseDuration(timeout); err == nil {
			cfg.AIOps.Timeout = parsed
		}
	}
	if topK := os.Getenv("AIOPS_SOP_TOP_K"); topK != "" {
		if parsed, err := strconv.Atoi(topK); err == nil {
			cfg.AIOps.SOPTopK = parsed
		}
	}
	if apiKey := os.Getenv("DASHSCOPE_API_KEY"); apiKey != "" {
		cfg.Embedding.DashScope.APIKey = apiKey
	}
	if model := os.Getenv("DASHSCOPE_EMBEDDING_MODEL"); model != "" {
		cfg.Embedding.DashScope.Model = model
	}
	if dim := os.Getenv("DASHSCOPE_EMBEDDING_DIM"); dim != "" {
		if parsed, err := strconv.Atoi(dim); err == nil {
			cfg.Embedding.DashScope.Dimensions = parsed
		}
	}
	if timeout := os.Getenv("DASHSCOPE_EMBEDDING_TIMEOUT"); timeout != "" {
		if parsed, err := time.ParseDuration(timeout); err == nil {
			cfg.Embedding.DashScope.Timeout = parsed
		}
	}
	if address := os.Getenv("MILVUS_ADDRESS"); address != "" {
		cfg.Milvus.Address = address
	}
	if database := os.Getenv("MILVUS_DATABASE"); database != "" {
		cfg.Milvus.Database = database
	}
	if collection := os.Getenv("MILVUS_COLLECTION"); collection != "" {
		cfg.Milvus.Collection = collection
	}
	if vectorField := os.Getenv("MILVUS_VECTOR_FIELD"); vectorField != "" {
		cfg.Milvus.VectorField = vectorField
	}
	if timeout := os.Getenv("MILVUS_TIMEOUT"); timeout != "" {
		if parsed, err := time.ParseDuration(timeout); err == nil {
			cfg.Milvus.Timeout = parsed
		}
	}
	if baseURL := os.Getenv("PROMETHEUS_BASE_URL"); baseURL != "" {
		cfg.Prometheus.BaseURL = baseURL
	}
	if timeout := os.Getenv("PROMETHEUS_TIMEOUT"); timeout != "" {
		if parsed, err := time.ParseDuration(timeout); err == nil {
			cfg.Prometheus.Timeout = parsed
		}
	}
}
