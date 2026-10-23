package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newTestStore() (*Store, *time.Time) {
	s := NewStore(nil)
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

	got := snapshot(s)
	if len(got) != 1 {
		t.Fatalf("want 1 session, got %d", len(got))
	}
	if got[0].Project != "api-backend" || got[0].Status != StatusWorking || got[0].Action != "Edytuje auth.ts" {
		t.Fatalf("unexpected session: %+v", got[0])
	}

	s.Apply(Event{SessionID: "a", HookEventName: "Notification", Message: "Claude needs your permission to use Bash"})
	if got := snapshot(s)[0]; got.Status != StatusWaiting || got.Action != "Prosi o zgodę: Bash" {
		t.Fatalf("unexpected waiting session: %+v", got)
	}

	s.Apply(Event{SessionID: "a", HookEventName: "PostToolUse"})
	if st := snapshot(s)[0].Status; st != StatusWorking {
		t.Fatalf("want working after tool ran, got %s", st)
	}

	*now = now.Add(5 * time.Minute)
	s.Apply(Event{SessionID: "a", HookEventName: "Stop"})
	if got := snapshot(s)[0]; got.Status != StatusDone || !got.FinishedAt.Equal(*now) {
		t.Fatalf("unexpected done session: %+v", got)
	}

	s.Apply(Event{SessionID: "a", HookEventName: "SessionEnd"})
	if n := len(snapshot(s)); n != 0 {
		t.Fatalf("want session removed, got %d", n)
	}
}

func TestTodoSteps(t *testing.T) {
	s, _ := newTestStore()
	s.Apply(Event{SessionID: "a", Cwd: "/p", HookEventName: "PreToolUse", ToolName: "TodoWrite",
		ToolInput: json.RawMessage(`{"todos":[{"status":"completed"},{"status":"completed"},{"status":"in_progress"},{"status":"pending"}]}`)})
	st := snapshot(s)[0].Steps
	if st == nil || st.Done != 2 || st.Total != 4 {
		t.Fatalf("want 2/4 steps, got %+v", st)
	}
}

// snapshot is Snapshot without the hidden count, for tests that don't need it.
func snapshot(s *Store) []Session {
	v, _ := s.Snapshot()
	return v
}

func slotOf(s *Store, id string) int {
	for _, sess := range snapshot(s) {
		if sess.ID == id {
			return sess.Slot
		}
	}
	return -1
}

func start(s *Store, id, cwd string) {
	s.Apply(Event{SessionID: id, Cwd: cwd, HookEventName: "SessionStart"})
}

func TestSlotsStayFixedWhileAgentsWork(t *testing.T) {
	s, now := newTestStore()
	start(s, "a", "/pa")
	start(s, "b", "/pb")
	start(s, "c", "/pc")
	before := map[string]int{"a": slotOf(s, "a"), "b": slotOf(s, "b"), "c": slotOf(s, "c")}

	// Activity in any order must not reshuffle the cards.
	for _, id := range []string{"c", "a", "b", "c"} {
		*now = now.Add(time.Minute)
		s.Apply(Event{SessionID: id, HookEventName: "PreToolUse", ToolName: "Read"})
	}
	s.Apply(Event{SessionID: "b", HookEventName: "Notification"})
	for id, slot := range before {
		if got := slotOf(s, id); got != slot {
			t.Fatalf("session %s moved from slot %d to %d", id, slot, got)
		}
	}
}

func TestOverflowWaitsForFreeSlot(t *testing.T) {
	s, _ := newTestStore()
	for i := range MaxCards + 1 {
		start(s, string(rune('a'+i)), "/p"+string(rune('a'+i)))
	}
	visible, hidden := s.Snapshot()
	if len(visible) != MaxCards || hidden != 1 {
		t.Fatalf("want %d visible and 1 hidden, got %d and %d", MaxCards, len(visible), hidden)
	}
	freed := slotOf(s, "c")
	s.Apply(Event{SessionID: "c", HookEventName: "SessionEnd"})
	if got := slotOf(s, "g"); got != freed {
		t.Fatalf("hidden session should take freed slot %d, got %d", freed, got)
	}
}

func TestMoveSwapsAndIsRemembered(t *testing.T) {
	s, _ := newTestStore()
	start(s, "a", "/alpha")
	start(s, "b", "/beta")
	slotA, slotB := slotOf(s, "a"), slotOf(s, "b")

	if !s.Move("a", slotB) {
		t.Fatal("move failed")
	}
	if slotOf(s, "a") != slotB || slotOf(s, "b") != slotA {
		t.Fatalf("cards should swap: a=%d b=%d", slotOf(s, "a"), slotOf(s, "b"))
	}
	if !s.Move("a", 5) || slotOf(s, "a") != 5 {
		t.Fatal("move to empty slot failed")
	}

	// A new session of the same project returns to its remembered slot.
	s.Apply(Event{SessionID: "a", HookEventName: "SessionEnd"})
	start(s, "a2", "/alpha")
	if got := slotOf(s, "a2"); got != 5 {
		t.Fatalf("want remembered slot 5, got %d", got)
	}

	if s.Move("nope", 1) || s.Move("a2", MaxCards) {
		t.Fatal("invalid moves should be rejected")
	}
}

func TestNewProjectAvoidsClaimedSlots(t *testing.T) {
	s, _ := newTestStore()
	start(s, "a", "/alpha")
	s.Move("a", 0)
	s.Apply(Event{SessionID: "a", HookEventName: "SessionEnd"})

	// Slot 0 is free but belongs to alpha, so a new project goes elsewhere.
	start(s, "b", "/beta")
	if got := slotOf(s, "b"); got == 0 {
		t.Fatal("new project took a slot claimed by another project")
	}
}

func TestLayoutPersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "layout.json")
	s := NewStore(LoadLayout(path))
	start(s, "a", "/alpha")
	s.Move("a", 4)

	s2 := NewStore(LoadLayout(path))
	start(s2, "x", "/Alpha")
	if got := slotOf(s2, "x"); got != 4 {
		t.Fatalf("want slot 4 loaded from disk, got %d", got)
	}
}

func TestStaleSessionsDropped(t *testing.T) {
	s, now := newTestStore()
	s.Apply(Event{SessionID: "a", Cwd: "/p", HookEventName: "Stop"})
	*now = now.Add(staleAfter + time.Minute)
	if n := len(snapshot(s)); n != 0 {
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
	for _, sess := range snapshot(s) {
		if seen[sess.Mascot] {
			t.Fatalf("mascot %d used twice", sess.Mascot)
		}
		seen[sess.Mascot] = true
	}

	// A second session in the same project still gets its own mascot.
	s.Apply(Event{SessionID: "b", HookEventName: "SessionEnd"}) // free a slot so it is visible
	s.Apply(Event{SessionID: "z", Cwd: "/pa", HookEventName: "SessionStart"})
	var first, second int
	for _, sess := range snapshot(s) {
		switch sess.ID {
		case "a":
			first = sess.Mascot
		case "z":
			second = sess.Mascot
		}
	}
	if first == second {
		t.Fatalf("two sessions of one project should look different, both got mascot %d", first)
	}
}

func TestTitleFromPromptThenTranscript(t *testing.T) {
	s, now := newTestStore()
	title := ""
	reads := 0
	s.readTitle = func(string) string { reads++; return title }

	ev := Event{SessionID: "a", Cwd: "/p", TranscriptPath: "/t.jsonl", HookEventName: "UserPromptSubmit",
		Prompt: "Napraw logowanie w panelu admina, bo po zmianie hasła wywala błąd 500"}
	s.Apply(ev)
	if got := snapshot(s)[0].Title; got != "Napraw logowanie w panelu admina, bo po…" {
		t.Fatalf("want prompt fallback, got %q", got)
	}

	// The transcript is not re-read on every event.
	title = "Naprawa logowania"
	s.Apply(Event{SessionID: "a", TranscriptPath: "/t.jsonl", HookEventName: "PreToolUse"})
	if reads != 1 {
		t.Fatalf("want 1 transcript read, got %d", reads)
	}

	*now = now.Add(titleRefresh)
	s.Apply(Event{SessionID: "a", TranscriptPath: "/t.jsonl", HookEventName: "PreToolUse"})
	if got := snapshot(s)[0].Title; got != "Naprawa logowania" {
		t.Fatalf("want transcript title, got %q", got)
	}

	// A later prompt doesn't replace an existing title.
	*now = now.Add(time.Second)
	s.Apply(Event{SessionID: "a", HookEventName: "UserPromptSubmit", Prompt: "a teraz testy"})
	if got := snapshot(s)[0].Title; got != "Naprawa logowania" {
		t.Fatalf("title should stay, got %q", got)
	}
}

func TestLastTitle(t *testing.T) {
	data := []byte(`{"type":"user","message":"mentions \"custom-title\" in text"}
{"type":"custom-title","customTitle":"Stary tytuł","sessionId":"x"}
{"type":"assistant"}
{"type":"custom-title","customTitle":"Nowy tytuł","sessionId":"x"}
{"type":"user","message":"another \"custom-title\" mention"}
{"type":"custom-ti`) // the tail may cut a line in half
	if got := lastTitle(data); got != "Nowy tytuł" {
		t.Fatalf("want newest title, got %q", got)
	}
	if got := lastTitle([]byte(`{"type":"user"}`)); got != "" {
		t.Fatalf("want empty title, got %q", got)
	}
}

func TestProjectNameUsesGitRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "my-repo")
	sub := filepath.Join(root, "daemon", "internal")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := projectName(sub); got != "my-repo" {
		t.Fatalf("want my-repo, got %s", got)
	}
	if got := projectName(filepath.Join(t.TempDir(), "plain")); got != "plain" {
		t.Fatalf("want plain, got %s", got)
	}
}
