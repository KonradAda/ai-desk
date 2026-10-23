// Package server exposes the session state over HTTP: hook events in,
// JSON state and a live browser simulator of the device screen out.
package server

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"sync"
	"time"

	"aidesk/internal/settings"
	"aidesk/internal/state"
)

//go:embed web
var webFS embed.FS

type Server struct {
	store    *state.Store
	settings *settings.Store
	// open hands a link to the OS; replaced in tests.
	open func(url string) error

	mu   sync.Mutex
	subs map[chan struct{}]struct{}
}

func New(store *state.Store, prefs *settings.Store) *Server {
	s := &Server{store: store, settings: prefs, open: openURL, subs: map[chan struct{}]struct{}{}}
	store.OnChange(s.notify)
	return s
}

type stateResponse struct {
	Now      time.Time         `json:"now"`
	Sessions []state.Session   `json:"sessions"`
	Settings settings.Settings `json:"settings"`
	// Languages lists the translations the display may offer in its settings.
	Languages []string `json:"languages"`
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	web, _ := fs.Sub(webFS, "web")
	mux.Handle("GET /", http.FileServerFS(web))
	mux.HandleFunc("POST /event", s.handleEvent)
	mux.HandleFunc("GET /api/state", s.handleState)
	mux.HandleFunc("GET /api/stream", s.handleStream)
	mux.HandleFunc("POST /api/move", s.handleMove)
	mux.HandleFunc("GET /api/settings", s.handleGetSettings)
	mux.HandleFunc("PUT /api/settings", s.handlePutSettings)
	mux.HandleFunc("POST /api/pin", s.handlePin)
	mux.HandleFunc("POST /api/open", s.handleOpen)
	return mux
}

// sameOrigin rejects requests sent by other web pages open in the user's
// browser: only the simulator itself (or a client without a browser, like the
// device) may make the daemon open things.
func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	return origin == "" || origin == "http://"+r.Host
}

// handleOpen brings the user to an agent's session: for sessions in the Claude
// desktop app it opens the app's claude:// link to that session. Sessions in a
// terminal can't be opened yet.
func (s *Server) handleOpen(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	hostID, ok := s.store.HostID(req.ID)
	if !ok {
		http.Error(w, "unknown session", http.StatusNotFound)
		return
	}
	if !state.ValidHostID(hostID) {
		http.Error(w, "this session can't be opened (not in the Claude app)", http.StatusConflict)
		return
	}
	if err := s.open("claude://code/continue?session=" + hostID + "&source=aidesk"); err != nil {
		log.Printf("open session %s: %v", shortID(req.ID), err)
		http.Error(w, "couldn't open the session", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handlePin puts a session on the screen or takes it off (manual mode).
func (s *Server) handlePin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID     string `json:"id"`
		Pinned bool   `json:"pinned"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if !s.store.Pin(req.ID, req.Pinned) {
		http.Error(w, "unknown session", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.settings.Get())
}

func (s *Server) handlePutSettings(w http.ResponseWriter, r *http.Request) {
	prev := s.settings.Get()
	next := prev // fields left out of the request keep their value
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&next); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := s.settings.Set(next); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if prev.SessionMode != "manual" && next.SessionMode == "manual" {
		s.store.PinAll() // keep what's on screen when switching to manual
	}
	s.notify()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMove(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID   string `json:"id"`
		Slot int    `json:"slot"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if !s.store.Move(req.ID, req.Slot) {
		http.Error(w, "unknown session or slot", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleEvent(w http.ResponseWriter, r *http.Request) {
	var ev state.Event
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&ev); err != nil {
		http.Error(w, "bad event", http.StatusBadRequest)
		return
	}
	log.Printf("%-16s %s %s", ev.HookEventName, shortID(ev.SessionID), ev.ToolName)
	s.store.Apply(ev)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.snapshot())
}

// handleStream pushes the full state as Server-Sent Events whenever it changes.
func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")

	ch := make(chan struct{}, 1)
	s.mu.Lock()
	s.subs[ch] = struct{}{}
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.subs, ch)
		s.mu.Unlock()
	}()

	// Periodic refresh so stale sessions disappear without new events.
	tick := time.NewTicker(30 * time.Second)
	defer tick.Stop()

	for {
		data, _ := json.Marshal(s.snapshot())
		if _, err := w.Write([]byte("data: " + string(data) + "\n\n")); err != nil {
			return
		}
		flusher.Flush()
		select {
		case <-r.Context().Done():
			return
		case <-ch:
		case <-tick.C:
		}
	}
}

func (s *Server) notify() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for ch := range s.subs {
		select {
		case ch <- struct{}{}:
		default: // an update is already pending for this subscriber
		}
	}
}

func (s *Server) snapshot() stateResponse {
	prefs := s.settings.Get()
	return stateResponse{
		Now:       time.Now(),
		Sessions:  s.store.Snapshot(prefs.HideDoneAfter()),
		Settings:  prefs,
		Languages: settings.Languages,
	}
}

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}
