// Package state tracks agent sessions based on Claude Code hook events.
package state

import (
	"encoding/json"
	"hash/fnv"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Status string

const (
	StatusIdle    Status = "idle"
	StatusWorking Status = "working"
	StatusWaiting Status = "waiting"
	StatusDone    Status = "done"
	StatusError   Status = "error"
)

// MascotCount is how many mascot variants the display knows about.
const MascotCount = 8

// MaxCards is how many sessions fit on the device screen.
const MaxCards = 6

// staleAfter drops sessions that have not sent any event for this long
// (e.g. the terminal was closed without a SessionEnd event).
const staleAfter = 3 * time.Hour

type Steps struct {
	Done  int `json:"done"`
	Total int `json:"total"`
}

type Session struct {
	ID         string    `json:"id"`
	Project    string    `json:"project"`
	Cwd        string    `json:"cwd"`
	Status     Status    `json:"status"`
	Action     string    `json:"action"`
	TurnStart  time.Time `json:"turn_start"`
	FinishedAt time.Time `json:"finished_at,omitzero"`
	UpdatedAt  time.Time `json:"updated_at"`
	Steps      *Steps    `json:"steps,omitempty"`
	Mascot     int       `json:"mascot"`
}

// Event is the subset of the Claude Code hook payload we use.
type Event struct {
	SessionID     string          `json:"session_id"`
	Cwd           string          `json:"cwd"`
	HookEventName string          `json:"hook_event_name"`
	ToolName      string          `json:"tool_name"`
	ToolInput     json.RawMessage `json:"tool_input"`
	Message       string          `json:"message"`
}

type Store struct {
	mu       sync.Mutex
	sessions map[string]*Session
	now      func() time.Time
	onChange func()
}

func NewStore() *Store {
	return &Store{sessions: map[string]*Session{}, now: time.Now}
}

// OnChange registers a callback invoked (outside the lock) after every update.
func (s *Store) OnChange(fn func()) { s.onChange = fn }

func (s *Store) Apply(ev Event) {
	if ev.SessionID == "" {
		return
	}
	s.mu.Lock()
	s.apply(ev)
	s.mu.Unlock()
	if s.onChange != nil {
		s.onChange()
	}
}

func (s *Store) apply(ev Event) {
	now := s.now()
	if ev.HookEventName == "SessionEnd" {
		delete(s.sessions, ev.SessionID)
		return
	}

	sess, ok := s.sessions[ev.SessionID]
	if !ok {
		project := filepath.Base(filepath.Clean(ev.Cwd))
		sess = &Session{
			ID:        ev.SessionID,
			Project:   project,
			Cwd:       ev.Cwd,
			Status:    StatusIdle,
			Action:    "Czeka na polecenie",
			TurnStart: now,
			Mascot:    s.freeMascot(project),
		}
		s.sessions[ev.SessionID] = sess
	}
	sess.UpdatedAt = now

	switch ev.HookEventName {
	case "UserPromptSubmit":
		sess.Status = StatusWorking
		sess.Action = "Myśli…"
		sess.TurnStart = now
		sess.FinishedAt = time.Time{}
	case "PreToolUse":
		sess.Status = StatusWorking
		sess.Action = describeTool(ev.ToolName, ev.ToolInput)
		if ev.ToolName == "TodoWrite" {
			sess.Steps = todoSteps(ev.ToolInput)
		}
	case "PostToolUse":
		// A tool finished, so any pending permission prompt was answered.
		sess.Status = StatusWorking
	case "Notification":
		sess.Status = StatusWaiting
		sess.Action = describeNotification(ev.Message)
	case "Stop":
		sess.Status = StatusDone
		sess.Action = "Gotowe"
		sess.FinishedAt = now
	}
}

// Snapshot returns up to MaxCards sessions, most recently active first.
func (s *Store) Snapshot() []Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	out := make([]Session, 0, len(s.sessions))
	for id, sess := range s.sessions {
		if now.Sub(sess.UpdatedAt) > staleAfter {
			delete(s.sessions, id)
			continue
		}
		out = append(out, *sess)
	}
	sort.Slice(out, func(i, j int) bool {
		if pi, pj := priority(out[i].Status), priority(out[j].Status); pi != pj {
			return pi < pj
		}
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	if len(out) > MaxCards {
		out = out[:MaxCards]
	}
	return out
}

// priority keeps sessions that need attention on screen when there are more than MaxCards.
func priority(st Status) int {
	switch st {
	case StatusWaiting:
		return 0
	case StatusWorking:
		return 1
	case StatusDone, StatusError:
		return 2
	}
	return 3
}

// freeMascot prefers the project's own mascot but skips ones already on screen,
// so two cards never look the same. Sessions of the same project share a mascot.
func (s *Store) freeMascot(project string) int {
	used := map[int]bool{}
	for _, sess := range s.sessions {
		if strings.EqualFold(sess.Project, project) {
			return sess.Mascot
		}
		used[sess.Mascot] = true
	}
	start := mascotFor(project)
	for i := range MascotCount {
		if m := (start + i) % MascotCount; !used[m] {
			return m
		}
	}
	return start
}

func mascotFor(project string) int {
	h := fnv.New32a()
	h.Write([]byte(strings.ToLower(project)))
	return int(h.Sum32() % MascotCount)
}

func describeTool(name string, raw json.RawMessage) string {
	var in struct {
		FilePath    string `json:"file_path"`
		Command     string `json:"command"`
		Description string `json:"description"`
		Pattern     string `json:"pattern"`
	}
	_ = json.Unmarshal(raw, &in)
	file := filepath.Base(in.FilePath)

	switch name {
	case "Edit", "MultiEdit", "Write", "NotebookEdit":
		return "Edytuje " + file
	case "Read":
		return "Czyta " + file
	case "Bash", "PowerShell":
		if in.Description != "" {
			return truncate(in.Description, 40)
		}
		return "Uruchamia: " + truncate(in.Command, 30)
	case "Grep", "Glob":
		return "Szuka w kodzie"
	case "WebFetch", "WebSearch":
		return "Szuka w sieci"
	case "Task", "Agent":
		return "Deleguje do subagenta"
	case "TodoWrite":
		return "Planuje kroki"
	}
	if strings.HasPrefix(name, "mcp__") {
		return "Używa narzędzia MCP"
	}
	return "Używa " + name
}

// describeNotification shortens Claude Code notification messages to fit a card.
func describeNotification(msg string) string {
	const permission = "Claude needs your permission to use "
	switch {
	case strings.HasPrefix(msg, permission):
		return "Prosi o zgodę: " + strings.TrimPrefix(msg, permission)
	case strings.Contains(msg, "waiting for your input"):
		return "Czeka na Twoją odpowiedź"
	case msg == "":
		return "Czeka na Ciebie"
	}
	return truncate(msg, 40)
}

func todoSteps(raw json.RawMessage) *Steps {
	var in struct {
		Todos []struct {
			Status string `json:"status"`
		} `json:"todos"`
	}
	if json.Unmarshal(raw, &in) != nil || len(in.Todos) == 0 {
		return nil
	}
	st := &Steps{Total: len(in.Todos)}
	for _, t := range in.Todos {
		if t.Status == "completed" {
			st.Done++
		}
	}
	return st
}

func truncate(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
