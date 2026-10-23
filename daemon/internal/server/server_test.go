package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aidesk/internal/settings"
	"aidesk/internal/state"
)

func request(t *testing.T, h http.Handler, method, path, body string) int {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
	return rec.Code
}

func pinned(store *state.Store) map[string]bool {
	out := map[string]bool{}
	for _, s := range store.Snapshot(0) {
		out[s.ID] = s.Pinned
	}
	return out
}

func TestManualModeKeepsScreenAndPins(t *testing.T) {
	store := state.NewStore(nil)
	h := New(store, settings.Load("")).Handler()

	request(t, h, "POST", "/event", `{"session_id":"a","cwd":"/a","hook_event_name":"SessionStart"}`)
	request(t, h, "POST", "/event", `{"session_id":"b","cwd":"/b","hook_event_name":"SessionStart"}`)
	if p := pinned(store); p["a"] || p["b"] {
		t.Fatalf("auto mode should not pin anything: %v", p)
	}

	// Switching to manual keeps what was on screen.
	if code := request(t, h, "PUT", "/api/settings", `{"session_mode":"manual"}`); code != http.StatusNoContent {
		t.Fatalf("switch to manual: %d", code)
	}
	if p := pinned(store); !p["a"] || !p["b"] {
		t.Fatalf("sessions on screen should stay pinned: %v", p)
	}

	// A new session doesn't get on screen by itself.
	request(t, h, "POST", "/event", `{"session_id":"c","cwd":"/c","hook_event_name":"SessionStart"}`)
	if pinned(store)["c"] {
		t.Fatal("new session should not be pinned in manual mode")
	}

	// The user picks and drops sessions.
	if code := request(t, h, "POST", "/api/pin", `{"id":"c","pinned":true}`); code != http.StatusNoContent {
		t.Fatalf("pin: %d", code)
	}
	if code := request(t, h, "POST", "/api/pin", `{"id":"a","pinned":false}`); code != http.StatusNoContent {
		t.Fatalf("unpin: %d", code)
	}
	if p := pinned(store); p["a"] || !p["b"] || !p["c"] {
		t.Fatalf("unexpected pins: %v", p)
	}
	if code := request(t, h, "POST", "/api/pin", `{"id":"nope","pinned":true}`); code != http.StatusNotFound {
		t.Fatalf("unknown session should be 404, got %d", code)
	}

	// Saving manual again (e.g. another setting) must not re-pin everything.
	request(t, h, "PUT", "/api/settings", `{"session_mode":"manual","cards_per_page":4}`)
	if pinned(store)["a"] {
		t.Fatal("re-saving manual mode should keep the user's choice")
	}
}

func TestInvalidSessionModeRejected(t *testing.T) {
	h := New(state.NewStore(nil), settings.Load("")).Handler()
	if code := request(t, h, "PUT", "/api/settings", `{"session_mode":"sometimes"}`); code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", code)
	}
}
