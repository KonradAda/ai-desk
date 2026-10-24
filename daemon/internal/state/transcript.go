package state

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
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
	// Asks is whether the last turn ended with a question for the user.
	Asks bool
}

// readTranscriptInfo reads the latest session info from a Claude Code
// transcript. Missing values are left empty.
func readTranscriptInfo(path string) transcriptInfo {
	data, err := readTail(path, transcriptTail)
	if err != nil {
		return transcriptInfo{}
	}
	return parseTranscriptTail(data)
}

// readTail reads at most the last n bytes of a file.
func readTail(path string, n int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if info, err := f.Stat(); err == nil && info.Size() > n {
		f.Seek(info.Size()-n, io.SeekStart)
	}
	return io.ReadAll(io.LimitReader(f, n))
}

// interruptTail is how much of a transcript's end is checked for an
// interrupt; it only needs the last few entries.
const interruptTail = 64 << 10

// readInterrupted says whether the user interrupted the session's last reply.
func readInterrupted(path string) bool {
	data, err := readTail(path, interruptTail)
	return err == nil && lastEntryInterrupted(bytes.Split(data, []byte("\n")))
}

// lastEntryInterrupted says whether the last message in the transcript is the
// entry Claude Code writes when the user stops a reply ("[Request interrupted
// by user]", or "... for tool use" when a tool was running).
func lastEntryInterrupted(lines [][]byte) bool {
	for i := len(lines) - 1; i >= 0; i-- {
		kind, blocks, ok := parseMessage(lines[i])
		if !ok {
			continue
		}
		if kind != "user" {
			return false
		}
		for _, b := range blocks {
			if b.Type == "text" && strings.HasPrefix(b.Text, "[Request interrupted by user") {
				return true
			}
		}
		return false
	}
	return false
}

// contentBlock is one part of a message's content.
type contentBlock struct {
	Type  string `json:"type"`
	Text  string `json:"text"`
	Name  string `json:"name"`
	Input struct {
		Ask string `json:"ask"`
	} `json:"input"`
}

// parseMessage reads a transcript line holding a user or assistant message
// of the main conversation (not a subagent's). A message whose content is
// plain text comes back as one text block. ok is false for other lines.
func parseMessage(line []byte) (kind string, blocks []contentBlock, ok bool) {
	if !bytes.Contains(line, []byte(`"type":"user"`)) && !bytes.Contains(line, []byte(`"type":"assistant"`)) {
		return "", nil, false
	}
	var entry struct {
		Type        string `json:"type"`
		IsSidechain bool   `json:"isSidechain"`
		Message     struct {
			Content json.RawMessage `json:"content"`
		} `json:"message"`
	}
	if json.Unmarshal(line, &entry) != nil || entry.IsSidechain || (entry.Type != "user" && entry.Type != "assistant") {
		return "", nil, false
	}
	var text string
	if json.Unmarshal(entry.Message.Content, &text) == nil {
		return entry.Type, []contentBlock{{Type: "text", Text: text}}, true
	}
	if json.Unmarshal(entry.Message.Content, &blocks) != nil {
		return "", nil, false
	}
	return entry.Type, blocks, true
}

func parseTranscriptTail(data []byte) transcriptInfo {
	var info transcriptInfo
	lines := bytes.Split(data, []byte("\n"))
	info.Asks = lastTurnAsks(lines)
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

// lastTurnAsks says whether the last turn ended with a question for the user.
// In the desktop app the agent closes a turn with a wrap_up tool whose "ask"
// holds the question, so that decides it. In the terminal there is no such
// tool, so it falls back to whether the last reply text ends with "?".
func lastTurnAsks(lines [][]byte) bool {
	lastText := ""
	for i := len(lines) - 1; i >= 0; i-- {
		kind, blocks, ok := parseMessage(lines[i])
		if !ok {
			continue
		}
		if kind == "user" {
			// Tool results come back as user messages; anything else is a prompt.
			if len(blocks) == 0 || blocks[0].Type != "tool_result" {
				break // the prompt that started the turn
			}
			continue
		}
		for j := len(blocks) - 1; j >= 0; j-- {
			b := blocks[j]
			switch {
			case b.Type == "tool_use" && strings.HasSuffix(b.Name, "__wrap_up"):
				return b.Input.Ask != ""
			case b.Type == "text" && lastText == "":
				lastText = b.Text
			}
		}
	}
	return strings.HasSuffix(strings.TrimRight(lastText, " \t\r\n*_`\"”)"), "?")
}
