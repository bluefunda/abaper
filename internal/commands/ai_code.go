package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/bluefunda/bluefunda-ai/sdk/agent"
	"github.com/spf13/cobra"
)

const abapCodeSystemPrompt = `You are an ABAP expert developer. You write, review, and deploy SAP ABAP code.

You can read and edit local ABAP source files. To interact with the SAP system use the abaper CLI via bash:
  abaper deploy --name <NAME> --type <type> --source-file <path>   # upload + activate
  abaper test --name <NAME> --type <type>                          # syntax check
  abaper list objects --package <PACKAGE>                          # list package contents
  abaper search <PATTERN>                                          # search objects

Rules:
- Always uppercase ABAP object names (ZCL_MY_CLASS, ZFM_VALIDATE, etc.)
- Valid types: program, class, interface, function, include
- Prefer reading existing source before writing new files
- After writing source, always deploy and check for syntax errors`

var aiCodeCmd = &cobra.Command{
	Use:   "code [prompt]",
	Short: "Agentic ABAP coding assistant — reads, writes, deploys",
	Long: `Run an agentic loop that can read and edit local ABAP source files
and deploy them to SAP via the abaper CLI. The agent iterates until the
task is complete or the turn limit is reached.`,
	Args: cobra.MinimumNArgs(1),
	RunE: runAICode,
}

func init() {
	aiCodeCmd.Flags().String("model", "auto", "LLM model alias: auto, fast, think")
	aiCodeCmd.Flags().Int("max-turns", 20, "Maximum agentic loop iterations")
	aiCodeCmd.Flags().String("context-file", "", "Seed the session with an ABAP source file")

	aiCmd.AddCommand(aiCodeCmd)
}

func runAICode(cmd *cobra.Command, args []string) error {
	model, _ := cmd.Flags().GetString("model")
	maxTurns, _ := cmd.Flags().GetInt("max-turns")
	contextFile, _ := cmd.Flags().GetString("context-file")

	prompt := strings.Join(args, " ")

	if contextFile != "" {
		data, err := os.ReadFile(contextFile)
		if err != nil {
			return fmt.Errorf("read context file: %w", err)
		}
		prompt = fmt.Sprintf("%s\n\nSource file (%s):\n```abap\n%s\n```", prompt, contextFile, string(data))
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	runner := agent.New(agent.Options{
		Model:    model,
		MaxTurns: maxTurns,
		OnEvent: func(ev agent.Event) {
			switch ev.Type {
			case "text":
				fmt.Print(ev.Text)
			case "tool_use":
				fmt.Fprintf(os.Stderr, "\n[%s]\n", ev.ToolName)
			case "result":
				fmt.Println()
				if ev.StopReason == "max_turns" {
					fmt.Fprintf(os.Stderr, "[max turns reached]\n")
				}
			case "error":
				fmt.Fprintf(os.Stderr, "\nError: %v\n", ev.Err)
			}
		},
	})
	defer func() { _ = runner.Close() }()

	runner.WithSystemPrompt(abapCodeSystemPrompt)

	if err := runner.Run(ctx, prompt); err != nil {
		if ctx.Err() != nil {
			return nil // user cancelled
		}
		if strings.Contains(err.Error(), "not signed in") {
			return fmt.Errorf("%w\n\nRun: bai login", err)
		}
		return err
	}
	return nil
}
