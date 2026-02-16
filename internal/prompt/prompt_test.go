package prompt

import (
	"strings"
	"testing"
)

func TestSystemPromptNotEmpty(t *testing.T) {
	if SystemPrompt == "" {
		t.Fatal("SystemPrompt should not be empty")
	}
}

func TestSystemPromptContainsFormat(t *testing.T) {
	if !strings.Contains(SystemPrompt, "COMMAND") {
		t.Error("SystemPrompt should mention COMMAND")
	}
	if !strings.Contains(SystemPrompt, "EXPLANATION") {
		t.Error("SystemPrompt should mention EXPLANATION")
	}
}
