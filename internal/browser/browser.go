package browser

import (
	"os/exec"
	"runtime"
)

// Open opens the given URL in the default system browser.
// Errors are silently ignored — the caller should always print the URL as fallback.
func Open(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}
