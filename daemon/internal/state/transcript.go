package state

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
)

// titleTail is how much of the end of a transcript is scanned for the title.
// Claude Code re-appends the title entry as the session goes on, so the tail
// holds the current one without reading files that grow to many megabytes.
const titleTail = 256 << 10

// readTranscriptTitle returns the latest session title ("custom-title" entry)
// from a Claude Code transcript, or "" if there is none.
func readTranscriptTitle(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	if info, err := f.Stat(); err == nil && info.Size() > titleTail {
		f.Seek(info.Size()-titleTail, io.SeekStart)
	}
	data, err := io.ReadAll(io.LimitReader(f, titleTail))
	if err != nil {
		return ""
	}
	return lastTitle(data)
}

func lastTitle(data []byte) string {
	lines := bytes.Split(data, []byte("\n"))
	for i := len(lines) - 1; i >= 0; i-- {
		if !bytes.Contains(lines[i], []byte(`"custom-title"`)) {
			continue
		}
		var entry struct {
			Type        string `json:"type"`
			CustomTitle string `json:"customTitle"`
		}
		if json.Unmarshal(lines[i], &entry) == nil && entry.Type == "custom-title" && entry.CustomTitle != "" {
			return entry.CustomTitle
		}
	}
	return ""
}
