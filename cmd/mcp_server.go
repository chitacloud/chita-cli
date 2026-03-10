package cmd

import (
	"fmt"
	"os"

	"github.com/chitacloud/chita-cli/internal/auth"
	"github.com/chitacloud/chita-cli/internal/mcpproxy"
	"github.com/spf13/cobra"
)

var mcpURL string

var mcpServerCmd = &cobra.Command{
	Use:   "mcp-server",
	Short: "Run a local MCP stdio proxy to a remote Chita MCP server",
	Long: `Starts an MCP stdio server that proxies all JSON-RPC traffic to the
given remote MCP URL, automatically injecting your Chita auth token.

Add it to your MCP client config (Claude Desktop, Cursor, etc.):

  {
    "mcpServers": {
      "chita": {
        "command": "chita",
        "args": ["mcp-server", "--mcp-url", "https://app.chitacloud.com/mcp"]
      }
    }
  }`,
	RunE: runMcpServer,
}

func init() {
	rootCmd.AddCommand(mcpServerCmd)
	mcpServerCmd.Flags().StringVar(&mcpURL, "mcp-url", "", "Remote MCP server URL (required)")
	mcpServerCmd.MarkFlagRequired("mcp-url") //nolint:errcheck
}

func runMcpServer(_ *cobra.Command, _ []string) error {
	token, err := auth.LoadToken()
	if err != nil {
		return fmt.Errorf("not logged in – run `chita login` first: %w", err)
	}

	proxy := mcpproxy.New(mcpURL, token)
	if err := proxy.Run(os.Stdin, os.Stdout, os.Stderr); err != nil {
		return fmt.Errorf("proxy error: %w", err)
	}
	return nil
}
