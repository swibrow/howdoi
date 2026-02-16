package config

import (
	"os"
	"testing"
)

func setupTestDir(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	ConfigDirFunc = func() (string, error) { return dir, nil }
	t.Cleanup(func() { ConfigDirFunc = nil })
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Provider != "anthropic" {
		t.Errorf("expected default provider 'anthropic', got %q", cfg.Provider)
	}
	if cfg.Anthropic.Model != "claude-sonnet-4-5-20250929" {
		t.Errorf("unexpected anthropic model: %q", cfg.Anthropic.Model)
	}
	if cfg.OpenAI.Model != "gpt-4o" {
		t.Errorf("unexpected openai model: %q", cfg.OpenAI.Model)
	}
	if cfg.Ollama.Model != "llama3" {
		t.Errorf("unexpected ollama model: %q", cfg.Ollama.Model)
	}
	if cfg.Ollama.URL != "http://localhost:11434/v1" {
		t.Errorf("unexpected ollama URL: %q", cfg.Ollama.URL)
	}
}

func TestLoadNoFile(t *testing.T) {
	setupTestDir(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if cfg.Provider != "anthropic" {
		t.Errorf("expected default provider, got %q", cfg.Provider)
	}
}

func TestSaveAndLoad(t *testing.T) {
	setupTestDir(t)

	original := DefaultConfig()
	original.Provider = "openai"
	original.OpenAI.APIKey = "test-key-123"
	original.OpenAI.Model = "gpt-4o-mini"

	if err := Save(original); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if loaded.Provider != original.Provider {
		t.Errorf("provider: got %q, want %q", loaded.Provider, original.Provider)
	}
	if loaded.OpenAI.APIKey != original.OpenAI.APIKey {
		t.Errorf("openai api_key: got %q, want %q", loaded.OpenAI.APIKey, original.OpenAI.APIKey)
	}
	if loaded.OpenAI.Model != original.OpenAI.Model {
		t.Errorf("openai model: got %q, want %q", loaded.OpenAI.Model, original.OpenAI.Model)
	}
}

func TestEnvVarOverride(t *testing.T) {
	setupTestDir(t)

	// Save a config with no API keys
	cfg := DefaultConfig()
	if err := Save(cfg); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	t.Setenv("ANTHROPIC_API_KEY", "env-anthropic-key")
	t.Setenv("OPENAI_API_KEY", "env-openai-key")

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if loaded.Anthropic.APIKey != "env-anthropic-key" {
		t.Errorf("anthropic key: got %q, want %q", loaded.Anthropic.APIKey, "env-anthropic-key")
	}
	if loaded.OpenAI.APIKey != "env-openai-key" {
		t.Errorf("openai key: got %q, want %q", loaded.OpenAI.APIKey, "env-openai-key")
	}
}

func TestShowNoFile(t *testing.T) {
	setupTestDir(t)

	output, err := Show()
	if err != nil {
		t.Fatalf("Show() error: %v", err)
	}
	if !contains(output, "No config file found") {
		t.Errorf("expected 'No config file found' message, got: %s", output)
	}
}

func TestShowWithFile(t *testing.T) {
	setupTestDir(t)

	cfg := DefaultConfig()
	if err := Save(cfg); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	output, err := Show()
	if err != nil {
		t.Fatalf("Show() error: %v", err)
	}
	if !contains(output, "Config file:") {
		t.Errorf("expected 'Config file:' header, got: %s", output)
	}
	if !contains(output, "provider: anthropic") {
		t.Errorf("expected config content in output, got: %s", output)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestMain(m *testing.M) {
	// Ensure tests don't accidentally use real env vars
	os.Unsetenv("ANTHROPIC_API_KEY")
	os.Unsetenv("OPENAI_API_KEY")
	os.Exit(m.Run())
}
