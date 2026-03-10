package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"time"
)

const timeoutDuration = 5 * time.Minute

var chitaOriginRe = regexp.MustCompile(`^https://app(-staging)?\.chitacloud\.com$`)

// CallbackServer is the temporary local HTTP server that receives the OAuth token.
type CallbackServer struct {
	Port    int
	server  *http.Server
	tokenCh chan string
	errCh   chan error
	appURL  string
}

// GenerateState generates a cryptographically random state string for CSRF protection.
func GenerateState() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// StartCallbackServer starts a temporary HTTP server on a random localhost port.
// Returns immediately with the assigned port; use WaitForToken to block until the
// browser delivers the token.
func StartCallbackServer(expectedState, appURL string) (*CallbackServer, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}

	port := listener.Addr().(*net.TCPAddr).Port
	tokenCh := make(chan string, 1)
	errCh := make(chan error, 1)

	cs := &CallbackServer{
		Port:    port,
		tokenCh: tokenCh,
		errCh:   errCh,
		appURL:  appURL,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", cs.handleCallback(expectedState))
	cs.server = &http.Server{Handler: mux}

	go cs.server.Serve(listener) //nolint:errcheck

	return cs, nil
}

func (cs *CallbackServer) handleCallback(expectedState string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		corsOrigin := "https://app.chitacloud.com"
		if chitaOriginRe.MatchString(origin) {
			corsOrigin = origin
		}

		// Chrome Private Network Access preflight
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", corsOrigin)
			w.Header().Set("Access-Control-Allow-Private-Network", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Max-Age", "300")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		token := r.URL.Query().Get("token")
		state := r.URL.Query().Get("state")

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Access-Control-Allow-Origin", corsOrigin)
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, cs.successHTML())

		// Shut down server after responding
		go cs.server.Shutdown(context.Background()) //nolint:errcheck

		if token == "" || state != expectedState {
			cs.errCh <- fmt.Errorf("invalid callback: missing or mismatched parameters")
			return
		}

		cs.tokenCh <- token
	}
}

// WaitForToken blocks until the browser delivers the token or the timeout expires.
func (cs *CallbackServer) WaitForToken() (string, error) {
	select {
	case token := <-cs.tokenCh:
		return token, nil
	case err := <-cs.errCh:
		return "", err
	case <-time.After(timeoutDuration):
		cs.server.Shutdown(context.Background()) //nolint:errcheck
		return "", fmt.Errorf("authentication timed out after 5 minutes. Run `chita login` again")
	}
}

func (cs *CallbackServer) successHTML() string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Chita Cloud – Authenticated</title>
  <style>
    *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      font-family: system-ui, -apple-system, sans-serif;
      background: #0f0c19;
      color: #fafafa;
      min-height: 100vh;
      display: flex;
      align-items: center;
      justify-content: center;
      padding: 1rem;
    }
    .card {
      background: #1a1528;
      border: 1px solid #2a1f45;
      border-radius: 12px;
      padding: 2.5rem 2rem;
      width: 100%%;
      max-width: 400px;
      text-align: center;
    }
    .logo { height: 40px; margin-bottom: 2rem; }
    .icon {
      width: 56px; height: 56px;
      background: #14532d;
      border-radius: 50%%;
      display: flex; align-items: center; justify-content: center;
      margin: 0 auto 1.25rem;
      font-size: 1.5rem;
    }
    h1 { font-size: 1.25rem; font-weight: 600; margin-bottom: 0.5rem; }
    p { color: #a3a3a3; font-size: 0.9rem; line-height: 1.5; }
    .badge {
      display: inline-flex; align-items: center; gap: 0.4rem;
      background: #14532d33;
      border: 1px solid #22c55e44;
      color: #22c55e;
      font-size: 0.75rem;
      padding: 0.25rem 0.75rem;
      border-radius: 999px;
      margin-bottom: 1.5rem;
    }
  </style>
</head>
<body>
  <div class="card">
    <img src="%s/chitacloud_v2_3.png" alt="Chita Cloud" class="logo" />
    <div class="icon">✓</div>
    <span class="badge">CLI authorized</span>
    <h1>Authentication complete</h1>
    <p>You can close this browser tab and return to the terminal.</p>
  </div>
</body>
</html>`, cs.appURL)
}
