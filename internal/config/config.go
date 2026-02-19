package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Provider     string          `yaml:"provider"`
	SystemPrompt string          `yaml:"system_prompt,omitempty"`
	Anthropic    AnthropicConfig `yaml:"anthropic"`
	OpenAI       OpenAIConfig    `yaml:"openai"`
	Ollama       OllamaConfig    `yaml:"ollama"`
	Memory       MemoryConfig    `yaml:"memory"`
}

type MemoryConfig struct {
	Enabled bool `yaml:"enabled"`
}

type AnthropicConfig struct {
	APIKey string `yaml:"api_key"`
	Model  string `yaml:"model"`
}

type OpenAIConfig struct {
	APIKey string `yaml:"api_key"`
	Model  string `yaml:"model"`
}

type OllamaConfig struct {
	Model string `yaml:"model"`
	URL   string `yaml:"url"`
}

func DefaultConfig() *Config {
	return &Config{
		Provider: "anthropic",
		Anthropic: AnthropicConfig{
			Model: "claude-sonnet-4-5-20250929",
		},
		OpenAI: OpenAIConfig{
			Model: "gpt-4o",
		},
		Ollama: OllamaConfig{
			Model: "llama3",
			URL:   "http://localhost:11434/v1",
		},
		Memory: MemoryConfig{
			Enabled: true,
		},
	}
}

// ConfigDirFunc overrides the default config directory resolution.
// When nil, the default (~/.config/howdoi) is used.
// Tests set this to redirect config I/O to a temp directory.
var ConfigDirFunc func() (string, error)

func ConfigDir() (string, error) {
	if ConfigDirFunc != nil {
		return ConfigDirFunc()
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}
	return filepath.Join(home, ".config", "howdoi"), nil
}

func configPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

func Load() (*Config, error) {
	cfg := DefaultConfig()

	path, err := configPath()
	if err != nil {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	// Env vars take precedence over config file
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		cfg.Anthropic.APIKey = key
	}
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		cfg.OpenAI.APIKey = key
	}

	return cfg, nil
}

func Save(cfg *Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}

func Show() (string, error) {
	path, err := configPath()
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Sprintf("No config file found. Create one at: %s", path), nil
		}
		return "", fmt.Errorf("reading config: %w", err)
	}

	return fmt.Sprintf("Config file: %s\n\n%s", path, string(data)), nil
}
