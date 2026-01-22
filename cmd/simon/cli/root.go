package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/felixgeelhaar/simon/internal/guard"
	"github.com/felixgeelhaar/simon/internal/observe"
	"github.com/felixgeelhaar/simon/internal/provider"
	"github.com/felixgeelhaar/simon/internal/store"
	"github.com/felixgeelhaar/simon/internal/ui"
	"github.com/felixgeelhaar/simon/internal/ui/tui"
	"github.com/spf13/cobra"
)

var (
	specPath     string
	verbose      bool
	providerType string
	modelName    string
	useCLI       bool
	useAPI       bool
	ciMode       bool
	interactive  bool
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "simon",
	Short: "AI Agent Governance Runtime",
	Long: `Simon enforces clarity, discipline, and resource limits on AI agent execution.
It acts as a runtime layer between your intent and the AI provider.`,
}

var runCmd = &cobra.Command{
	Use:   "run [spec-file]",
	Short: "Execute a task defined in a spec file",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		specPath = args[0]
		runSession()
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List past sessions",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Listing sessions feature pending...")
	},
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	RootCmd.AddCommand(runCmd)
	RootCmd.AddCommand(listCmd)
	runCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
	runCmd.Flags().StringVarP(&providerType, "provider", "p", "ollama", "AI Provider (ollama, openai, gemini, anthropic)")
	runCmd.Flags().StringVarP(&modelName, "model", "m", "", "Model name (default depends on provider)")
	runCmd.Flags().BoolVar(&useCLI, "cli", false, "Use local CLI tool as provider if available")
	runCmd.Flags().BoolVar(&useAPI, "api", false, "Use direct API integration (default)")
	runCmd.Flags().BoolVar(&ciMode, "ci", false, "CI mode: JSON output, non-interactive")
	runCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Start interactive TUI")
}

func runSession() {
	// Initialize Observer
	var obs *observe.Observer
	if ciMode {
		obs = observe.NewJSON(os.Stdout, verbose)
	} else {
		obs = observe.New(os.Stdout, verbose)
	}
	defer obs.Close()

	// Initialize Store
	home, _ := os.UserHomeDir()
	simonDir := filepath.Join(home, ".simon")
	storeLayer, err := store.NewSQLiteStore(
		filepath.Join(simonDir, "metadata.db"),
		filepath.Join(simonDir, "artifacts"),
	)
	if err != nil {
		obs.Log().Fatal().Err(err).Msg("Failed to init store")
	}
	defer storeLayer.Close()

	// Initialize Provider
	var p provider.Provider
	var pErr error

	if useCLI {
		p, pErr = detectCLIProvider(storeLayer)
		if pErr != nil {
			obs.Log().Fatal().Err(pErr).Msg("Failed to initialize CLI provider")
		}
	} else {
		switch providerType {
		case "openai":
			apiKey, _ := storeLayer.GetConfig("openai.api_key")
			baseURL, _ := storeLayer.GetConfig("openai.base_url")
			p, pErr = provider.NewOpenAIProvider(apiKey, baseURL, modelName)
		case "ollama":
			p, pErr = provider.NewOllamaProvider(modelName)
		case "gemini":
			apiKey, _ := storeLayer.GetConfig("gemini.api_key")
			p, pErr = provider.NewGeminiProvider(apiKey, modelName)
		case "anthropic":
			apiKey, _ := storeLayer.GetConfig("anthropic.api_key")
			p, pErr = provider.NewAnthropicProvider(apiKey, modelName)
		default:
			obs.Log().Fatal().Str("provider", providerType).Msg("Unknown provider")
		}
	}

	if pErr != nil {
		obs.Log().Fatal().Err(pErr).Msg("Failed to initialize provider")
	}

	var u ui.UI
	if interactive {
		model := tui.NewModel("Simon execution", guard.DefaultPolicy.MaxIterations)
		program := tea.NewProgram(model)
		u = tui.NewTUI(program)
		
		go func() {
			runner := NewRunner(obs, storeLayer, p, specPath, u)
			_ = runner.Run(context.Background())
			program.Quit()
		}()

		if _, err := program.Run(); err != nil {
			fmt.Printf("Alas, there's been an error: %v", err)
			os.Exit(1)
		}
	} else {
		runner := NewRunner(obs, storeLayer, p, specPath, nil)
		if err := runner.Run(context.Background()); err != nil {
			os.Exit(1)
		}
	}
}

func detectCLIProvider(s store.Storage) (provider.Provider, error) {
	// 1. Check config first
	cliPath, _ := s.GetConfig("provider.cli.path")
	if cliPath != "" {
		return provider.NewCLIProvider(cliPath, []string{})
	}

	// 2. Auto-detect common tools
	tools := []string{"claude", "codex", "gemini", "llm"}
	for _, t := range tools {
		path, err := exec.LookPath(t)
		if err == nil {
			return provider.NewCLIProvider(path, []string{})
		}
	}

	return nil, fmt.Errorf("no local CLI agents detected (tried claude, codex, gemini, llm)")
}