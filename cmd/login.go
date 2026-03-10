package cmd

import (
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/briandowns/spinner"
	"github.com/chitacloud/chita-cli/internal/auth"
	"github.com/chitacloud/chita-cli/internal/browser"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to Chita Cloud",
	RunE:  runLogin,
}

func init() {
	rootCmd.AddCommand(loginCmd)
}

func runLogin(_ *cobra.Command, _ []string) error {
	appURL := os.Getenv("CHITA_APP_URL")
	if appURL == "" {
		appURL = "https://app.chitacloud.com"
	}

	state := auth.GenerateState()

	srv, err := auth.StartCallbackServer(state, appURL)
	if err != nil {
		return fmt.Errorf("failed to start local server: %w", err)
	}

	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", srv.Port)
	authURL := fmt.Sprintf("%s/login/cli?redirect_uri=%s&state=%s",
		appURL,
		url.QueryEscape(redirectURI),
		state,
	)

	browser.Open(authURL)

	bold := color.New(color.Bold)
	dim := color.New(color.Faint)

	fmt.Println()
	bold.Println("Chita Cloud – CLI Login")
	fmt.Println("────────────────────────────────────────")
	fmt.Println()
	fmt.Println("Opening browser for authentication...")
	fmt.Println()
	dim.Println("If your browser did not open, visit:")
	color.Cyan("%s\n", authURL)

	s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
	s.Suffix = " Waiting for authentication..."
	s.Start()

	token, err := srv.WaitForToken()
	s.Stop()

	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	color.Green("✔ Authentication successful")

	if err := auth.SaveToken(token); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	color.Green("\n✓ You are now logged in to Chita Cloud.\n")
	return nil
}
