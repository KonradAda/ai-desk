package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// RestoreRunning adds the sessions Claude Code is running right now, read
// from its sessions folder (see SessionsDir), so the screen fills up as soon
// as the daemon starts instead of waiting for each session's next hook event.
// Files of sessions whose process is gone are skipped.
func (s *Store) RestoreRunning(dir string) {
	if dir == "" {
		return
	}
	files, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	s.mu.Lock()
	for _, f := range files {
		if info, err := os.Stat(f); err != nil || info.Size() > 64<<10 {
			continue
		}
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var entry struct {
			PID             int    `json:"pid"`
			SessionID       string `json:"sessionId"`
			Cwd             string `json:"cwd"`
			Name            string `json:"name"`
			Status          string `json:"status"` // "busy" while working, "idle" otherwise
			Entrypoint      string `json:"entrypoint"`
			HostSessionID   string `json:"hostSessionId"`
			StatusUpdatedAt int64  `json:"statusUpdatedAt"` // ms since the epoch
		}
		if json.Unmarshal(data, &entry) != nil || entry.SessionID == "" || !processAlive(entry.PID) {
			continue
		}
		s.restore(entry.SessionID, entry.Cwd, entry.Name, entry.Status, entry.Entrypoint, entry.HostSessionID,
			time.UnixMilli(entry.StatusUpdatedAt), transcriptFor(dir, entry.SessionID))
	}
	s.mu.Unlock()
	if s.onChange != nil {
		s.onChange()
	}
}

// restore adds one running session, or updates its saved card. since is when
// its status last changed. An idle session is shown as it ended: waiting if
// it asked something, interrupted if the user stopped it, else done.
func (s *Store) restore(id, cwd, name, status, client, hostID string, since time.Time, transcript string) {
	now := s.now()
	sess := s.sessions[id]
	if sess == nil {
		sess = s.newSession(id, cwd)
	}
	sess.UpdatedAt = now
	sess.Client = client
	if ValidHostID(hostID) {
		sess.HostID = hostID
		sess.hostCheckedAt = now
	}
	sess.Pinned = sess.Pinned || s.pins.has(id) || s.pins.has(sess.HostID)
	sess.transcriptPath = transcript
	info := s.refreshInfo(sess, transcript, true)
	if name != "" {
		sess.Title = truncate(name, 40)
	}
	switch {
	case status == "busy":
		sess.Status, sess.Action, sess.TurnStart = StatusWorking, Action{Kind: ActThinking}, since
	case transcript != "" && s.interrupted(transcript):
		sess.Status, sess.Action, sess.FinishedAt = StatusInterrupted, Action{Kind: ActInterrupted}, since
	case info.Asks:
		sess.Status, sess.Action = StatusWaiting, Action{Kind: ActQuestion}
	default:
		sess.Status, sess.Action, sess.FinishedAt = StatusDone, Action{Kind: ActDone}, since
	}
}

// transcriptFor finds a session's transcript: Claude Code keeps it in
// projects/<folder named after the cwd>/<session id>.jsonl, next to the
// sessions folder.
func transcriptFor(sessionsDir, sessionID string) string {
	matches, _ := filepath.Glob(filepath.Join(filepath.Dir(sessionsDir), "projects", "*", sessionID+".jsonl"))
	if len(matches) == 0 {
		return ""
	}
	return matches[0]
}
