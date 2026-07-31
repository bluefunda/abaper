package commands

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/bluefunda/abaper/internal/client"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Sign in (delegates to bai login — shared auth for AI and ABAPer gateway features)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runBai("login")
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Sign out (delegates to bai logout — clears the shared credential store)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runBai("logout")
	},
}

// runBai delegates to `bai login`/`bai logout`. abaper authenticates against
// the same Keycloak realm and OAuth client as bai (see
// internal/config.ClientID) and reads bai's stored credentials
// (~/.bai/config.yaml + OS keychain) directly, so a single bai login is
// sufficient for both AI features (ai chat / ai code) and the ABAPer
// gateway commands (search, list, generate, deploy, test, system test).
func runBai(subcommand string) error {
	bai, err := exec.LookPath("bai")
	if err != nil {
		return fmt.Errorf("bai not found in PATH — install bai to manage credentials: %w", err)
	}
	c := exec.Command(bai, subcommand)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		return err
	}
	go client.Track(subcommand, nil)
	return nil
}
