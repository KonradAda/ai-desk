package state

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// Layout remembers which screen slot the user put each project in,
// so a project comes back to the same place in later sessions.
type Layout struct {
	path  string         // empty means in-memory only
	Slots map[string]int `json:"slots"`
}

// LoadLayout reads the layout file; a missing or broken file gives an empty layout.
func LoadLayout(path string) *Layout {
	l := &Layout{path: path, Slots: map[string]int{}}
	if path == "" {
		return l
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return l
	}
	if err := json.Unmarshal(data, l); err != nil || l.Slots == nil {
		log.Printf("layout: ignoring unreadable %s: %v", path, err)
		l.Slots = map[string]int{}
	}
	return l
}

func (l *Layout) preferred(project string) (int, bool) {
	slot, ok := l.Slots[layoutKey(project)]
	return slot, ok && slot >= 0 && slot < MaxSlots
}

// reservedBy reports whether another project prefers this slot.
func (l *Layout) reservedBy(slot int, project string) bool {
	key := layoutKey(project)
	for p, s := range l.Slots {
		if s == slot && p != key {
			return true
		}
	}
	return false
}

func (l *Layout) set(project string, slot int) {
	key := layoutKey(project)
	for p, s := range l.Slots {
		if s == slot && p != key {
			delete(l.Slots, p) // one project per slot
		}
	}
	l.Slots[key] = slot
}

func (l *Layout) save() {
	if l.path == "" {
		return
	}
	data, _ := json.MarshalIndent(l, "", "  ")
	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		log.Printf("layout: %v", err)
		return
	}
	if err := os.WriteFile(l.path, data, 0o644); err != nil {
		log.Printf("layout: %v", err)
	}
}

func layoutKey(project string) string { return strings.ToLower(project) }
