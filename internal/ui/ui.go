package ui

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// Catppuccin Mocha palette
var (
	commandStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#a6e3a1")) // Green
	explanationStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#a6adc8"))            // Subtext0
	labelStyle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#f5c2e7")) // Pink
	errorStyle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#f38ba8")) // Red
)

type Result struct {
	Command     string
	Explanation string
}

// ParseResponse extracts command and explanation from the LLM response.
func ParseResponse(response string) Result {
	var result Result

	lines := strings.Split(response, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "COMMAND:") {
			result.Command = strings.TrimSpace(strings.TrimPrefix(line, "COMMAND:"))
		} else if strings.HasPrefix(line, "EXPLANATION:") {
			result.Explanation = strings.TrimSpace(strings.TrimPrefix(line, "EXPLANATION:"))
		}
	}

	// Fallback: if no COMMAND: prefix was found, treat lines before
	// EXPLANATION: (or the entire response) as the command. This handles
	// models that omit the COMMAND: prefix.
	if result.Command == "" {
		var cmdLines []string
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "EXPLANATION:") {
				break
			}
			if trimmed != "" {
				cmdLines = append(cmdLines, trimmed)
			}
		}
		if len(cmdLines) > 0 {
			result.Command = strings.Join(cmdLines, " ")
		}
	}

	result.Command = stripBackticks(result.Command)
	return result
}

// stripBackticks removes backtick wrapping that LLMs sometimes add.
func stripBackticks(cmd string) string {
	switch {
	case strings.HasPrefix(cmd, "```"):
		cmd = strings.TrimPrefix(cmd, "```")
		cmd = strings.TrimSuffix(cmd, "```")
	case strings.HasPrefix(cmd, "`") && strings.HasSuffix(cmd, "`"):
		cmd = cmd[1 : len(cmd)-1]
	case strings.HasPrefix(cmd, "`"):
		cmd = strings.TrimPrefix(cmd, "`")
	}
	return strings.TrimSpace(cmd)
}

// Display shows the formatted result to the user.
func Display(result Result) {
	fmt.Println()
	fmt.Printf("  %s %s\n", labelStyle.Render("$"), commandStyle.Render(result.Command))
	if result.Explanation != "" {
		fmt.Printf("  %s\n", explanationStyle.Render(result.Explanation))
	}
	fmt.Println()
}

// DisplayQuiet shows only the command (for piping).
func DisplayQuiet(result Result) {
	fmt.Println(result.Command)
}

// DisplayError shows a formatted error message.
func DisplayError(msg string) {
	fmt.Fprintf(os.Stderr, "\n  %s %s\n\n", errorStyle.Render("Error:"), msg)
}

// ConfirmAndRun prompts the user to run the command and executes it.
// Returns (true, nil) if confirmed and succeeded, (true, err) if confirmed
// but the command failed, and (false, nil) if the user declined.
func ConfirmAndRun(command string) (bool, error) {
	fmt.Printf("  Run this command? [y/N] ")

	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		// Not a terminal (e.g. piped input) — can't use raw mode
		return false, nil
	}

	var buf [1]byte
	_, err = os.Stdin.Read(buf[:])
	_ = term.Restore(fd, oldState)
	fmt.Println() // move to next line after the keypress

	if err != nil {
		return false, fmt.Errorf("reading input: %w", err)
	}

	if buf[0] != 'y' && buf[0] != 'Y' {
		return false, nil
	}

	return true, RunCommand(command)
}

// RunCommand executes a command via the shell.
// If the command is not found (exit code 127), it suggests how to install it.
func RunCommand(command string) error {
	fmt.Println()
	cmd := exec.Command("sh", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin

	var stderrBuf bytes.Buffer
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)

	err := cmd.Run()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 127 {
			cmdName := parseNotFoundCommand(stderrBuf.String(), command)
			if cmdName != "" {
				fmt.Fprintln(os.Stderr)
				fmt.Fprintf(os.Stderr, "  %s %s is not installed.\n", hintStyle.Render("Hint:"), cmdName)
				fmt.Fprintf(os.Stderr, "  %s\n", installSuggestion(cmdName))
			}
		}
	} else {
		addToShellHistory(command)
	}
	return err
}

// addToShellHistory appends the command to the user's shell history file.
func addToShellHistory(command string) {
	shell := os.Getenv("SHELL")
	histFile := shellHistoryFile(shell)
	if histFile == "" {
		return
	}

	f, err := os.OpenFile(histFile, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close() //nolint:errcheck

	if strings.Contains(shell, "zsh") && isZshExtendedHistory(histFile) {
		_, _ = fmt.Fprintf(f, ": %d:0;%s\n", time.Now().Unix(), command)
	} else {
		_, _ = fmt.Fprintf(f, "%s\n", command)
	}
}

// shellHistoryFile returns the path to the shell history file,
// using $HISTFILE if set, otherwise falling back to shell-specific defaults.
func shellHistoryFile(shell string) string {
	if histFile := os.Getenv("HISTFILE"); histFile != "" {
		return histFile
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	switch {
	case strings.Contains(shell, "zsh"):
		return filepath.Join(home, ".zsh_history")
	case strings.Contains(shell, "bash"):
		return filepath.Join(home, ".bash_history")
	default:
		return ""
	}
}

// isZshExtendedHistory checks whether the history file uses zsh extended
// history format (": timestamp:duration;command") by sampling the tail.
func isZshExtendedHistory(histFile string) bool {
	f, err := os.Open(histFile)
	if err != nil {
		return false
	}
	defer f.Close() //nolint:errcheck

	info, err := f.Stat()
	if err != nil || info.Size() == 0 {
		return false
	}

	offset := info.Size() - 1024
	if offset < 0 {
		offset = 0
	}

	buf := make([]byte, 1024)
	n, err := f.ReadAt(buf, offset)
	if err != nil && n == 0 {
		return false
	}

	return zshExtendedRe.Match(buf[:n])
}

var zshExtendedRe = regexp.MustCompile(`(?m)^: \d+:\d+;`)

var (
	hintStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#f9e2af")) // Yellow

	// Matches patterns like "sh: ss: command not found" or "bash: ss: command not found"
	notFoundRe = regexp.MustCompile(`(?:sh|bash):\s*(?:line \d+:\s*)?(\S+):\s*(?:command )?not found`)
	// Matches zsh pattern: "zsh: command not found: ss"
	notFoundZshRe = regexp.MustCompile(`zsh:\s*command not found:\s*(\S+)`)
)

// parseNotFoundCommand extracts the missing command name from shell stderr output.
// Falls back to the first token of the original command.
func parseNotFoundCommand(stderr, command string) string {
	if m := notFoundRe.FindStringSubmatch(stderr); len(m) > 1 {
		return m[1]
	}
	if m := notFoundZshRe.FindStringSubmatch(stderr); len(m) > 1 {
		return m[1]
	}
	// Fallback: first token of the command
	if fields := strings.Fields(command); len(fields) > 0 {
		return fields[0]
	}
	return ""
}

// shellBuiltins is the set of shell builtins that should be skipped during
// command validation since they won't be found by LookPath.
var shellBuiltins = map[string]bool{
	"cd": true, "echo": true, "export": true, "source": true,
	"test": true, "[": true, "set": true, "unset": true,
	"read": true, "return": true, "exit": true, "exec": true,
	"eval": true, "trap": true, "shift": true, "wait": true,
	"true": true, "false": true, "for": true, "while": true,
	"do": true, "done": true, "if": true, "then": true,
	"else": true, "fi": true, "case": true, "esac": true,
	"function": true, "select": true, "until": true, "time": true,
	"printf": true, "type": true, "alias": true, "unalias": true,
	"builtin": true, "command": true, "declare": true, "local": true,
	"readonly": true, "typeset": true, "ulimit": true, "umask": true,
}

// splitShellOperators is a regex that splits on |, &&, ||, and ;
// but not on | inside $() or quotes (best-effort for simple cases).
var splitShellOperators = regexp.MustCompile(`\s*(?:\|\||&&|[|;])\s*`)

// ValidateCommand extracts base command names from a shell command string
// and checks whether each exists on the system. Returns names of missing commands.
func ValidateCommand(command string) []string {
	segments := splitShellOperators.Split(command, -1)

	seen := make(map[string]bool)
	var missing []string

	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}

		cmdName := extractBaseCommand(seg)
		if cmdName == "" || shellBuiltins[cmdName] || seen[cmdName] {
			continue
		}
		seen[cmdName] = true

		if _, err := exec.LookPath(cmdName); err != nil {
			missing = append(missing, cmdName)
		}
	}
	return missing
}

// extractBaseCommand gets the executable name from a shell segment,
// skipping leading env var assignments (e.g. FOO=bar cmd).
func extractBaseCommand(segment string) string {
	for f := range strings.FieldsSeq(segment) {
		// Skip env var assignments like FOO=bar
		if strings.Contains(f, "=") && !strings.HasPrefix(f, "-") {
			continue
		}
		// Skip subshell prefixes
		f = strings.TrimLeft(f, "(")
		if f == "" {
			continue
		}
		return f
	}
	return ""
}

// DisplayWarnings prints yellow warnings for missing commands.
func DisplayWarnings(missing []string) {
	for _, cmd := range missing {
		fmt.Fprintf(os.Stderr, "  %s '%s' not found. The suggested command may be incorrect.\n",
			hintStyle.Render("Warning:"), cmd)
	}
}

// installSuggestion returns a platform-aware install hint.
func installSuggestion(cmdName string) string {
	switch runtime.GOOS {
	case "darwin":
		return fmt.Sprintf("Install with: brew install %s", cmdName)
	case "linux":
		if _, err := exec.LookPath("apt"); err == nil {
			return fmt.Sprintf("Install with: sudo apt install %s", cmdName)
		}
		if _, err := exec.LookPath("dnf"); err == nil {
			return fmt.Sprintf("Install with: sudo dnf install %s", cmdName)
		}
		if _, err := exec.LookPath("pacman"); err == nil {
			return fmt.Sprintf("Install with: sudo pacman -S %s", cmdName)
		}
		return fmt.Sprintf("Install %s using your system package manager", cmdName)
	default:
		return fmt.Sprintf("Install %s using your system package manager", cmdName)
	}
}
