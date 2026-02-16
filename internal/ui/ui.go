package ui

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	commandStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("82"))
	explanationStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	labelStyle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	errorStyle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
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
func ConfirmAndRun(command string) error {
	fmt.Printf("  Run this command? [y/N] ")
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("reading input: %w", err)
	}

	input = strings.TrimSpace(strings.ToLower(input))
	if input != "y" && input != "yes" {
		return nil
	}

	return RunCommand(command)
}

// RunCommand executes a command via the shell.
func RunCommand(command string) error {
	fmt.Println()
	cmd := exec.Command("sh", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}
