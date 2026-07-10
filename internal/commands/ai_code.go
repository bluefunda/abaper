package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/bluefunda/bluefunda-ai/sdk/agent"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const abapCodeSystemPrompt = `You are an ABAP expert developer. You write, review, and deploy SAP ABAP code.

You can read and edit local ABAP source files. To interact with the SAP system use the abaper CLI via bash:
  abaper deploy --name <NAME> --type <type> --source-file <path>   # upload + activate
  abaper test --name <NAME> --type <type>                          # syntax check
  abaper list objects --package <PACKAGE>                          # list package contents
  abaper search <PATTERN>                                          # search objects

Rules:
- Always uppercase ABAP object names (ZCL_MY_CLASS, ZFM_VALIDATE, etc.)
- Valid types: program, class, interface, function, include, structure, table
- Prefer reading existing source before writing new files
- After writing source, always deploy and check for syntax errors
- DO NOT explain or describe what you are about to do — act immediately with tool calls
- DO NOT show code in prose — write it directly to a file with write_file
- Keep all text responses to one line maximum
- If deploy returns "already exists", append _V2 (then _V3, etc.) to the name and retry
- Interfaces MUST declare "INTERFACE <name> PUBLIC." — omitting PUBLIC deploys and
  activates the object shell fine, then fails with a misleading, unrelated-looking
  "A class already exists with the name X" error. If you see that exact error on an
  interface, the real cause is almost always the missing PUBLIC keyword, not a
  naming conflict — fix the source, do not rename and retry.
- DDIC structures and tables MUST use the curly-brace form: "define structure NAME { ... }"
  / "define table NAME { ... }" with a "}" close — NOT the older dot-terminated
  "define structure NAME.\n...\nend structure." form, which this system rejects.
- DDIC structures (define structure) need BOTH "@EndUserText.label : '...'" AND
  "@AbapCatalog.enhancement.category" annotations or save fails with a generic
  "errors in source; execute check for details" that the syntax-check tool does
  NOT catch.
- DDIC tables (define table) need "@AbapCatalog.tableCategory", "@AbapCatalog.deliveryClass",
  "@AbapCatalog.dataMaintenance", and a "key client : abap.clnt not null;" first field.`

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
	aiCodeCmd.Flags().String("model", "fast", "LLM model alias: auto, fast, think")
	aiCodeCmd.Flags().Int("max-turns", 20, "Maximum agentic loop iterations")
	aiCodeCmd.Flags().String("context-file", "", "Seed the session with an ABAP source file")
	aiCodeCmd.Flags().Bool("verbose", false, "Show every tool call and its output instead of a collapsed progress line")

	aiCmd.AddCommand(aiCodeCmd)
}

func runAICode(cmd *cobra.Command, args []string) error {
	model, _ := cmd.Flags().GetString("model")
	maxTurns, _ := cmd.Flags().GetInt("max-turns")
	contextFile, _ := cmd.Flags().GetString("context-file")
	verbose, _ := cmd.Flags().GetBool("verbose")

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

	// Collapse tool-call noise into a single progress line unless the caller
	// asked for --verbose, or stdout isn't a terminal (piped/redirected output
	// can't overwrite a line in place, so fall back to the plain per-call log).
	collapse := !verbose && term.IsTerminal(int(os.Stdout.Fd()))
	toolCalls := 0
	progressDrawn := false

	clearProgressLine := func() {
		if progressDrawn {
			fmt.Fprint(os.Stderr, "\r\x1b[K")
			progressDrawn = false
		}
	}

	runner := agent.New(agent.Options{
		Model:    model,
		MaxTurns: maxTurns,
		OnEvent: func(ev agent.Event) {
			switch ev.Type {
			case "text":
				clearProgressLine()
				fmt.Print(ev.Text)
			case "tool_use":
				toolCalls++
				if collapse {
					fmt.Fprintf(os.Stderr, "\r\x1b[KWorking… (%d tool call%s)", toolCalls, plural(toolCalls))
					progressDrawn = true
				} else {
					fmt.Fprintf(os.Stderr, "\n[%s] %s\n", ev.ToolName, truncate(ev.ToolInput, 200))
				}
			case "tool_result":
				if !collapse {
					fmt.Fprintf(os.Stderr, "  -> %s\n", truncate(ev.ToolOutput, 200))
				}
			case "result":
				clearProgressLine()
				fmt.Println()
				if ev.StopReason == "max_turns" {
					fmt.Fprintf(os.Stderr, "[max turns reached]\n")
				}
			case "error":
				clearProgressLine()
				fmt.Fprintf(os.Stderr, "\nError: %v\n", ev.Err)
			}
		},
	})
	defer func() { _ = runner.Close() }()

	runner.WithSystemPrompt(abapCodeSystemPrompt)

	if err := runner.Run(ctx, prompt); err != nil {
		clearProgressLine()
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

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func truncate(s string, max int) string {
	s = strings.ReplaceAll(strings.TrimSpace(s), "\n", " ")
	if len(s) <= max {
		return s
	}
	return s[:max] + "… (truncated)"
}
