package state

import (
	"encoding/json"
	"testing"
	"time"
)

func newTestStore() (*Store, *time.Time) {
	s := NewStore()
	now := time.Date(2026, 10, 23, 14, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	return s, &now
}

func TestSessionLifecycle(t *testing.T) {
	s, now := newTestStore()
	cwd := `C:\Projects\api-backend`

	s.Apply(Event{SessionID: "a", Cwd: cwd, HookEventName: "SessionStart"})
	s.Apply(Event{SessionID: "a", Cwd: cwd, HookEventName: "UserPromptSubmit"})
	s.Apply(Event{SessionID: "a", Cwd: cwd, HookEventName: "PreToolUse", ToolName: "Edit",
		ToolInput: json.RawMessage(`{"file_path":"C:\\Projects\\api-backend\\auth.ts"}`)})

	got := s.Snapshot()
	if len(got) != 1 {
		t.Fatalf("want 1 session, got %d", len(got))
	}
	if got[0].Project != "api-backend" || got[0].Status != StatusWorking || got[0].Action != "Edytuje auth.ts" {
		t.Fatalf("unexpected session: %+v", got[0])
	}

	s.Apply(Event{SessionID: "a", HookEventName: "Notification", Message: "Claude needs your permission to use Bash"})
	if got := s.Snapshot()[0]; got.Status != StatusWaiting || got.Action != "Prosi o zgodę: Bash" {
		t.Fatalf("unexpected waiting session: %+v", got)
	}

	s.Apply(Event{SessionID: "a", HookEventName: "PostToolUse"})
	if st := s.Snapshot()[0].Status; st != StatusWorking {
		t.Fatalf("want working after tool ran, got %s", st)
	}

	*now = now.Add(5 * time.Minute)
	s.Apply(Event{SessionID: "a", HookEventName: "Stop"})
	if got := s.Snapshot()[0]; got.Status != StatusDone || !got.FinishedAt.Equal(*now) {
		t.Fatalf("unexpected done session: %+v", got)
	}

	s.Apply(Event{SessionID: "a", HookEventName: "SessionEnd"})
	if n := len(s.Snapshot()); n != 0 {
		t.Fatalf("want session removed, got %d", n)
	}
}

func TestTodoSteps(t *testing.T) {
	s, _ := newTestStore()
	s.Apply(Event{SessionID: "a", Cwd: "/p", HookEventName: "PreToolUse", ToolName: "TodoWrite",
		ToolInput: json.RawMessage(`{"todos":[{"status":"completed"},{"status":"completed"},{"status":"in_progress"},{"status":"pending"}]}`)})
	st := s.Snapshot()[0].Steps
	if st == nil || st.Done != 2 || st.Total != 4 {
		t.Fatalf("want 2/4 steps, got %+v", st)
	}
}

func TestSnapshotLimitsAndPrioritizes(t *testing.T) {
	s, now := newTestStore()
	for i := range 7 {
		*now = now.Add(time.Minute)
		s.Apply(Event{SessionID: string(rune('a' + i)), Cwd: "/p", HookEventName: "Stop"})
	}
	// The oldest session asks for attention, so it must stay on screen.
	s.Apply(Event{SessionID: "a", HookEventName: "Notification"})

	got := s.Snapshot()
	if len(got) != MaxCards {
		t.Fatalf("want %d cards, got %d", MaxCards, len(got))
	}
	if got[0].ID != "a" {
		t.Fatalf("want waiting session first, got %s", got[0].ID)
	}
}

func TestStaleSessionsDropped(t *testing.T) {
	s, now := newTestStore()
	s.Apply(Event{SessionID: "a", Cwd: "/p", HookEventName: "Stop"})
	*now = now.Add(staleAfter + time.Minute)
	if n := len(s.Snapshot()); n != 0 {
		t.Fatalf("want stale session dropped, got %d", n)
	}
}

func TestMascotStablePerProject(t *testing.T) {
	if mascotFor("api-backend") != mascotFor("API-Backend") {
		t.Fatal("mascot should not depend on case")
	}
}

func TestMascotsUniqueOnScreen(t *testing.T) {
	s, _ := newTestStore()
	for i := range MaxCards {
		s.Apply(Event{SessionID: string(rune('a' + i)), Cwd: "/p" + string(rune('a'+i)), HookEventName: "SessionStart"})
	}
	seen := map[int]bool{}
	for _, sess := range s.Snapshot() {
		if seen[sess.Mascot] {
			t.Fatalf("mascot %d used twice", sess.Mascot)
		}
		seen[sess.Mascot] = true
	}

	// A second session in the same project keeps the project's mascot.
	s.Apply(Event{SessionID: "z", Cwd: "/pa", HookEventName: "SessionStart"})
	var first, second int
	for _, sess := range s.Snapshot() {
		switch sess.ID {
		case "a":
			first = sess.Mascot
		case "z":
			second = sess.Mascot
		}
	}
	if first != second {
		t.Fatalf("same project should share mascot: %d vs %d", first, second)
	}
}
