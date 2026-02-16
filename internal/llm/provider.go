package llm

import (
	"context"
	"fmt"

	"github.com/swibrow/howdoi/internal/config"
)

// Provider defines the interface for LLM backends.
type Provider interface {
	Complete(ctx context.Context, systemPrompt, userQuery string) (string, error)
}

// NewProvider creates a provider based on the config.
func NewProvider(cfg *config.Config) (Provider, error) {
	switch cfg.Provider {
	case "anthropic":
		return NewAnthropic(cfg.Anthropic)
	case "openai":
		return NewOpenAI(cfg.OpenAI)
	case "ollama":
		return NewOllama(cfg.Ollama)
	default:
		return nil, fmt.Errorf("unknown provider: %s", cfg.Provider)
	}
}
