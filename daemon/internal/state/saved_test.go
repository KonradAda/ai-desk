package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCardsSurviveRestart(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sessions.json")

	s, _ := newTestStore()
	s.UseSessionsFile(path)
	s.Apply(Event{SessionID: "done", Cwd: "/alpha", TranscriptPath: "/t/done.jsonl", HookEventName: "UserPromptSubmit", Prompt: "Fix the login"})
	s.Apply(Event{SessionID: "done", HookEventName: "Stop"})
	s.Apply(Event{SessionID: "busy", Cwd: "/beta", HookEventName: "UserPromptSubmit", Prompt: "Write docs"})
	s.Apply(Event{SessionID: "fresh", Cwd: "/gamma", HookEventName: "SessionStart"})
	s.Move("busy", 4)
	s.Pin("done", true)
	s.SaveSessions()

	// A new daemon: talked-in sessions come back on their slots, asleep;
	// the one that never got a prompt doesn't.
	s2, _ := newTestStore()
	s2.UseSessionsFile(path)
	got := map[string]Session{}
	for _, sess := range snapshot(s2) {
		got[sess.ID] = sess
	}
	if len(got) != 2 {
		t.Fatalf("want 2 cards back, got %+v", got)
	}
	if d := got["done"]; d.Status != StatusDone || d.Title != "Fix the login" || !d.Pinned || d.Project != "alpha" {
		t.Errorf("unexpected card: %+v", d)
	}
	if b := got["busy"]; b.Status != StatusDone || b.Slot != 4 {
		t.Errorf("want the working card back asleep on slot 4, got %+v", b)
	}

	// Talking in it again wakes the same card.
	s2.Apply(Event{SessionID: "busy", Cwd: "/beta", HookEventName: "UserPromptSubmit"})
	if n := len(snapshot(s2)); n != 2 || slotOf(s2, "busy") != 4 {
		t.Fatalf("want the same card woken on slot 4, got %+v", snapshot(s2))
	}
}

func TestPruneDeletedSessions(t *testing.T) {
	dir := t.TempDir()
	kept, deleted := filepath.Join(dir, "kept.jsonl"), filepath.Join(dir, "deleted.jsonl")
	for _, p := range []string{kept, deleted} {
		if err := os.WriteFile(p, []byte("{}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	s, now := newTestStore()
	s.Apply(Event{SessionID: "kept", Cwd: "/a", TranscriptPath: kept, HookEventName: "Stop"})
	s.Apply(Event{SessionID: "deleted", Cwd: "/b", TranscriptPath: deleted, HookEventName: "Stop"})
	s.Apply(Event{SessionID: "new", Cwd: "/c", TranscriptPath: filepath.Join(dir, "not-yet.jsonl"), HookEventName: "Stop"})
	os.Remove(deleted)

	*now = now.Add(deletedAfter)
	s.Apply(Event{SessionID: "new", HookEventName: "Stop"}) // just active, its transcript may still come
	s.PruneDeleted()
	ids := map[string]bool{}
	for _, sess := range snapshot(s) {
		ids[sess.ID] = true
	}
	if !ids["kept"] || ids["deleted"] || !ids["new"] {
		t.Fatalf("want only the deleted session gone, got %v", ids)
	}
}

func TestRemoveCard(t *testing.T) {
	s, _ := newTestStore()
	s.Apply(Event{SessionID: "a", Cwd: "/p", HookEventName: "Stop"})
	if !s.Remove("a") || len(snapshot(s)) != 0 || s.Remove("a") {
		t.Fatal("want the card removed once")
	}
	// It comes back when the session is used again.
	s.Apply(Event{SessionID: "a", Cwd: "/p", HookEventName: "UserPromptSubmit"})
	if n := len(snapshot(s)); n != 1 {
		t.Fatalf("want the card back, got %d", n)
	}
}

func TestSeenStopsCalling(t *testing.T) {
	s, _ := newTestStore()
	s.readInfo = func(string) transcriptInfo { return transcriptInfo{Asks: true} }

	// A reply that ended with a question: opening the card counts as noticing it.
	s.Apply(Event{SessionID: "q", Cwd: "/p", TranscriptPath: "/t.jsonl", HookEventName: "Stop"})
	s.Seen("q")
	if got := snapshot(s)[0]; got.Status != StatusDone {
		t.Fatalf("want an opened question to stop calling, got %+v", got)
	}

	// The agent can't go on without an answer to these, so they keep calling.
	waits := []Event{
		{SessionID: "q", HookEventName: "Notification", Message: "Claude needs your permission to use Bash"},
		{SessionID: "q", HookEventName: "PreToolUse", ToolName: "AskUserQuestion"},
		{SessionID: "q", HookEventName: "PreToolUse", ToolName: "ExitPlanMode"},
	}
	for _, ev := range waits {
		s.Apply(ev)
		s.Seen("q")
		if got := snapshot(s)[0]; got.Status != StatusWaiting {
			t.Fatalf("%s %s: want still waiting after opening, got %+v", ev.HookEventName, ev.ToolName, got)
		}
	}
}
