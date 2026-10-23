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

	"aidesk/internal/state"
)

//go:embed web
var webFS embed.FS

type Server struct {
	store *state.Store

	mu   sync.Mutex
	subs map[chan struct{}]struct{}
}

func New(store *state.Store) *Server {
	s := &Server{store: store, subs: map[chan struct{}]struct{}{}}
	store.OnChange(s.notify)
	return s
}

type stateResponse struct {
	Now      time.Time       `json:"now"`
	Sessions []state.Session `json:"sessions"`
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	web, _ := fs.Sub(webFS, "web")
	mux.Handle("GET /", http.FileServerFS(web))
	mux.HandleFunc("POST /event", s.handleEvent)
	mux.HandleFunc("GET /api/state", s.handleState)
	mux.HandleFunc("GET /api/stream", s.handleStream)
	return mux
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
	return stateResponse{Now: time.Now(), Sessions: s.store.Snapshot()}
}

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}
