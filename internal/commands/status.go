package commands

import (
	"fmt"

	"github.com/bluefunda/abaper/internal/client"
	"github.com/bluefunda/abaper/internal/config"
	"github.com/bluefunda/abaper/pkg/output"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show connection and authentication status",
	RunE:  runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	cfg := config.Load()

	status := map[string]any{
		"base_url":      cfg.BaseURL,
		"realm":         cfg.Realm,
		"org":           cfg.Org,
		"authenticated": false,
		"api_reachable": false,
	}

	tokens, err := config.LoadTokens()
	if err == nil && tokens.AccessToken != "" {
		status["authenticated"] = true
	}

	// Check API health
	if tokens != nil && tokens.AccessToken != "" {
		c, err := client.NewClient()
		if err == nil {
			if health, err := c.HealthCheck(); err == nil {
				status["api_reachable"] = true
				status["api_status"] = health["status"]
			}
		}
	}

	outputFmt, _ := cmd.Flags().GetString("output")
	if outputFmt == "json" {
		output.PrintJSON(status)
	} else {
		fmt.Printf("ABAPer CLI Status\n")
		fmt.Printf("  Base URL:       %s\n", cfg.BaseURL)
		fmt.Printf("  Realm:          %s\n", cfg.Realm)
		fmt.Printf("  Organization:   %s\n", cfg.Org)
		fmt.Printf("  Authenticated:  %v\n", status["authenticated"])
		fmt.Printf("  API Reachable:  %v\n", status["api_reachable"])
		if s, ok := status["api_status"]; ok {
			fmt.Printf("  API Status:     %v\n", s)
		}
	}

	return nil
}
