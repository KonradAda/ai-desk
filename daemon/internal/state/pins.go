package state

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

// Pins remembers which sessions the user put on the screen in manual mode,
// so they stay there after the daemon restarts. A session is saved by the
// desktop app's id when it has one, else by its own id.
type Pins struct {
	path string          // empty means in-memory only
	IDs  map[string]bool `json:"ids"`
}

// LoadPins reads the pins file; a missing or broken file gives no pins.
func LoadPins(path string) *Pins {
	p := &Pins{path: path, IDs: map[string]bool{}}
	if path == "" {
		return p
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return p
	}
	if err := json.Unmarshal(data, p); err != nil || p.IDs == nil {
		log.Printf("pins: ignoring unreadable %s: %v", path, err)
		p.IDs = map[string]bool{}
	}
	return p
}

func (p *Pins) has(id string) bool { return id != "" && p.IDs[id] }

// set pins or unpins a session under all of its ids.
func (p *Pins) set(sess *Session, pinned bool) {
	for _, id := range []string{sess.ID, sess.HostID} {
		delete(p.IDs, id)
	}
	if pinned {
		key := sess.HostID
		if key == "" {
			key = sess.ID
		}
		p.IDs[key] = true
	}
}

func (p *Pins) save() {
	if p.path == "" {
		return
	}
	data, _ := json.MarshalIndent(p, "", "  ")
	if err := os.MkdirAll(filepath.Dir(p.path), 0o755); err != nil {
		log.Printf("pins: %v", err)
		return
	}
	if err := os.WriteFile(p.path, data, 0o644); err != nil {
		log.Printf("pins: %v", err)
	}
}
