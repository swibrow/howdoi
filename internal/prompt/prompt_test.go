package prompt

import (
	"strings"
	"testing"

	"github.com/swibrow/howdoi/internal/memory"
)

func TestSystemPromptNotEmpty(t *testing.T) {
	if SystemPrompt("") == "" {
		t.Fatal("SystemPrompt() should not be empty")
	}
}

func TestSystemPromptContainsFormat(t *testing.T) {
	p := SystemPrompt("")
	if !strings.Contains(p, "COMMAND") {
		t.Error("SystemPrompt should mention COMMAND")
	}
	if !strings.Contains(p, "EXPLANATION") {
		t.Error("SystemPrompt should mention EXPLANATION")
	}
}

func TestSystemPromptContainsOSContext(t *testing.T) {
	p := SystemPrompt("")
	if !strings.Contains(p, "user is on") {
		t.Error("SystemPrompt should contain OS-specific context")
	}
}

func TestSystemPromptCustomOverride(t *testing.T) {
	custom := "You are a helpful DevOps assistant. Respond with COMMAND: and EXPLANATION: format."
	p := SystemPrompt(custom)

	if !strings.Contains(p, "DevOps assistant") {
		t.Error("custom prompt should replace the default base prompt")
	}
	// OS context should still be appended
	if !strings.Contains(p, "user is on") {
		t.Error("OS context should still be appended to custom prompt")
	}
	// Default base prompt content should NOT be present
	if strings.Contains(p, "terminal command expert") {
		t.Error("default base prompt should be replaced by custom prompt")
	}
}

func TestSystemPromptEmptyUsesDefault(t *testing.T) {
	p := SystemPrompt("")
	if !strings.Contains(p, "terminal command expert") {
		t.Error("empty custom prompt should use the default base prompt")
	}
}

func TestFormatMemoryContextEmpty(t *testing.T) {
	result := FormatMemoryContext(nil)
	if result != "" {
		t.Errorf("expected empty string for nil interactions, got %q", result)
	}

	result = FormatMemoryContext([]memory.Interaction{})
	if result != "" {
		t.Errorf("expected empty string for empty interactions, got %q", result)
	}
}

func TestFormatMemoryContextWithInteractions(t *testing.T) {
	interactions := []memory.Interaction{
		{Question: "list files", Command: "ls -la", UseCount: 3},
		{Question: "git status", Command: "git status", UseCount: 1},
	}

	result := FormatMemoryContext(interactions)

	if !strings.Contains(result, "ls -la") {
		t.Error("expected result to contain 'ls -la'")
	}
	if !strings.Contains(result, "git status") {
		t.Error("expected result to contain 'git status'")
	}
	if !strings.Contains(result, "used 3 times") {
		t.Error("expected result to contain use count for ls -la")
	}
	// UseCount 1 should NOT show the "(used X times)" suffix
	if strings.Contains(result, "used 1 times") {
		t.Error("should not show use count for single-use commands")
	}
	if !strings.Contains(result, "Consider these patterns") {
		t.Error("expected result to contain instruction text")
	}
}
