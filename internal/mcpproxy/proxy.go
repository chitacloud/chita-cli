package mcpproxy

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

// Proxy is a stdio-to-HTTP MCP proxy. It reads newline-delimited JSON-RPC
// messages from an io.Reader (typically os.Stdin), forwards each one as an
// HTTP POST to the remote MCP server with Bearer auth, and writes the
// response(s) back to an io.Writer (typically os.Stdout).
//
// Supports both plain JSON responses and SSE streaming responses.
// Maintains the Mcp-Session-Id header across requests for session continuity.
type Proxy struct {
	mcpURL    string
	token     string
	sessionID string
	mu        sync.Mutex
	client    *http.Client
}

// New creates a Proxy that forwards to mcpURL authenticating with token.
func New(mcpURL, token string) *Proxy {
	return &Proxy{
		mcpURL: mcpURL,
		token:  token,
		// No global timeout: SSE streams can be long-lived.
		client: &http.Client{},
	}
}

// Run reads JSON-RPC messages from in line by line and forwards each to the
// remote server, writing all responses to out. Errors per-message are written
// to errOut (use os.Stderr). Returns when in reaches EOF or a scan error.
func (p *Proxy) Run(in io.Reader, out io.Writer, errOut io.Writer) error {
	scanner := bufio.NewScanner(in)
	// MCP messages can be large (file contents etc) – 4 MB buffer.
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)

	w := bufio.NewWriter(out)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		if err := p.forward(line, w, errOut); err != nil {
			fmt.Fprintf(errOut, "mcpproxy: %v\n", err)
		}
		w.Flush() //nolint:errcheck
	}
	return scanner.Err()
}

func (p *Proxy) forward(msg string, out *bufio.Writer, errOut io.Writer) error {
	req, err := http.NewRequest(http.MethodPost, p.mcpURL, strings.NewReader(msg))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Authorization", "Bearer "+p.token)

	p.mu.Lock()
	if p.sessionID != "" {
		req.Header.Set("Mcp-Session-Id", p.sessionID)
	}
	p.mu.Unlock()

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("POST %s: %w", p.mcpURL, err)
	}
	defer resp.Body.Close()

	// Persist session ID returned by the server.
	if sid := resp.Header.Get("Mcp-Session-Id"); sid != "" {
		p.mu.Lock()
		p.sessionID = sid
		p.mu.Unlock()
	}

	ct := resp.Header.Get("Content-Type")
	if strings.Contains(ct, "text/event-stream") {
		return readSSE(resp.Body, out, errOut)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	body = bytes.TrimSpace(body)
	if len(body) > 0 {
		fmt.Fprintf(out, "%s\n", body)
	}
	return nil
}

// readSSE parses a text/event-stream response and writes each JSON-RPC data
// payload as a single line to out.
func readSSE(r io.Reader, out *bufio.Writer, _ io.Writer) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "" || data == "[DONE]" {
			continue
		}
		fmt.Fprintf(out, "%s\n", data)
		out.Flush() //nolint:errcheck
	}
	return scanner.Err()
}
