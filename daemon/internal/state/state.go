// Package state tracks agent sessions based on Claude Code hook events.
package state

import (
	"encoding/json"
	"hash/fnv"
	"os"
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
	// Slot is the fixed card position on screen (0..MaxCards-1), or -1 when
	// all slots are taken and the session waits off screen.
	Slot int `json:"slot"`
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
	layout   *Layout
	now      func() time.Time
	onChange func()
}

// NewStore creates a store; layout may be nil to keep slot preferences in memory only.
func NewStore(layout *Layout) *Store {
	if layout == nil {
		layout = LoadLayout("")
	}
	return &Store{sessions: map[string]*Session{}, layout: layout, now: time.Now}
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
		s.fillSlots()
		return
	}

	sess, ok := s.sessions[ev.SessionID]
	if !ok {
		project := projectName(ev.Cwd)
		sess = &Session{
			ID:        ev.SessionID,
			Project:   project,
			Cwd:       ev.Cwd,
			Status:    StatusIdle,
			Action:    "Czeka na polecenie",
			TurnStart: now,
			Mascot:    s.freeMascot(project),
			Slot:      s.pickSlot(project),
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

// Snapshot returns the sessions on screen ordered by slot, and how many
// more are waiting off screen because all slots are taken.
func (s *Store) Snapshot() (visible []Session, hidden int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	pruned := false
	for id, sess := range s.sessions {
		if now.Sub(sess.UpdatedAt) > staleAfter {
			delete(s.sessions, id)
			pruned = true
		}
	}
	if pruned {
		s.fillSlots()
	}
	visible = make([]Session, 0, MaxCards)
	for _, sess := range s.sessions {
		if sess.Slot < 0 {
			hidden++
			continue
		}
		visible = append(visible, *sess)
	}
	sort.Slice(visible, func(i, j int) bool { return visible[i].Slot < visible[j].Slot })
	return visible, hidden
}

// Move puts a session into the given slot, swapping with whatever card is there,
// and remembers the new places of both projects for future sessions.
func (s *Store) Move(id string, slot int) bool {
	if slot < 0 || slot >= MaxCards {
		return false
	}
	s.mu.Lock()
	sess, ok := s.sessions[id]
	if !ok || sess.Slot < 0 {
		s.mu.Unlock()
		return false
	}
	from := sess.Slot
	for _, other := range s.sessions {
		if other != sess && other.Slot == slot {
			other.Slot = from
			s.layout.set(other.Project, from)
		}
	}
	sess.Slot = slot
	s.layout.set(sess.Project, slot)
	s.layout.save()
	s.mu.Unlock()
	if s.onChange != nil {
		s.onChange()
	}
	return true
}

// pickSlot chooses a free slot for a new session: the project's remembered slot
// if free, then a free slot no other project has claimed, then any free slot.
// It returns -1 when the screen is full.
func (s *Store) pickSlot(project string) int {
	used := map[int]bool{}
	for _, sess := range s.sessions {
		if sess.Slot >= 0 {
			used[sess.Slot] = true
		}
	}
	if slot, ok := s.layout.preferred(project); ok && !used[slot] {
		return slot
	}
	for slot := range MaxCards {
		if !used[slot] && !s.layout.reservedBy(slot, project) {
			return slot
		}
	}
	for slot := range MaxCards {
		if !used[slot] {
			return slot
		}
	}
	return -1
}

// fillSlots moves off-screen sessions into slots freed by ended sessions,
// those that need attention first.
func (s *Store) fillSlots() {
	var waiting []*Session
	for _, sess := range s.sessions {
		if sess.Slot < 0 {
			waiting = append(waiting, sess)
		}
	}
	sort.Slice(waiting, func(i, j int) bool {
		if pi, pj := priority(waiting[i].Status), priority(waiting[j].Status); pi != pj {
			return pi < pj
		}
		return waiting[i].UpdatedAt.After(waiting[j].UpdatedAt)
	})
	for _, sess := range waiting {
		if sess.Slot = s.pickSlot(sess.Project); sess.Slot < 0 {
			return
		}
	}
}

// priority decides which off-screen session gets a freed slot first.
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

// projectName names a session after its git repository root, so an agent that
// works in a subdirectory still shows the project name. Without git it falls
// back to the working directory's name.
func projectName(cwd string) string {
	dir := filepath.Clean(cwd)
	for d := dir; ; {
		if _, err := os.Stat(filepath.Join(d, ".git")); err == nil {
			return filepath.Base(d)
		}
		parent := filepath.Dir(d)
		if parent == d {
			break
		}
		d = parent
	}
	return filepath.Base(dir)
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
