package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
)

// Claude Code keeps a small file per running session in
// ~/.claude/sessions/<pid>.json. It holds both the session id the hooks send
// and, for sessions in the Claude desktop app, the app's own id for it
// ("local_..."), which is what a claude:// link needs to open that session.

// hostIDPattern is the format the desktop app accepts in its links.
var hostIDPattern = regexp.MustCompile(`^local_[A-Za-z0-9-]{1,64}$`)

// ValidHostID reports whether id is a desktop app session id safe to put in a link.
func ValidHostID(id string) bool { return hostIDPattern.MatchString(id) }

// SessionsDir is Claude Code's folder of running sessions.
func SessionsDir() string {
	if dir := os.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		return filepath.Join(dir, "sessions")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude", "sessions")
}

// findHostID returns the desktop app's id for a session, or "" when the
// session doesn't run in the app (e.g. in a terminal) or isn't found.
func findHostID(dir, sessionID string) string {
	if dir == "" || sessionID == "" {
		return ""
	}
	// Only the .json files: the .key files next to them hold secrets and are never read.
	files, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil || info.Size() > 64<<10 {
			continue
		}
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var entry struct {
			SessionID     string `json:"sessionId"`
			HostSessionID string `json:"hostSessionId"`
		}
		if json.Unmarshal(data, &entry) != nil || entry.SessionID != sessionID {
			continue
		}
		if ValidHostID(entry.HostSessionID) {
			return entry.HostSessionID
		}
		return ""
	}
	return ""
}
