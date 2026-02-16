package llm

import (
	"strings"
	"testing"

	"github.com/swibrow/howdoi/internal/config"
)

func TestNewProviderUnknown(t *testing.T) {
	cfg := &config.Config{Provider: "unknown-provider"}
	_, err := NewProvider(cfg)
	if err == nil {
		t.Fatal("expected error for unknown provider, got nil")
	}
	if !strings.Contains(err.Error(), "unknown provider") {
		t.Errorf("expected 'unknown provider' in error, got: %v", err)
	}
}

func TestNewProviderAnthropicNoKey(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Provider = "anthropic"
	cfg.Anthropic.APIKey = ""

	_, err := NewProvider(cfg)
	if err == nil {
		t.Fatal("expected error for missing anthropic API key, got nil")
	}
	if !strings.Contains(err.Error(), "API key") {
		t.Errorf("expected 'API key' in error, got: %v", err)
	}
}

func TestNewProviderOpenAINoKey(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Provider = "openai"
	cfg.OpenAI.APIKey = ""

	_, err := NewProvider(cfg)
	if err == nil {
		t.Fatal("expected error for missing openai API key, got nil")
	}
	if !strings.Contains(err.Error(), "API key") {
		t.Errorf("expected 'API key' in error, got: %v", err)
	}
}

func TestNewProviderOllama(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Provider = "ollama"

	provider, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("expected no error for ollama, got: %v", err)
	}
	if provider == nil {
		t.Fatal("expected non-nil provider for ollama")
	}
}
