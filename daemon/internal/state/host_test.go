package state

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFindHostID(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "100.json"),
		`{"pid":100,"sessionId":"desk-uuid","entrypoint":"claude-desktop","hostSessionId":"local_f79cc0fc-40a1"}`)
	writeFile(t, filepath.Join(dir, "200.json"), `{"pid":200,"sessionId":"cli-uuid","entrypoint":"cli"}`)
	writeFile(t, filepath.Join(dir, "300.json"), `{"pid":300,"sessionId":"odd-uuid","hostSessionId":"local_x\"&calc"}`)
	writeFile(t, filepath.Join(dir, "100.abc.key"), `{"sessionId":"desk-uuid","hostSessionId":"local_from_key"}`)
	writeFile(t, filepath.Join(dir, "broken.json"), `{not json`)

	cases := map[string]string{
		"desk-uuid": "local_f79cc0fc-40a1", // app session
		"cli-uuid":  "",                    // terminal session: nothing to open
		"odd-uuid":  "",                    // id in an unexpected format is never used
		"missing":   "",
	}
	for id, want := range cases {
		if got := findHostID(dir, id); got != want {
			t.Errorf("%s: want %q, got %q", id, want, got)
		}
	}
}

func TestHostIDLookedUpForStore(t *testing.T) {
	s, now := newTestStore()
	calls := 0
	host := ""
	s.findHost = func(string) string { calls++; return host }

	start(s, "a", "/p")
	if snapshot(s)[0].HostID != "" {
		t.Fatal("no host id yet")
	}
	// The session file shows up a moment later; a click looks it up right away.
	host = "local_abc"
	if id, ok := s.HostID("a"); !ok || id != "local_abc" {
		t.Fatalf("want local_abc, got %q %v", id, ok)
	}
	if snapshot(s)[0].HostID != "local_abc" {
		t.Fatal("host id should be kept on the session")
	}
	before := calls
	*now = now.Add(infoRefresh * 2)
	s.Apply(Event{SessionID: "a", HookEventName: "PreToolUse", ToolName: "Read"})
	if calls != before {
		t.Fatal("a known host id should not be looked up again")
	}
	if _, ok := s.HostID("nope"); ok {
		t.Fatal("unknown session should not be found")
	}
}
