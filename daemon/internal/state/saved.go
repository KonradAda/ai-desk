package state

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Cards stay on the screen once the user has talked in a session, across
// daemon and computer restarts, until the user takes them off or the session
// is deleted in Claude Code. The daemon keeps them in a sessions file.

// savedSession is a session as written to the sessions file.
type savedSession struct {
	Session
	TranscriptPath string `json:"transcript_path,omitempty"`
}

// UseSessionsFile loads the cards saved in path and makes SaveSessions write
// there. Call it before any events arrive. Sessions that were working when
// the daemon stopped come back as done; RestoreRunning then marks the ones
// still running.
func (s *Store) UseSessionsFile(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessionsPath = path
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var saved []savedSession
	if err := json.Unmarshal(data, &saved); err != nil {
		log.Printf("sessions: ignoring unreadable %s: %v", path, err)
		return
	}
	for _, sv := range saved {
		if sv.ID == "" || s.sessions[sv.ID] != nil {
			continue
		}
		sess := sv.Session
		sess.transcriptPath = sv.TranscriptPath
		if sess.Status == StatusWorking {
			sess.Status, sess.Action, sess.FinishedAt = StatusDone, Action{Kind: ActDone}, sess.UpdatedAt
		}
		sess.Steps = nil // the plan behind them isn't saved
		s.sessions[sess.ID] = &sess
	}
	s.lastSaved = data
}

// SaveSessions writes the cards to the sessions file if they changed since
// the last write. Sessions nobody has talked in yet (idle) are left out.
func (s *Store) SaveSessions() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sessionsPath == "" {
		return
	}
	saved := make([]savedSession, 0, len(s.sessions))
	for _, sess := range s.sessions {
		if sess.Status != StatusIdle {
			saved = append(saved, savedSession{Session: *sess, TranscriptPath: sess.transcriptPath})
		}
	}
	sort.Slice(saved, func(i, j int) bool { return saved[i].Slot < saved[j].Slot })
	data, _ := json.MarshalIndent(saved, "", "  ")
	if bytes.Equal(data, s.lastSaved) {
		return
	}
	if err := os.MkdirAll(filepath.Dir(s.sessionsPath), 0o755); err != nil {
		log.Printf("sessions: %v", err)
		return
	}
	if err := os.WriteFile(s.sessionsPath, data, 0o644); err != nil {
		log.Printf("sessions: %v", err)
		return
	}
	s.lastSaved = data
}

// deletedAfter is how long a session must go without events before a
// missing transcript counts as deleted; a brand-new session may not have
// written its transcript yet.
const deletedAfter = time.Minute

// PruneDeleted drops cards of sessions deleted in Claude Code: their
// transcript file is gone. Claude Code also deletes old transcripts by
// itself (after 30 days by default), so old cards go away on their own.
func (s *Store) PruneDeleted() {
	s.mu.Lock()
	check := map[string]string{} // session id -> transcript
	for id, sess := range s.sessions {
		if sess.transcriptPath != "" && s.now().Sub(sess.UpdatedAt) >= deletedAfter {
			check[id] = sess.transcriptPath
		}
	}
	s.mu.Unlock()

	var gone []string
	for id, path := range check {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			gone = append(gone, id)
		}
	}
	if len(gone) == 0 {
		return
	}
	s.mu.Lock()
	for _, id := range gone {
		delete(s.sessions, id)
	}
	s.mu.Unlock()
	if s.onChange != nil {
		s.onChange()
	}
}

// Seen tells the store the user opened a session from its card. If the agent
// finished its reply with a question, that counts as noticed: the card stops
// calling and goes to sleep. A wait the agent can't get past without the user
// (permission, question tool, plan approval) keeps calling until it's answered.
func (s *Store) Seen(id string) {
	s.mu.Lock()
	sess, ok := s.sessions[id]
	seen := ok && sess.Status == StatusWaiting && !sess.blocked
	if seen {
		sess.Status, sess.Action, sess.FinishedAt = StatusDone, Action{Kind: ActDone}, s.now()
	}
	s.mu.Unlock()
	if seen && s.onChange != nil {
		s.onChange()
	}
}

// Remove takes a card off the screen for good. The session comes back as a
// new card only if it sends events again. It reports whether the session was known.
func (s *Store) Remove(id string) bool {
	s.mu.Lock()
	_, ok := s.sessions[id]
	delete(s.sessions, id)
	s.mu.Unlock()
	if ok && s.onChange != nil {
		s.onChange()
	}
	return ok
}
