package llm

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/swibrow/howdoi/internal/config"
)

type Ollama struct {
	client *openai.Client
	model  string
}

func NewOllama(cfg config.OllamaConfig) (*Ollama, error) {
	client := openai.NewClient(
		option.WithBaseURL(cfg.URL),
		option.WithAPIKey("ollama"), // Ollama doesn't need a real key
	)

	return &Ollama{
		client: &client,
		model:  cfg.Model,
	}, nil
}

func (o *Ollama) Complete(ctx context.Context, systemPrompt, userQuery string) (string, error) {
	resp, err := o.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: o.model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(userQuery),
		},
	})
	if err != nil {
		return "", fmt.Errorf("ollama API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("ollama returned no choices")
	}

	return resp.Choices[0].Message.Content, nil
}
