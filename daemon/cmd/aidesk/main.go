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
	"time"

	"aidesk/internal/hook"
	"aidesk/internal/server"
	"aidesk/internal/settings"
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
	var layoutPath, settingsPath string
	if dir := configDir(); dir != "" {
		layoutPath = filepath.Join(dir, "layout.json")
		settingsPath = filepath.Join(dir, "settings.json")
	}
	store := state.NewStore(state.LoadLayout(layoutPath))
	go func() {
		for range time.Tick(2 * time.Second) {
			store.CheckInterrupts()
		}
	}()
	srv := server.New(store, settings.Load(settingsPath))
	log.Printf("aidesk listening on http://%s", addr)
	log.Fatal(http.ListenAndServe(addr, srv.Handler()))
}

// configDir is where settings and layout are saved: AIDESK_CONFIG_DIR if set
// (handy for a separate test daemon), else the user's config directory.
func configDir() string {
	if dir := os.Getenv("AIDESK_CONFIG_DIR"); dir != "" {
		return dir
	}
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "aidesk")
	}
	return ""
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
