package state

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
)

// transcriptTail is how much of the end of a transcript is scanned.
// Claude Code re-appends the title entry as the session goes on and every
// reply carries the model, so the tail holds the current values without
// reading files that grow to many megabytes.
const transcriptTail = 256 << 10

// transcriptInfo is what the display shows about a session beyond hook events.
type transcriptInfo struct {
	Title      string // session title ("custom-title" entry)
	Model      string // e.g. claude-opus-5-5
	Effort     string // e.g. medium
	Entrypoint string // where the session runs, e.g. claude-desktop or cli
}

// readTranscriptInfo reads the latest session info from a Claude Code
// transcript. Missing values are left empty.
func readTranscriptInfo(path string) transcriptInfo {
	f, err := os.Open(path)
	if err != nil {
		return transcriptInfo{}
	}
	defer f.Close()
	if info, err := f.Stat(); err == nil && info.Size() > transcriptTail {
		f.Seek(info.Size()-transcriptTail, io.SeekStart)
	}
	data, err := io.ReadAll(io.LimitReader(f, transcriptTail))
	if err != nil {
		return transcriptInfo{}
	}
	return parseTranscriptTail(data)
}

func parseTranscriptTail(data []byte) transcriptInfo {
	var info transcriptInfo
	lines := bytes.Split(data, []byte("\n"))
	for i := len(lines) - 1; i >= 0 && (info.Title == "" || info.Model == ""); i-- {
		line := lines[i]
		switch {
		case info.Title == "" && bytes.Contains(line, []byte(`"custom-title"`)):
			var entry struct {
				Type        string `json:"type"`
				CustomTitle string `json:"customTitle"`
			}
			if json.Unmarshal(line, &entry) == nil && entry.Type == "custom-title" {
				info.Title = entry.CustomTitle
			}
		case info.Model == "" && bytes.Contains(line, []byte(`"type":"assistant"`)):
			var entry struct {
				Type          string `json:"type"`
				IsSidechain   bool   `json:"isSidechain"` // subagent replies may use another model
				Effort        string `json:"effort"`
				PerTurnEffort string `json:"perTurnEffort"`
				Entrypoint    string `json:"entrypoint"`
				Message       struct {
					Model string `json:"model"`
				} `json:"message"`
			}
			if json.Unmarshal(line, &entry) != nil || entry.Type != "assistant" || entry.IsSidechain || entry.Message.Model == "" {
				continue
			}
			info.Model = entry.Message.Model
			info.Effort = entry.PerTurnEffort
			if info.Effort == "" {
				info.Effort = entry.Effort
			}
			info.Entrypoint = entry.Entrypoint
		}
	}
	return info
}
