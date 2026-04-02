package commands

import (
	"github.com/bluefunda/abaper-cli/internal/client"
	"github.com/bluefunda/abaper-cli/internal/config"
	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "abaper",
	Short: "ABAPer CLI — interact with ABAPer APIs from the command line",
	Long: `ABAPer CLI is a command line interface for the ABAPer platform.
It communicates with ABAPer APIs exposed through the ABAPer gateway
to support developer workflows including code generation, compilation,
deployment, and inspection.`,
	SilenceUsage: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		go client.Track("cli_invoked", map[string]string{"command": cmd.Name()})
	},
}

func init() {
	cobra.OnInitialize(config.Init)

	rootCmd.PersistentFlags().String("base-url", "", "ABAPer API base URL (default: https://api.bluefunda.com)")
	rootCmd.PersistentFlags().String("realm", "", "Keycloak realm (default: trm)")
	rootCmd.PersistentFlags().StringP("output", "o", "text", "Output format: text, json")

	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(signupCmd)
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(aiCmd)
	rootCmd.AddCommand(testCmd)
	rootCmd.AddCommand(listCmd)
}

func Execute() error {
	return rootCmd.Execute()
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the CLI version",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Printf("abaper version %s\n", version)
	},
}
