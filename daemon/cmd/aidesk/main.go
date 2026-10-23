// Command aidesk is the PC side of the ai-desk hub.
//
//	aidesk serve          run the daemon and the browser simulator
//	aidesk hook           forward a Claude Code hook payload from stdin (used in settings.json)
//	aidesk hooks-config   print the hooks block to paste into Claude Code settings.json
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"aidesk/internal/hook"
	"aidesk/internal/server"
	"aidesk/internal/state"
)

const defaultAddr = "127.0.0.1:7777"

func main() {
	addr := os.Getenv("AIDESK_ADDR")
	if addr == "" {
		addr = defaultAddr
	}

	cmd := "serve"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	switch cmd {
	case "serve":
		serve(addr)
	case "hook":
		hook.Forward(addr)
	case "hooks-config":
		printHooksConfig()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q (use: serve, hook, hooks-config)\n", cmd)
		os.Exit(2)
	}
}

func serve(addr string) {
	srv := server.New(state.NewStore())
	log.Printf("aidesk listening on http://%s", addr)
	log.Fatal(http.ListenAndServe(addr, srv.Handler()))
}

var hookEvents = []string{
	"SessionStart", "UserPromptSubmit", "PreToolUse", "PostToolUse",
	"Notification", "Stop", "SessionEnd",
}

func printHooksConfig() {
	exe, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	// Forward slashes work both in Git Bash (used by Claude Code on Windows) and on Unix.
	command := `"` + filepath.ToSlash(exe) + `" hook`

	type handler struct {
		Type    string `json:"type"`
		Command string `json:"command"`
		Timeout int    `json:"timeout"`
	}
	type matcher struct {
		Matcher string    `json:"matcher,omitempty"`
		Hooks   []handler `json:"hooks"`
	}
	hooks := map[string][]matcher{}
	for _, ev := range hookEvents {
		m := matcher{Hooks: []handler{{Type: "command", Command: command, Timeout: 5}}}
		if ev == "PreToolUse" || ev == "PostToolUse" {
			m.Matcher = "*"
		}
		hooks[ev] = []matcher{m}
	}
	out, _ := json.MarshalIndent(map[string]any{"hooks": hooks}, "", "  ")
	fmt.Println(string(out))
}
