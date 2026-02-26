package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/swibrow/how/internal/config"
	"github.com/swibrow/how/internal/llm"
	"github.com/swibrow/how/internal/memory"
	"github.com/swibrow/how/internal/prompt"
	"github.com/swibrow/how/internal/ui"
)

var (
	flagYes   bool
	flagQuiet bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:           "how [question]",
		Short:         "Smart terminal cheatsheet — ask a question, get a command",
		Long:          "Ask a natural language question and get back a shell command with explanation.",
		Args:          cobra.MinimumNArgs(1),
		RunE:          run,
		SilenceUsage:  true,
		SilenceErrors: true,
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
			fmt.Println("Default config created at ~/.config/how/config.yaml")
			fmt.Println("Edit it to add your API keys and select a provider.")
			return nil
		},
	}

	memoryCmd := &cobra.Command{
		Use:   "memory",
		Short: "Manage command memory",
	}

	memoryListCmd := &cobra.Command{
		Use:   "list",
		Short: "List remembered commands",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openMemoryStore()
			if err != nil {
				return err
			}
			defer store.Close() //nolint:errcheck

			interactions, err := store.List(context.Background(), 20)
			if err != nil {
				return fmt.Errorf("listing memory: %w", err)
			}

			if len(interactions) == 0 {
				fmt.Println("No remembered commands yet.")
				return nil
			}

			for _, ix := range interactions {
				fmt.Printf("  Q: %s\n  $ %s\n", ix.Question, ix.Command)
				if ix.UseCount > 1 {
					fmt.Printf("  (used %d times)\n", ix.UseCount)
				}
				fmt.Println()
			}
			return nil
		},
	}

	memoryClearCmd := &cobra.Command{
		Use:   "clear",
		Short: "Clear all remembered commands",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openMemoryStore()
			if err != nil {
				return err
			}
			defer store.Close() //nolint:errcheck

			if err := store.Clear(context.Background()); err != nil {
				return fmt.Errorf("clearing memory: %w", err)
			}
			fmt.Println("Memory cleared.")
			return nil
		},
	}

	memoryCmd.AddCommand(memoryListCmd, memoryClearCmd)
	configCmd.AddCommand(configShowCmd, configInitCmd)
	rootCmd.AddCommand(configCmd, memoryCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func openMemoryStore() (*memory.Store, error) {
	dir, err := config.ConfigDir()
	if err != nil {
		return nil, fmt.Errorf("config directory: %w", err)
	}
	store, err := memory.Open(dir)
	if err != nil {
		return nil, fmt.Errorf("opening memory: %w", err)
	}
	return store, nil
}

func run(cmd *cobra.Command, args []string) error {
	question := strings.Join(args, " ")

	cfg, err := config.Load()
	if err != nil {
		ui.DisplayError(fmt.Sprintf("loading config: %v", err))
		return err
	}

	// Open memory store (non-fatal on failure)
	var store *memory.Store
	if cfg.Memory.Enabled {
		store, err = openMemoryStore()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: memory disabled: %v\n", err)
		} else {
			defer store.Close() //nolint:errcheck
		}
	}

	// Build system prompt, enriching with memory context if available
	ctx := context.Background()
	sysPrompt := prompt.SystemPrompt(cfg.SystemPrompt)
	if store != nil {
		if past, err := store.Search(ctx, question, 10); err == nil && len(past) > 0 {
			sysPrompt += prompt.FormatMemoryContext(past)
		}
	}

	provider, err := llm.NewProvider(cfg)
	if err != nil {
		ui.DisplayError(fmt.Sprintf("initializing provider: %v", err))
		return err
	}

	response, err := provider.Complete(ctx, sysPrompt, question)
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
		err := ui.RunCommand(result.Command)
		if err == nil && store != nil {
			_ = store.Save(ctx, question, result.Command, result.Explanation)
		}
		return err
	}

	confirmed, err := ui.ConfirmAndRun(result.Command)
	if confirmed && err == nil && store != nil {
		_ = store.Save(ctx, question, result.Command, result.Explanation)
	}
	return err
}
