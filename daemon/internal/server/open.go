package server

import (
	"os/exec"
	"runtime"
)

// openURL hands a link to the operating system, which passes claude:// links
// to the Claude desktop app. The link goes as a single argument, never through
// a shell, so its "&" and other characters aren't interpreted.
func openURL(url string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}
