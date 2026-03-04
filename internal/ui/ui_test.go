package ui

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"runtime"
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

func TestParseResponseStripsBackticks(t *testing.T) {
	cases := []struct {
		name     string
		response string
		wantCmd  string
	}{
		{
			name:     "single backticks",
			response: "COMMAND: `ls -la`\nEXPLANATION: List files",
			wantCmd:  "ls -la",
		},
		{
			name:     "triple backticks",
			response: "COMMAND: ```ls -la```\nEXPLANATION: List files",
			wantCmd:  "ls -la",
		},
		{
			name:     "leading backtick only",
			response: "COMMAND: `gh api -X GET /repos/owner/repo/actions\nEXPLANATION: Get actions",
			wantCmd:  "gh api -X GET /repos/owner/repo/actions",
		},
		{
			name:     "no backticks unchanged",
			response: "COMMAND: ls -la\nEXPLANATION: List files",
			wantCmd:  "ls -la",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := ParseResponse(tc.response)
			if result.Command != tc.wantCmd {
				t.Errorf("command: got %q, want %q", result.Command, tc.wantCmd)
			}
		})
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

func TestParseNotFoundCommandBash(t *testing.T) {
	cases := []struct {
		name    string
		stderr  string
		command string
		want    string
	}{
		{
			name:    "bash style",
			stderr:  "sh: ss: command not found\n",
			command: "ss -tuln",
			want:    "ss",
		},
		{
			name:    "bash with line number",
			stderr:  "bash: line 1: htop: command not found\n",
			command: "htop",
			want:    "htop",
		},
		{
			name:    "zsh style",
			stderr:  "zsh: command not found: rg\n",
			command: "rg foo",
			want:    "rg",
		},
		{
			name:    "fallback to first token",
			stderr:  "some unexpected error\n",
			command: "nonexistent --flag",
			want:    "nonexistent",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseNotFoundCommand(tc.stderr, tc.command)
			if got != tc.want {
				t.Errorf("parseNotFoundCommand(%q, %q) = %q, want %q", tc.stderr, tc.command, got, tc.want)
			}
		})
	}
}

func TestInstallSuggestion(t *testing.T) {
	suggestion := installSuggestion("ripgrep")

	switch runtime.GOOS {
	case "darwin":
		if !strings.Contains(suggestion, "brew install ripgrep") {
			t.Errorf("expected brew suggestion on macOS, got: %s", suggestion)
		}
	case "linux":
		if !strings.Contains(suggestion, "ripgrep") {
			t.Errorf("expected ripgrep in suggestion, got: %s", suggestion)
		}
	default:
		if !strings.Contains(suggestion, "ripgrep") {
			t.Errorf("expected ripgrep in suggestion, got: %s", suggestion)
		}
	}
}

func TestShellHistoryFile(t *testing.T) {
	cases := []struct {
		name     string
		shell    string
		histFile string
		wantEnd  string // expected suffix of the result
	}{
		{name: "zsh default", shell: "/bin/zsh", histFile: "", wantEnd: ".zsh_history"},
		{name: "bash default", shell: "/bin/bash", histFile: "", wantEnd: ".bash_history"},
		{name: "HISTFILE override", shell: "/bin/zsh", histFile: "/tmp/my_history", wantEnd: "/tmp/my_history"},
		{name: "unsupported shell", shell: "/bin/fish", histFile: "", wantEnd: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("SHELL", tc.shell)
			t.Setenv("HISTFILE", tc.histFile)

			got := shellHistoryFile(tc.shell)
			if tc.wantEnd == "" {
				if got != "" {
					t.Errorf("shellHistoryFile(%q) = %q, want empty", tc.shell, got)
				}
			} else if !strings.HasSuffix(got, tc.wantEnd) {
				t.Errorf("shellHistoryFile(%q) = %q, want suffix %q", tc.shell, got, tc.wantEnd)
			}
		})
	}
}

func TestAddToShellHistory(t *testing.T) {
	// Create a temp file to use as the history file
	tmpFile, err := os.CreateTemp(t.TempDir(), "history")
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	t.Setenv("SHELL", "/bin/bash")
	t.Setenv("HISTFILE", tmpFile.Name())

	addToShellHistory("echo hello")

	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "echo hello\n") {
		t.Errorf("history file should contain 'echo hello', got: %q", string(data))
	}
}

func TestAddToShellHistoryZshExtended(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "zsh_history")
	if err != nil {
		t.Fatal(err)
	}
	// Write extended history format to the file
	fmt.Fprintf(tmpFile, ": 1700000000:0;ls -la\n")
	tmpFile.Close()

	t.Setenv("SHELL", "/bin/zsh")
	t.Setenv("HISTFILE", tmpFile.Name())

	addToShellHistory("git status")

	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, ":0;git status\n") {
		t.Errorf("expected extended zsh history entry, got: %q", content)
	}
}

func TestIsZshExtendedHistory(t *testing.T) {
	t.Run("extended format", func(t *testing.T) {
		f, _ := os.CreateTemp(t.TempDir(), "hist")
		fmt.Fprintf(f, ": 1700000000:0;ls\n: 1700000001:0;pwd\n")
		f.Close()
		if !isZshExtendedHistory(f.Name()) {
			t.Error("expected true for extended history format")
		}
	})

	t.Run("plain format", func(t *testing.T) {
		f, _ := os.CreateTemp(t.TempDir(), "hist")
		fmt.Fprintf(f, "ls\npwd\n")
		f.Close()
		if isZshExtendedHistory(f.Name()) {
			t.Error("expected false for plain history format")
		}
	})

	t.Run("empty file", func(t *testing.T) {
		f, _ := os.CreateTemp(t.TempDir(), "hist")
		f.Close()
		if isZshExtendedHistory(f.Name()) {
			t.Error("expected false for empty file")
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		if isZshExtendedHistory("/tmp/nonexistent_history_file_xyz") {
			t.Error("expected false for nonexistent file")
		}
	})
}

func TestValidateCommand(t *testing.T) {
	cases := []struct {
		name        string
		command     string
		wantMissing bool // true if we expect at least one missing command
	}{
		{
			name:        "simple existing command",
			command:     "ls -la",
			wantMissing: false,
		},
		{
			name:        "nonexistent command",
			command:     "nonexistent_cmd_xyz123 --flag",
			wantMissing: true,
		},
		{
			name:        "piped with nonexistent",
			command:     "nonexistent_cmd_xyz123 | grep foo",
			wantMissing: true,
		},
		{
			name:        "chained with builtin",
			command:     "echo hello && cd /tmp",
			wantMissing: false,
		},
		{
			name:        "env var prefix",
			command:     "FOO=bar ls -la",
			wantMissing: false,
		},
		{
			name:        "all builtins",
			command:     "echo hello | export FOO=bar",
			wantMissing: false,
		},
		{
			name:        "empty command",
			command:     "",
			wantMissing: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			missing := ValidateCommand(tc.command)
			if tc.wantMissing && len(missing) == 0 {
				t.Errorf("expected missing commands for %q, got none", tc.command)
			}
			if !tc.wantMissing && len(missing) > 0 {
				t.Errorf("expected no missing commands for %q, got %v", tc.command, missing)
			}
		})
	}
}

func TestExtractBaseCommand(t *testing.T) {
	cases := []struct {
		name    string
		segment string
		want    string
	}{
		{name: "simple", segment: "ls -la", want: "ls"},
		{name: "env var prefix", segment: "FOO=bar git commit", want: "git"},
		{name: "multiple env vars", segment: "A=1 B=2 make build", want: "make"},
		{name: "subshell prefix", segment: "(cd /tmp && ls)", want: "cd"},
		{name: "empty", segment: "", want: ""},
		{name: "only env var", segment: "FOO=bar", want: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractBaseCommand(tc.segment)
			if got != tc.want {
				t.Errorf("extractBaseCommand(%q) = %q, want %q", tc.segment, got, tc.want)
			}
		})
	}
}

func TestValidateCommandDeduplicates(t *testing.T) {
	missing := ValidateCommand("nonexistent_xyz123 | nonexistent_xyz123")
	if len(missing) != 1 {
		t.Errorf("expected 1 unique missing command, got %d: %v", len(missing), missing)
	}
}

func TestRunCommandNotFound(t *testing.T) {
	// Capture stderr to verify the hint is printed
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := RunCommand("this_command_does_not_exist_xyz123")

	w.Close()
	os.Stderr = oldStderr

	if err == nil {
		t.Fatal("expected error for nonexistent command")
	}

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "not installed") {
		t.Errorf("expected 'not installed' hint in stderr, got: %q", output)
	}
}
