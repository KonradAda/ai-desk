// Package settings holds the user's display preferences, shared by the
// browser simulator and the device and saved next to the layout file.
package settings

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"
)

// Languages the display has translations for. The first one is the default.
var Languages = []string{"en"}

// Session modes: "auto" shows every agent as soon as it starts, "manual" only
// the ones the user picked. The first is the default.
var SessionModes = []string{"auto", "manual"}

type Settings struct {
	Language string `json:"language"`
	// CardsPerPage is how many agents one screen shows; more agents go to further pages.
	CardsPerPage int `json:"cards_per_page"`
	// JumpToWaiting switches to the page of an agent that starts waiting for the user.
	JumpToWaiting bool `json:"jump_to_waiting"`
	// HideDoneAfterMin hides finished agents after this many minutes (0 = never).
	HideDoneAfterMin int `json:"hide_done_after_min"`
	// SessionMode is how agents get on the screen, see SessionModes.
	SessionMode string `json:"session_mode"`
}

func Defaults() Settings {
	return Settings{Language: Languages[0], CardsPerPage: 6, JumpToWaiting: true, SessionMode: SessionModes[0]}
}

func (s Settings) Validate() error {
	if !slices.Contains(Languages, s.Language) {
		return fmt.Errorf("unsupported language %q", s.Language)
	}
	if !slices.Contains(SessionModes, s.SessionMode) {
		return fmt.Errorf("unsupported session_mode %q", s.SessionMode)
	}
	if s.CardsPerPage < 1 || s.CardsPerPage > 6 {
		return fmt.Errorf("cards_per_page must be 1-6, got %d", s.CardsPerPage)
	}
	if s.HideDoneAfterMin < 0 || s.HideDoneAfterMin > 24*60 {
		return fmt.Errorf("hide_done_after_min must be 0-1440, got %d", s.HideDoneAfterMin)
	}
	return nil
}

func (s Settings) HideDoneAfter() time.Duration {
	return time.Duration(s.HideDoneAfterMin) * time.Minute
}

type Store struct {
	mu   sync.Mutex
	path string // empty means in-memory only
	cur  Settings
}

// Load reads settings from path; missing or invalid values fall back to defaults.
func Load(path string) *Store {
	st := &Store{path: path, cur: Defaults()}
	if path == "" {
		return st
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return st
	}
	loaded := Defaults()
	if err := json.Unmarshal(data, &loaded); err != nil {
		log.Printf("settings: ignoring unreadable %s: %v", path, err)
		return st
	}
	if err := loaded.Validate(); err != nil {
		log.Printf("settings: ignoring %s: %v", path, err)
		return st
	}
	st.cur = loaded
	return st
}

func (st *Store) Get() Settings {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.cur
}

func (st *Store) Set(s Settings) error {
	if err := s.Validate(); err != nil {
		return err
	}
	st.mu.Lock()
	st.cur = s
	st.mu.Unlock()
	if st.path == "" {
		return nil
	}
	data, _ := json.MarshalIndent(s, "", "  ")
	if err := os.MkdirAll(filepath.Dir(st.path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(st.path, data, 0o644)
}
