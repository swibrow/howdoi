package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/swibrow/howdoi/internal/config"
	"github.com/swibrow/howdoi/internal/llm"
	"github.com/swibrow/howdoi/internal/prompt"
	"github.com/swibrow/howdoi/internal/ui"
)

var (
	flagYes   bool
	flagQuiet bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "howdoi [question]",
		Short: "Smart terminal cheatsheet — ask a question, get a command",
		Long:  "Ask a natural language question and get back a shell command with explanation.",
		Args:  cobra.MinimumNArgs(1),
		RunE:  run,
	}

	rootCmd.Flags().BoolVarP(&flagYes, "yes", "y", false, "Run the command without confirmation")
	rootCmd.Flags().BoolVarP(&flagQuiet, "quiet", "q", false, "Output only the command (for piping)")

	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Show or manage configuration",
	}

	configShowCmd := &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			output, err := config.Show()
			if err != nil {
				return err
			}
			fmt.Println(output)
			return nil
		},
	}

	configInitCmd := &cobra.Command{
		Use:   "init",
		Short: "Create a default configuration file",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.DefaultConfig()
			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}
			fmt.Println("Default config created at ~/.config/howdoi/config.yaml")
			fmt.Println("Edit it to add your API keys and select a provider.")
			return nil
		},
	}

	configCmd.AddCommand(configShowCmd, configInitCmd)
	rootCmd.AddCommand(configCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	question := strings.Join(args, " ")

	cfg, err := config.Load()
	if err != nil {
		ui.DisplayError(fmt.Sprintf("loading config: %v", err))
		return err
	}

	provider, err := llm.NewProvider(cfg)
	if err != nil {
		ui.DisplayError(fmt.Sprintf("initializing provider: %v", err))
		return err
	}

	ctx := context.Background()
	response, err := provider.Complete(ctx, prompt.SystemPrompt, question)
	if err != nil {
		ui.DisplayError(fmt.Sprintf("LLM request failed: %v", err))
		return err
	}

	result := ui.ParseResponse(response)
	if result.Command == "" {
		ui.DisplayError("could not parse a command from the response")
		return fmt.Errorf("no command in response")
	}

	if flagQuiet {
		ui.DisplayQuiet(result)
		return nil
	}

	ui.Display(result)

	if flagYes {
		return ui.RunCommand(result.Command)
	}

	return ui.ConfirmAndRun(result.Command)
}
