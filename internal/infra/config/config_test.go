package config

import "testing"

func TestLoadDefaultsAndEnvOverride(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("MOCK_ENABLED", "false")
	t.Setenv("KNOWLEDGE_UPLOAD_DIR", "tmp/uploads")
	t.Setenv("KNOWLEDGE_MAX_FILE_SIZE_BYTES", "1024")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.App.Env != "test" {
		t.Fatalf("env = %q, want test", cfg.App.Env)
	}
	if cfg.Server.Port != 9090 {
		t.Fatalf("port = %d, want 9090", cfg.Server.Port)
	}
	if cfg.Mock.Enabled {
		t.Fatal("mock should be disabled by env")
	}
	if cfg.Knowledge.UploadDir != "tmp/uploads" {
		t.Fatalf("upload dir = %q", cfg.Knowledge.UploadDir)
	}
	if cfg.Knowledge.MaxFileSizeBytes != 1024 {
		t.Fatalf("max size = %d, want 1024", cfg.Knowledge.MaxFileSizeBytes)
	}
}
