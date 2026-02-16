package llm

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/swibrow/howdoi/internal/config"
)

type OpenAI struct {
	client *openai.Client
	model  string
}

func NewOpenAI(cfg config.OpenAIConfig) (*OpenAI, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("openai API key not set (set OPENAI_API_KEY or configure in ~/.config/howdoi/config.yaml)")
	}

	client := openai.NewClient(option.WithAPIKey(cfg.APIKey))

	return &OpenAI{
		client: &client,
		model:  cfg.Model,
	}, nil
}

func (o *OpenAI) Complete(ctx context.Context, systemPrompt, userQuery string) (string, error) {
	resp, err := o.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: o.model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(userQuery),
		},
	})
	if err != nil {
		return "", fmt.Errorf("openai API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("openai returned no choices")
	}

	return resp.Choices[0].Message.Content, nil
}
