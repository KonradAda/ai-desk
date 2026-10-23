// Package hook forwards a Claude Code hook payload (read from stdin) to the daemon.
//
// It must never slow down or break the agent: short timeout, errors are
// swallowed, and the caller always exits with code 0.
package hook

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const timeout = 300 * time.Millisecond

func Forward(addr string) {
	body, err := io.ReadAll(io.LimitReader(os.Stdin, 1<<20))
	if err != nil || len(body) == 0 {
		return
	}
	client := http.Client{Timeout: timeout}
	resp, err := client.Post("http://"+addr+"/event", "application/json", bytes.NewReader(body))
	if err != nil {
		// Daemon not running: stay silent unless debugging.
		if os.Getenv("AIDESK_DEBUG") != "" {
			fmt.Fprintln(os.Stderr, "aidesk hook:", err)
		}
		return
	}
	resp.Body.Close()
	if resp.StatusCode >= 300 && os.Getenv("AIDESK_DEBUG") != "" {
		fmt.Fprintln(os.Stderr, "aidesk hook: daemon replied", resp.Status)
	}
}
