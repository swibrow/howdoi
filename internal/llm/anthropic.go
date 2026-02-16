package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/swibrow/howdoi/internal/config"
)

type Anthropic struct {
	client *anthropic.Client
	model  string
}

func NewAnthropic(cfg config.AnthropicConfig) (*Anthropic, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("anthropic API key not set (set ANTHROPIC_API_KEY or configure in ~/.config/howdoi/config.yaml)")
	}

	client := anthropic.NewClient(option.WithAPIKey(cfg.APIKey))

	return &Anthropic{
		client: &client,
		model:  cfg.Model,
	}, nil
}

func (a *Anthropic) Complete(ctx context.Context, systemPrompt, userQuery string) (string, error) {
	resp, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(a.model),
		MaxTokens: 1024,
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userQuery)),
		},
	})
	if err != nil {
		return "", fmt.Errorf("anthropic API error: %w", err)
	}

	var parts []string
	for _, block := range resp.Content {
		if block.Type == "text" {
			parts = append(parts, block.Text)
		}
	}

	return strings.Join(parts, ""), nil
}
