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

// MaxSlots caps card positions. Slots only order the cards; the display
// splits them into pages, so there is no limit of one screen.
const MaxSlots = 60

// staleAfter drops sessions that have not sent any event for this long
// (e.g. the terminal was closed without a SessionEnd event).
const staleAfter = 3 * time.Hour

// Steps summarizes the agent's task list for the progress bar.
type Steps struct {
	Done    int    `json:"done"`
	Total   int    `json:"total"`
	Current string `json:"current,omitempty"` // the step in progress
	// Since is when the agent reached the current step.
	Since time.Time `json:"since"`
}

// Action says what an agent is doing as a language-neutral code, so the
// display translates it. Arg carries the detail (file name, tool, command).
type Action struct {
	Kind string `json:"kind"`
	Arg  string `json:"arg,omitempty"`
}

// Action kinds. "text" is free text written by the agent itself (e.g. the
// description of a shell command) and is shown as is.
const (
	ActIdle       = "idle"
	ActThinking   = "thinking"
	ActEdit       = "edit"
	ActRead       = "read"
	ActRun        = "run"
	ActText       = "text"
	ActSearchCode = "search_code"
	ActSearchWeb  = "search_web"
	ActDelegate   = "delegate"
	ActPlan       = "plan"
	ActMCP        = "mcp"
	ActTool       = "tool"
	ActPermission = "permission"
	ActInput      = "input"
	ActAttention  = "attention"
	ActNotice     = "notice"
	ActDone       = "done"
)

type Session struct {
	ID         string    `json:"id"`
	Project    string    `json:"project"`
	Cwd        string    `json:"cwd"`
	Status     Status    `json:"status"`
	Action     Action    `json:"action"`
	TurnStart  time.Time `json:"turn_start"`
	FinishedAt time.Time `json:"finished_at,omitzero"`
	UpdatedAt  time.Time `json:"updated_at"`
	Steps      *Steps    `json:"steps,omitempty"`
	plan       []planItem
	Mascot     int `json:"mascot"`
	// Title tells sessions of one project apart: the session title from the
	// transcript, or the start of the first prompt until a title exists.
	Title string `json:"title"`
	// Agent is which coding agent runs the session; only Claude Code for now.
	Agent string `json:"agent"`
	// Client is where the session runs (e.g. claude-desktop, cli), Model and
	// Effort what it runs with, all read from the transcript.
	Client        string `json:"client,omitempty"`
	Model         string `json:"model,omitempty"`
	Effort        string `json:"effort,omitempty"`
	infoCheckedAt time.Time
	// Slot is the card's fixed position in the order of cards (0..MaxSlots-1).
	Slot int `json:"slot"`
}

// Event is the subset of the Claude Code hook payload we use.
type Event struct {
	SessionID      string          `json:"session_id"`
	Cwd            string          `json:"cwd"`
	TranscriptPath string          `json:"transcript_path"`
	Prompt         string          `json:"prompt"`
	HookEventName  string          `json:"hook_event_name"`
	ToolName       string          `json:"tool_name"`
	ToolInput      json.RawMessage `json:"tool_input"`
	Message        string          `json:"message"`
}

type Store struct {
	mu       sync.Mutex
	sessions map[string]*Session
	layout   *Layout
	now      func() time.Time
	onChange func()
	readInfo func(transcriptPath string) transcriptInfo
}

// infoRefresh limits how often a session's transcript is checked for a new
// title, model or effort, since hook events arrive several times per second.
const infoRefresh = 20 * time.Second

// AgentClaudeCode identifies sessions reported by Claude Code hooks.
const AgentClaudeCode = "claude-code"

// NewStore creates a store; layout may be nil to keep slot preferences in memory only.
func NewStore(layout *Layout) *Store {
	if layout == nil {
		layout = LoadLayout("")
	}
	return &Store{sessions: map[string]*Session{}, layout: layout, now: time.Now, readInfo: readTranscriptInfo}
}

// refreshInfo updates title, model and effort from the transcript. Values
// not found keep what was known, e.g. the prompt fallback for the title.
func (s *Store) refreshInfo(sess *Session, transcriptPath string, force bool) {
	now := s.now()
	if transcriptPath == "" || (!force && now.Sub(sess.infoCheckedAt) < infoRefresh) {
		return
	}
	sess.infoCheckedAt = now
	info := s.readInfo(transcriptPath)
	if info.Title != "" {
		sess.Title = truncate(info.Title, 40)
	}
	if info.Model != "" {
		sess.Model, sess.Effort, sess.Client = info.Model, info.Effort, info.Entrypoint
	}
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
		project := projectName(ev.Cwd)
		sess = &Session{
			ID:        ev.SessionID,
			Project:   project,
			Cwd:       ev.Cwd,
			Status:    StatusIdle,
			Action:    Action{Kind: ActIdle},
			Agent:     AgentClaudeCode,
			TurnStart: now,
			Mascot:    s.freeMascot(project),
			Slot:      s.pickSlot(project),
		}
		s.sessions[ev.SessionID] = sess
	}
	sess.UpdatedAt = now
	// A reply just ended, so check right away in case the model or effort changed.
	s.refreshInfo(sess, ev.TranscriptPath, ev.HookEventName == "Stop")

	switch ev.HookEventName {
	case "UserPromptSubmit":
		sess.Status = StatusWorking
		sess.Action = Action{Kind: ActThinking}
		sess.TurnStart = now
		sess.FinishedAt = time.Time{}
		if sess.planFinished() {
			sess.plan = nil // a new task starts without the previous plan
			sess.updateSteps(now)
		}
		if sess.Title == "" {
			sess.Title = truncate(ev.Prompt, 40)
		}
	case "PreToolUse":
		sess.Status = StatusWorking
		sess.Action = describeTool(ev.ToolName, ev.ToolInput)
		if planTool(ev.ToolName) {
			sess.applyPlan(ev.ToolName, ev.ToolInput, now)
		}
	case "PostToolUse":
		// A tool finished, so any pending permission prompt was answered.
		sess.Status = StatusWorking
	case "Notification":
		sess.Status = StatusWaiting
		sess.Action = describeNotification(ev.Message)
	case "Stop":
		sess.Status = StatusDone
		sess.Action = Action{Kind: ActDone}
		sess.FinishedAt = now
	}
}

// Snapshot returns all sessions ordered by slot. Sessions that finished more
// than hideDoneAfter ago are left out (0 keeps them until they end).
func (s *Store) Snapshot(hideDoneAfter time.Duration) []Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	out := make([]Session, 0, len(s.sessions))
	for id, sess := range s.sessions {
		if now.Sub(sess.UpdatedAt) > staleAfter {
			delete(s.sessions, id)
			continue
		}
		if hideDoneAfter > 0 && sess.Status == StatusDone && now.Sub(sess.FinishedAt) > hideDoneAfter {
			continue
		}
		out = append(out, *sess)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Slot < out[j].Slot })
	return out
}

// Move puts a session into the given slot, swapping with whatever card is there,
// and remembers the new places of both projects for future sessions.
func (s *Store) Move(id string, slot int) bool {
	if slot < 0 || slot >= MaxSlots {
		return false
	}
	s.mu.Lock()
	sess, ok := s.sessions[id]
	if !ok {
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
func (s *Store) pickSlot(project string) int {
	used := map[int]bool{}
	for _, sess := range s.sessions {
		used[sess.Slot] = true
	}
	if slot, ok := s.layout.preferred(project); ok && !used[slot] {
		return slot
	}
	for slot := range MaxSlots {
		if !used[slot] && !s.layout.reservedBy(slot, project) {
			return slot
		}
	}
	for slot := range MaxSlots {
		if !used[slot] {
			return slot
		}
	}
	return MaxSlots - 1
}

// freeMascot prefers the project's own mascot but skips ones already in use,
// so two cards never look the same, even for two sessions of one project.
func (s *Store) freeMascot(project string) int {
	used := map[int]bool{}
	for _, sess := range s.sessions {
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

func describeTool(name string, raw json.RawMessage) Action {
	var in struct {
		FilePath    string `json:"file_path"`
		Command     string `json:"command"`
		Description string `json:"description"`
	}
	_ = json.Unmarshal(raw, &in)
	file := filepath.Base(in.FilePath)

	switch name {
	case "Edit", "MultiEdit", "Write", "NotebookEdit":
		return Action{Kind: ActEdit, Arg: file}
	case "Read":
		return Action{Kind: ActRead, Arg: file}
	case "Bash", "PowerShell":
		if in.Description != "" {
			return Action{Kind: ActText, Arg: truncate(in.Description, 40)}
		}
		return Action{Kind: ActRun, Arg: truncate(in.Command, 30)}
	case "Grep", "Glob":
		return Action{Kind: ActSearchCode}
	case "WebFetch", "WebSearch":
		return Action{Kind: ActSearchWeb}
	case "Task", "Agent":
		return Action{Kind: ActDelegate}
	}
	if planTool(name) {
		return Action{Kind: ActPlan}
	}
	if strings.HasPrefix(name, "mcp__") {
		return Action{Kind: ActMCP}
	}
	return Action{Kind: ActTool, Arg: name}
}

// describeNotification turns Claude Code notification messages into action codes.
func describeNotification(msg string) Action {
	const permission = "Claude needs your permission to use "
	switch {
	case strings.HasPrefix(msg, permission):
		return Action{Kind: ActPermission, Arg: strings.TrimPrefix(msg, permission)}
	case strings.Contains(msg, "waiting for your input"):
		return Action{Kind: ActInput}
	case msg == "":
		return Action{Kind: ActAttention}
	}
	return Action{Kind: ActNotice, Arg: truncate(msg, 40)}
}

func truncate(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
