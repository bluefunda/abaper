package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/bluefunda/abaper-cli/internal/client"
	"github.com/bluefunda/abaper-cli/pkg/output"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var aiCmd = &cobra.Command{
	Use:   "ai",
	Short: "AI-powered developer assistance",
	Long:  "Interact with ABAPer AI features including chat and code analysis.",
}

var aiChatCmd = &cobra.Command{
	Use:   "chat [prompt]",
	Short: "Start an AI chat session",
	Long: `Send a prompt to the ABAPer AI assistant and stream the response.
Optionally include ABAP source context from a file.`,
	Args: cobra.MinimumNArgs(1),
	RunE: runAIChat,
}

func init() {
	aiChatCmd.Flags().String("model", "groq", "LLM model to use")
	aiChatCmd.Flags().String("context-file", "", "ABAP source file to include as context")
	aiChatCmd.Flags().String("chat-id", "", "Resume an existing chat session (default: new session)")

	aiCmd.AddCommand(aiChatCmd)
}

func runAIChat(cmd *cobra.Command, args []string) error {
	prompt := args[0]
	model, _ := cmd.Flags().GetString("model")
	contextFile, _ := cmd.Flags().GetString("context-file")
	chatID, _ := cmd.Flags().GetString("chat-id")

	if contextFile != "" {
		data, err := os.ReadFile(contextFile)
		if err != nil {
			return fmt.Errorf("read context file: %w", err)
		}
		prompt = fmt.Sprintf("%s\n\nContext:\n```abap\n%s\n```", prompt, string(data))
	}

	isNewChat := chatID == ""
	if isNewChat {
		chatID = uuid.New().String()
	}

	go client.Track("chat_opened", nil)

	c, err := client.NewClient()
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	outputFmt, _ := cmd.Flags().GetString("output")

	req := client.ChatRequest{
		Prompt:    prompt,
		Model:     model,
		AgentName: "abaper",
		IsNewChat: isNewChat,
	}

	if outputFmt == "json" {
		var fullContent string
		var events []client.ChatEvent
		err = c.StreamChat(ctx, chatID, req, func(event client.ChatEvent) {
			events = append(events, event)
			if event.Content != "" {
				fullContent += event.Content
			}
			if event.FullContent != "" {
				fullContent = event.FullContent
			}
		})
		if err != nil {
			return fmt.Errorf("chat failed: %w", err)
		}
		output.PrintJSON(map[string]any{
			"chat_id": chatID,
			"content": fullContent,
			"events":  events,
		})
	} else {
		err = c.StreamChat(ctx, chatID, req, func(event client.ChatEvent) {
			switch event.Type {
			case "stream_chunk":
				fmt.Print(event.Content)
			case "stream_end":
				fmt.Println()
			case "stream_tool_execution":
				fmt.Fprintf(os.Stderr, "\n[tool: %s — %s]\n", event.ToolName, event.Status)
			case "stream_progress":
				fmt.Fprintf(os.Stderr, "[thinking...]\n")
			case "error", "stream_error":
				msg := event.Error
				if msg == "" {
					msg = event.Message
				}
				fmt.Fprintf(os.Stderr, "\nError: %s\n", msg)
			}
		})
		if err != nil {
			return fmt.Errorf("chat failed: %w", err)
		}
		fmt.Fprintf(os.Stderr, "\nchat-id: %s\n", chatID)
	}

	return nil
}
