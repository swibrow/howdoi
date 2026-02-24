package prompt

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/swibrow/howdoi/internal/memory"
)

const baseSystemPrompt = `You are a terminal command expert. The user will ask how to do something on the command line. Respond with the most appropriate command and a brief explanation.

You MUST respond in exactly this format:

COMMAND: <the command>
EXPLANATION: <brief one-line explanation>

Rules:
- Give the simplest, most portable command that works on modern systems
- Prefer standard Unix tools (coreutils, grep, sed, awk, jq, curl, etc.)
- If multiple commands are needed, chain them with pipes or && as appropriate
- Do not wrap the command in backticks or code blocks
- Do not include any text outside the COMMAND/EXPLANATION format
- If the question is ambiguous, pick the most common interpretation
- Use placeholder values like <filename> only when the user hasn't specified one AND the value cannot be determined dynamically
- NEVER use placeholders for values that can be resolved from the environment. Use command substitution instead. For example, use $(gh repo view --json nameWithOwner -q .nameWithOwner) instead of <OWNER>/<REPO>, or prefer CLI subcommands that infer context automatically (e.g. gh run list instead of gh api /repos/<OWNER>/<REPO>/actions/runs)
- IMPORTANT: If a command requires the user to choose from a list of inputs (a branch, a file, a process, a container, a pod, etc.), do NOT use placeholders. Instead, construct a pipeline that generates the list and pipes it through fzf for interactive selection, then feeds the selection into the command.

Examples of interactive selection:
- Switching git branch: git branch --format='%(refname:short)' | fzf | xargs git checkout
- Killing a process: ps -eo pid,comm | fzf --header='Select process' | awk '{print $1}' | xargs kill
- Deleting a docker container: docker ps -a --format '{{.Names}}' | fzf | xargs docker rm
- Opening a file: find . -type f | fzf | xargs open
- Checking out a PR: gh pr list | fzf | awk '{print $1}' | xargs gh pr checkout`

// SystemPrompt returns the system prompt with OS-specific context appended.
// If customPrompt is non-empty, it replaces the default base prompt.
func SystemPrompt(customPrompt string) string {
	base := baseSystemPrompt
	if customPrompt != "" {
		base = customPrompt
	}
	osHint := osContext()
	if osHint == "" {
		return base
	}
	return base + "\n- " + osHint
}

// FormatMemoryContext formats past interactions as context for the LLM prompt.
func FormatMemoryContext(interactions []memory.Interaction) string {
	if len(interactions) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("\nThe user has previously run these commands successfully:\n")
	for _, ix := range interactions {
		fmt.Fprintf(&b, "- Q: %s → $ %s", ix.Question, ix.Command)
		if ix.UseCount > 1 {
			fmt.Fprintf(&b, " (used %d times)", ix.UseCount)
		}
		b.WriteString("\n")
	}
	b.WriteString("Consider these patterns when suggesting commands.\n")
	return b.String()
}

func osContext() string {
	switch runtime.GOOS {
	case "darwin":
		return "The user is on macOS. Prefer macOS-compatible tools (e.g. lsof over ss, pbcopy over xclip, open over xdg-open). GNU coreutils may not be installed."
	case "linux":
		return "The user is on Linux. Prefer standard GNU/Linux tools."
	case "windows":
		return "The user is on Windows. Prefer PowerShell or cmd.exe compatible commands."
	default:
		return ""
	}
}
