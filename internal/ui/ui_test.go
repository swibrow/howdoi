package ui

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestParseResponse(t *testing.T) {
	response := "COMMAND: ls -la\nEXPLANATION: List all files in long format"
	result := ParseResponse(response)

	if result.Command != "ls -la" {
		t.Errorf("command: got %q, want %q", result.Command, "ls -la")
	}
	if result.Explanation != "List all files in long format" {
		t.Errorf("explanation: got %q, want %q", result.Explanation, "List all files in long format")
	}
}

func TestParseResponseCommandOnly(t *testing.T) {
	response := "COMMAND: git status"
	result := ParseResponse(response)

	if result.Command != "git status" {
		t.Errorf("command: got %q, want %q", result.Command, "git status")
	}
	if result.Explanation != "" {
		t.Errorf("explanation: got %q, want empty", result.Explanation)
	}
}

func TestParseResponseEmpty(t *testing.T) {
	result := ParseResponse("")

	if result.Command != "" {
		t.Errorf("command: got %q, want empty", result.Command)
	}
	if result.Explanation != "" {
		t.Errorf("explanation: got %q, want empty", result.Explanation)
	}
}

func TestParseResponseExtraWhitespace(t *testing.T) {
	response := "  COMMAND:   find . -name '*.go'   \n  EXPLANATION:   Find all Go files   "
	result := ParseResponse(response)

	if result.Command != "find . -name '*.go'" {
		t.Errorf("command: got %q, want %q", result.Command, "find . -name '*.go'")
	}
	if result.Explanation != "Find all Go files" {
		t.Errorf("explanation: got %q, want %q", result.Explanation, "Find all Go files")
	}
}

func TestDisplayQuiet(t *testing.T) {
	result := Result{
		Command:     "echo hello",
		Explanation: "Print hello",
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	DisplayQuiet(result)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "echo hello") {
		t.Errorf("expected 'echo hello' in output, got: %q", output)
	}
	// Quiet mode should not include the explanation
	if strings.Contains(output, "Print hello") {
		t.Error("quiet mode should not include explanation")
	}
}
