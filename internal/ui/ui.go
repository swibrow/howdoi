package ui

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

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

	for _, line := range strings.Split(response, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "COMMAND:") {
			result.Command = strings.TrimSpace(strings.TrimPrefix(line, "COMMAND:"))
		} else if strings.HasPrefix(line, "EXPLANATION:") {
			result.Explanation = strings.TrimSpace(strings.TrimPrefix(line, "EXPLANATION:"))
		}
	}

	return result
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
// Reads a single keypress without requiring Enter.
func ConfirmAndRun(command string) error {
	fmt.Printf("  Run this command? [y/N] ")

	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		// Not a terminal (e.g. piped input) — can't use raw mode
		return nil
	}

	var buf [1]byte
	_, err = os.Stdin.Read(buf[:])
	term.Restore(fd, oldState)
	fmt.Println() // move to next line after the keypress

	if err != nil {
		return fmt.Errorf("reading input: %w", err)
	}

	if buf[0] != 'y' && buf[0] != 'Y' {
		return nil
	}

	return RunCommand(command)
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
	}
	return err
}

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
