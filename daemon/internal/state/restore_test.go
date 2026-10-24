package state

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestRestoreRunning(t *testing.T) {
	root := t.TempDir()
	sessions := filepath.Join(root, "sessions")
	project := filepath.Join(root, "projects", "C--Projects-shop")
	for _, dir := range []string{sessions, project} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write := func(path, data string) {
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	alive, dead := os.Getpid(), 0x7ffffff0
	file := func(pid int, id, status, extra string) string {
		return fmt.Sprintf(`{"pid":%d,"sessionId":%q,"cwd":"C:\\Projects\\shop","name":"Name of %s","status":%q,"statusUpdatedAt":1792861439512%s}`, pid, id, id, status, extra)
	}
	write(filepath.Join(sessions, "1.json"), file(alive, "busy", "busy", `,"entrypoint":"claude-desktop","hostSessionId":"local_abc"`))
	write(filepath.Join(sessions, "2.json"), file(alive, "asks", "idle", ""))
	write(filepath.Join(sessions, "3.json"), file(alive, "stopped", "idle", ""))
	write(filepath.Join(sessions, "4.json"), file(alive, "done", "idle", ""))
	write(filepath.Join(sessions, "5.json"), file(dead, "gone", "idle", ""))
	write(filepath.Join(project, "asks.jsonl"), `{"type":"assistant","message":{"content":[{"type":"text","text":"Commit it?"}]}}`+"\n")
	write(filepath.Join(project, "stopped.jsonl"), `{"type":"user","message":{"content":[{"type":"text","text":"[Request interrupted by user]"}]}}`+"\n")

	s, _ := newTestStore()
	s.RestoreRunning(sessions)

	got := map[string]Session{}
	for _, sess := range snapshot(s) {
		got[sess.ID] = sess
	}
	if _, ok := got["gone"]; ok || len(got) != 4 {
		t.Fatalf("want the 4 running sessions, got %v", got)
	}
	want := map[string]Status{"busy": StatusWorking, "asks": StatusWaiting, "stopped": StatusInterrupted, "done": StatusDone}
	for id, st := range want {
		if got[id].Status != st {
			t.Errorf("%s: want %s, got %+v", id, st, got[id])
		}
	}
	if b := got["busy"]; b.Title != "Name of busy" || b.HostID != "local_abc" || b.Client != "claude-desktop" || b.Project != "shop" {
		t.Errorf("unexpected restored session: %+v", b)
	}

	// Hook events for a restored session update it rather than adding another.
	s.Apply(Event{SessionID: "done", Cwd: `C:\Projects\shop`, HookEventName: "UserPromptSubmit"})
	if n := len(snapshot(s)); n != 4 {
		t.Fatalf("want 4 sessions after an event, got %d", n)
	}
}

func TestPinsSurviveRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pins.json")

	s, _ := newTestStore()
	s.UsePins(LoadPins(path))
	s.Apply(Event{SessionID: "a", Cwd: "/p", HookEventName: "UserPromptSubmit"})
	s.Apply(Event{SessionID: "b", Cwd: "/q", HookEventName: "UserPromptSubmit"})
	s.Pin("a", true)

	// A new daemon: pinned sessions come back pinned, others don't.
	s2, _ := newTestStore()
	s2.UsePins(LoadPins(path))
	s2.Apply(Event{SessionID: "a", Cwd: "/p", HookEventName: "Stop"})
	s2.Apply(Event{SessionID: "b", Cwd: "/q", HookEventName: "Stop"})
	for _, sess := range snapshot(s2) {
		if sess.Pinned != (sess.ID == "a") {
			t.Fatalf("unexpected pin after restart: %+v", sess)
		}
	}

	// A desktop session is pinned by the app's id, so it's found by that too.
	s2.findHost = func(id string) string { return "local_" + id }
	s2.Apply(Event{SessionID: "c", Cwd: "/r", HookEventName: "UserPromptSubmit"})
	s2.Pin("c", true)
	s3, _ := newTestStore()
	s3.UsePins(LoadPins(path))
	s3.findHost = func(string) string { return "local_c" }
	s3.Apply(Event{SessionID: "c-after-clear", Cwd: "/r", HookEventName: "UserPromptSubmit"})
	if !snapshot(s3)[0].Pinned {
		t.Fatalf("want the session pinned by the app's id, got %+v", snapshot(s3)[0])
	}

	// Unpinning is saved too.
	s3.Pin("c-after-clear", false)
	if LoadPins(path).has("local_c") {
		t.Fatal("want the pin removed from the file")
	}
}
