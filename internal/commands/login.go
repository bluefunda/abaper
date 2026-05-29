package commands

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/bluefunda/abaper/internal/client"
	"github.com/bluefunda/abaper/internal/config"
	"github.com/spf13/cobra"
)

const (
	ansiReset = "\033[0m"
	ansiBold  = "\033[1m"
	ansiDim   = "\033[2m"
	ansiGreen = "\033[32m"
	ansiCyan  = "\033[36m"
	eraseLine = "\r\033[K"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with ABAPer using device authorization flow",
	RunE:  runLogin,
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear stored credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.ClearTokens(); err != nil {
			return fmt.Errorf("logout: %w", err)
		}
		printLoginSuccess("Logged out successfully.")
		go client.Track("logout", nil)
		return nil
	},
}

func runLogin(cmd *cobra.Command, args []string) error {
	cfg := config.Load()
	realm := cfg.Realm

	go client.Track("login_started", nil)

	printLoginHeader("ABAPer — Login")
	fmt.Println()

	// Step 1: Request device code
	deviceResp, err := client.RequestDeviceCode(realm)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	// Build login URL
	verifyURL := deviceResp.VerificationURIComplete
	if verifyURL == "" {
		verifyURL = deviceResp.VerificationURI
	}
	loginURL := fmt.Sprintf("https://bluefunda.com/login?redirect_uri=%s&utm_source=cli&utm_medium=command&utm_campaign=login",
		url.QueryEscape(verifyURL))

	fmt.Println("  To authenticate, open this URL in your browser:")
	fmt.Println()
	fmt.Printf("    %s%s%s\n", ansiCyan, loginURL, ansiReset)
	fmt.Println()
	if deviceResp.UserCode != "" {
		fmt.Printf("  Your code: %s%s%s\n\n", ansiBold, deviceResp.UserCode, ansiReset)
	}

	// Copy URL to clipboard
	if copyLoginURLToClipboard(loginURL) {
		printLoginCheck("URL copied to clipboard")
	}

	// Open browser
	if err := openBrowser(loginURL); err == nil {
		printLoginCheck("Opening browser automatically...")
	} else {
		fmt.Fprintf(os.Stderr, "  %sCould not open browser — please copy the URL above%s\n", ansiDim, ansiReset)
	}
	fmt.Println()

	// Step 3: Poll for token with spinner
	done := make(chan struct{})
	go runLoginSpinner("Waiting for authentication in the browser", done)

	tokenResp, err := client.PollForToken(realm, deviceResp.DeviceCode, deviceResp.Interval)
	close(done)
	fmt.Print(eraseLine)

	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	// Step 4: Save tokens
	tokens := &config.Tokens{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second).UnixMilli(),
	}
	if err := config.SaveTokens(tokens); err != nil {
		return fmt.Errorf("save credentials: %w", err)
	}

	printLoginSuccess("Logged in successfully!")
	go client.Track("login_completed", map[string]string{"success": "true"})
	return nil
}

func printLoginHeader(title string) {
	width := len(title) + 4
	border := strings.Repeat("─", width)
	fmt.Printf("╭%s╮\n", border)
	fmt.Printf("│  %s%s%s  │\n", ansiBold, title, ansiReset)
	fmt.Printf("╰%s╯\n", border)
}

func printLoginCheck(msg string) {
	fmt.Printf("  %s✓%s %s\n", ansiGreen, ansiReset, msg)
}

func printLoginSuccess(msg string) {
	fmt.Printf("\n%s✓%s %s%s%s\n", ansiGreen, ansiReset, ansiBold, msg, ansiReset)
}

func runLoginSpinner(msg string, done <-chan struct{}) {
	i := 0
	for {
		select {
		case <-done:
			return
		default:
			fmt.Printf("\r  %s%s%s %s...", ansiDim, spinnerFrames[i%len(spinnerFrames)], ansiReset, msg)
			i++
			time.Sleep(80 * time.Millisecond)
		}
	}
}

func copyLoginURLToClipboard(text string) bool {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else if _, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		} else {
			return false
		}
	case "windows":
		cmd = exec.Command("clip")
	default:
		return false
	}

	in, err := cmd.StdinPipe()
	if err != nil {
		return false
	}
	if err := cmd.Start(); err != nil {
		return false
	}
	_, _ = fmt.Fprint(in, text)
	_ = in.Close()
	return cmd.Wait() == nil
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform")
	}
	return cmd.Start()
}
