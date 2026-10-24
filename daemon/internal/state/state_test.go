package state

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newTestStore() (*Store, *time.Time) {
	s := NewStore(nil)
	s.findHost = func(string) string { return "" } // keep tests off the real ~/.claude
	now := time.Date(2026, 10, 23, 14, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	return s, &now
}

func TestSessionLifecycle(t *testing.T) {
	s, now := newTestStore()
	cwd := `C:\Projects\api-backend`

	s.Apply(Event{SessionID: "a", Cwd: cwd, HookEventName: "SessionStart"})
	s.Apply(Event{SessionID: "a", Cwd: cwd, HookEventName: "UserPromptSubmit"})
	s.Apply(Event{SessionID: "a", Cwd: cwd, HookEventName: "PreToolUse", ToolName: "Edit",
		ToolInput: json.RawMessage(`{"file_path":"C:\\Projects\\api-backend\\auth.ts"}`)})

	got := snapshot(s)
	if len(got) != 1 {
		t.Fatalf("want 1 session, got %d", len(got))
	}
	if got[0].Project != "api-backend" || got[0].Status != StatusWorking || got[0].Action != (Action{Kind: ActEdit, Arg: "auth.ts"}) {
		t.Fatalf("unexpected session: %+v", got[0])
	}

	s.Apply(Event{SessionID: "a", HookEventName: "Notification", Message: "Claude needs your permission to use Bash"})
	if got := snapshot(s)[0]; got.Status != StatusWaiting || got.Action != (Action{Kind: ActPermission, Arg: "Bash"}) {
		t.Fatalf("unexpected waiting session: %+v", got)
	}

	s.Apply(Event{SessionID: "a", HookEventName: "PostToolUse"})
	if st := snapshot(s)[0].Status; st != StatusWorking {
		t.Fatalf("want working after tool ran, got %s", st)
	}

	*now = now.Add(5 * time.Minute)
	s.Apply(Event{SessionID: "a", HookEventName: "Stop"})
	if got := snapshot(s)[0]; got.Status != StatusDone || !got.FinishedAt.Equal(*now) {
		t.Fatalf("unexpected done session: %+v", got)
	}

	s.Apply(Event{SessionID: "a", HookEventName: "SessionEnd"})
	if n := len(snapshot(s)); n != 0 {
		t.Fatalf("want session removed, got %d", n)
	}
}

func TestTodoSteps(t *testing.T) {
	s, _ := newTestStore()
	s.Apply(Event{SessionID: "a", Cwd: "/p", HookEventName: "PreToolUse", ToolName: "TodoWrite",
		ToolInput: json.RawMessage(`{"todos":[
			{"content":"Plan","status":"completed"},
			{"content":"Code","status":"completed"},
			{"content":"Run tests","activeForm":"Running tests","status":"in_progress"},
			{"content":"Commit","status":"pending"}]}`)})
	st := snapshot(s)[0].Steps
	if st == nil || st.Done != 2 || st.Total != 4 || st.Current != "Running tests" {
		t.Fatalf("want 2/4 steps running tests, got %+v", st)
	}
}

// snapshot returns all sessions, without hiding finished ones.
func snapshot(s *Store) []Session { return s.Snapshot(0) }

func slotOf(s *Store, id string) int {
	for _, sess := range snapshot(s) {
		if sess.ID == id {
			return sess.Slot
		}
	}
	return -1
}

func start(s *Store, id, cwd string) {
	s.Apply(Event{SessionID: id, Cwd: cwd, HookEventName: "SessionStart"})
}

func TestSlotsStayFixedWhileAgentsWork(t *testing.T) {
	s, now := newTestStore()
	start(s, "a", "/pa")
	start(s, "b", "/pb")
	start(s, "c", "/pc")
	before := map[string]int{"a": slotOf(s, "a"), "b": slotOf(s, "b"), "c": slotOf(s, "c")}

	// Activity in any order must not reshuffle the cards.
	for _, id := range []string{"c", "a", "b", "c"} {
		*now = now.Add(time.Minute)
		s.Apply(Event{SessionID: id, HookEventName: "PreToolUse", ToolName: "Read"})
	}
	s.Apply(Event{SessionID: "b", HookEventName: "Notification"})
	for id, slot := range before {
		if got := slotOf(s, id); got != slot {
			t.Fatalf("session %s moved from slot %d to %d", id, slot, got)
		}
	}
}

func TestNoLimitOnSessions(t *testing.T) {
	s, _ := newTestStore()
	for i := range 8 {
		start(s, string(rune('a'+i)), "/p"+string(rune('a'+i)))
	}
	got := snapshot(s)
	if len(got) != 8 {
		t.Fatalf("want all 8 sessions, got %d", len(got))
	}
	for i := 1; i < len(got); i++ {
		if got[i].Slot <= got[i-1].Slot {
			t.Fatalf("sessions must be ordered by unique slots: %d then %d", got[i-1].Slot, got[i].Slot)
		}
	}
}

func TestHideDoneAfter(t *testing.T) {
	s, now := newTestStore()
	s.Apply(Event{SessionID: "done", Cwd: "/a", HookEventName: "Stop"})
	s.Apply(Event{SessionID: "busy", Cwd: "/b", HookEventName: "UserPromptSubmit"})
	*now = now.Add(10 * time.Minute)

	if n := len(s.Snapshot(15 * time.Minute)); n != 2 {
		t.Fatalf("finished 10 min ago should still show with a 15 min limit, got %d", n)
	}
	got := s.Snapshot(5 * time.Minute)
	if len(got) != 1 || got[0].ID != "busy" {
		t.Fatalf("finished session should be hidden after 5 min, got %+v", got)
	}

	// Starting work again brings it back.
	s.Apply(Event{SessionID: "done", HookEventName: "UserPromptSubmit"})
	if n := len(s.Snapshot(5 * time.Minute)); n != 2 {
		t.Fatalf("session working again should show, got %d", n)
	}
}

func TestMoveSwapsAndIsRemembered(t *testing.T) {
	s, _ := newTestStore()
	start(s, "a", "/alpha")
	start(s, "b", "/beta")
	slotA, slotB := slotOf(s, "a"), slotOf(s, "b")

	if !s.Move("a", slotB) {
		t.Fatal("move failed")
	}
	if slotOf(s, "a") != slotB || slotOf(s, "b") != slotA {
		t.Fatalf("cards should swap: a=%d b=%d", slotOf(s, "a"), slotOf(s, "b"))
	}
	if !s.Move("a", 5) || slotOf(s, "a") != 5 {
		t.Fatal("move to empty slot failed")
	}

	// A new session of the same project returns to its remembered slot.
	s.Apply(Event{SessionID: "a", HookEventName: "SessionEnd"})
	start(s, "a2", "/alpha")
	if got := slotOf(s, "a2"); got != 5 {
		t.Fatalf("want remembered slot 5, got %d", got)
	}

	if s.Move("nope", 1) || s.Move("a2", MaxSlots) {
		t.Fatal("invalid moves should be rejected")
	}
}

func TestNewProjectAvoidsClaimedSlots(t *testing.T) {
	s, _ := newTestStore()
	start(s, "a", "/alpha")
	s.Move("a", 0)
	s.Apply(Event{SessionID: "a", HookEventName: "SessionEnd"})

	// Slot 0 is free but belongs to alpha, so a new project goes elsewhere.
	start(s, "b", "/beta")
	if got := slotOf(s, "b"); got == 0 {
		t.Fatal("new project took a slot claimed by another project")
	}
}

func TestLayoutPersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "layout.json")
	s := NewStore(LoadLayout(path))
	start(s, "a", "/alpha")
	s.Move("a", 4)

	s2 := NewStore(LoadLayout(path))
	start(s2, "x", "/Alpha")
	if got := slotOf(s2, "x"); got != 4 {
		t.Fatalf("want slot 4 loaded from disk, got %d", got)
	}
}

func TestStaleSessionsDropped(t *testing.T) {
	s, now := newTestStore()
	s.Apply(Event{SessionID: "a", Cwd: "/p", HookEventName: "Stop"})
	*now = now.Add(staleAfter + time.Minute)
	if n := len(snapshot(s)); n != 0 {
		t.Fatalf("want stale session dropped, got %d", n)
	}
}

func TestMascotStablePerProject(t *testing.T) {
	if mascotFor("api-backend") != mascotFor("API-Backend") {
		t.Fatal("mascot should not depend on case")
	}
}

func TestMascotsUniqueOnScreen(t *testing.T) {
	s, _ := newTestStore()
	for i := range 6 {
		s.Apply(Event{SessionID: string(rune('a' + i)), Cwd: "/p" + string(rune('a'+i)), HookEventName: "SessionStart"})
	}
	seen := map[int]bool{}
	for _, sess := range snapshot(s) {
		if seen[sess.Mascot] {
			t.Fatalf("mascot %d used twice", sess.Mascot)
		}
		seen[sess.Mascot] = true
	}

	// A second session in the same project still gets its own mascot.
	s.Apply(Event{SessionID: "b", HookEventName: "SessionEnd"}) // free a slot so it is visible
	s.Apply(Event{SessionID: "z", Cwd: "/pa", HookEventName: "SessionStart"})
	var first, second int
	for _, sess := range snapshot(s) {
		switch sess.ID {
		case "a":
			first = sess.Mascot
		case "z":
			second = sess.Mascot
		}
	}
	if first == second {
		t.Fatalf("two sessions of one project should look different, both got mascot %d", first)
	}
}

func TestTitleFromPromptThenTranscript(t *testing.T) {
	s, now := newTestStore()
	title := ""
	reads := 0
	s.readInfo = func(string) transcriptInfo { reads++; return transcriptInfo{Title: title} }

	ev := Event{SessionID: "a", Cwd: "/p", TranscriptPath: "/t.jsonl", HookEventName: "UserPromptSubmit",
		Prompt: "Napraw logowanie w panelu admina, bo po zmianie hasła wywala błąd 500"}
	s.Apply(ev)
	if got := snapshot(s)[0].Title; got != "Napraw logowanie w panelu admina, bo po…" {
		t.Fatalf("want prompt fallback, got %q", got)
	}

	// The transcript is not re-read on every event.
	title = "Naprawa logowania"
	s.Apply(Event{SessionID: "a", TranscriptPath: "/t.jsonl", HookEventName: "PreToolUse"})
	if reads != 1 {
		t.Fatalf("want 1 transcript read, got %d", reads)
	}

	*now = now.Add(infoRefresh)
	s.Apply(Event{SessionID: "a", TranscriptPath: "/t.jsonl", HookEventName: "PreToolUse"})
	if got := snapshot(s)[0].Title; got != "Naprawa logowania" {
		t.Fatalf("want transcript title, got %q", got)
	}

	// A later prompt doesn't replace an existing title.
	*now = now.Add(time.Second)
	s.Apply(Event{SessionID: "a", HookEventName: "UserPromptSubmit", Prompt: "a teraz testy"})
	if got := snapshot(s)[0].Title; got != "Naprawa logowania" {
		t.Fatalf("title should stay, got %q", got)
	}
}

func TestParseTranscriptTail(t *testing.T) {
	data := []byte(`{"type":"user","message":"mentions \"custom-title\" and \"type\":\"assistant\" in text"}
{"type":"custom-title","customTitle":"Old title","sessionId":"x"}
{"type":"assistant","message":{"model":"claude-sonnet-5-5"},"effort":"low","entrypoint":"cli"}
{"type":"custom-title","customTitle":"New title","sessionId":"x"}
{"type":"assistant","message":{"model":"claude-opus-5-5"},"effort":"medium","perTurnEffort":"high","entrypoint":"claude-desktop"}
{"type":"assistant","isSidechain":true,"message":{"model":"claude-haiku-5-5"},"effort":"low"}
{"type":"user","message":"another \"custom-title\" mention"}
{"type":"custom-ti`) // the tail may cut a line in half
	got := parseTranscriptTail(data)
	want := transcriptInfo{Title: "New title", Model: "claude-opus-5-5", Effort: "high", Entrypoint: "claude-desktop"}
	if got != want {
		t.Fatalf("want %+v, got %+v", want, got)
	}
	if got := parseTranscriptTail([]byte(`{"type":"user"}`)); got != (transcriptInfo{}) {
		t.Fatalf("want empty info, got %+v", got)
	}
}

func TestModelAndEffortFromTranscript(t *testing.T) {
	s, now := newTestStore()
	info := transcriptInfo{Model: "claude-opus-5-5", Effort: "medium", Entrypoint: "claude-desktop"}
	s.readInfo = func(string) transcriptInfo { return info }

	s.Apply(Event{SessionID: "a", Cwd: "/p", TranscriptPath: "/t.jsonl", HookEventName: "UserPromptSubmit"})
	got := snapshot(s)[0]
	if got.Agent != AgentClaudeCode || got.Model != "claude-opus-5-5" || got.Effort != "medium" || got.Client != "claude-desktop" {
		t.Fatalf("unexpected session info: %+v", got)
	}

	// Effort changed mid-session: Stop re-reads at once, without waiting for the refresh interval.
	info.Effort = "high"
	*now = now.Add(time.Second)
	s.Apply(Event{SessionID: "a", TranscriptPath: "/t.jsonl", HookEventName: "Stop"})
	if got := snapshot(s)[0].Effort; got != "high" {
		t.Fatalf("want effort updated on Stop, got %q", got)
	}
}

func TestProjectNameUsesGitRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "my-repo")
	sub := filepath.Join(root, "daemon", "internal")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := projectName(sub); got != "my-repo" {
		t.Fatalf("want my-repo, got %s", got)
	}
	if got := projectName(filepath.Join(t.TempDir(), "plain")); got != "plain" {
		t.Fatalf("want plain, got %s", got)
	}
}

func planEvent(id, input string) Event {
	return Event{SessionID: id, Cwd: "/p", HookEventName: "PreToolUse", ToolName: "mcp__ccd_turn__update_plan",
		ToolInput: json.RawMessage(input)}
}

func TestDesktopPlanUpdates(t *testing.T) {
	s, now := newTestStore()
	s.Apply(planEvent("a", `{"items":[
		{"content":"Write code","status":"in_progress"},
		{"content":"Test it","status":"pending"},
		{"content":"Verify in browser","status":"pending"},
		{"content":"Commit","status":"pending"}]}`))
	st := snapshot(s)[0].Steps
	if st == nil || st.Done != 0 || st.Total != 4 || st.Current != "Write code" {
		t.Fatalf("unexpected steps after full list: %+v", st)
	}
	if a := snapshot(s)[0].Action; a.Kind != ActPlan {
		t.Fatalf("plan tool should show as planning, got %+v", a)
	}
	started := st.Since

	*now = now.Add(time.Minute)
	s.Apply(planEvent("a", `{"advance":true}`))
	st = snapshot(s)[0].Steps
	if st.Done != 1 || st.Current != "Test it" || !st.Since.After(started) {
		t.Fatalf("advance should complete step 1 and start step 2: %+v", st)
	}

	// Other tool calls inside the same step keep its start time.
	since := st.Since
	*now = now.Add(time.Minute)
	s.Apply(Event{SessionID: "a", HookEventName: "PreToolUse", ToolName: "Read"})
	if got := snapshot(s)[0].Steps.Since; !got.Equal(since) {
		t.Fatal("step start time should not move without plan progress")
	}

	s.Apply(planEvent("a", `{"done":[2,3],"start":4}`))
	st = snapshot(s)[0].Steps
	if st.Done != 3 || st.Current != "Commit" {
		t.Fatalf("done/start should jump to step 4: %+v", st)
	}

	s.Apply(planEvent("a", `{"advance":true}`))
	if st = snapshot(s)[0].Steps; st.Done != 4 || st.Current != "" {
		t.Fatalf("last advance should finish the plan: %+v", st)
	}

	// The next task starts without the finished plan.
	s.Apply(Event{SessionID: "a", HookEventName: "UserPromptSubmit"})
	if st = snapshot(s)[0].Steps; st != nil {
		t.Fatalf("finished plan should be cleared on a new prompt: %+v", st)
	}
}

func TestNewPromptClearsUnfinishedPlan(t *testing.T) {
	// The desktop app checks off the last step itself when the turn ends,
	// so the daemon still sees it in progress when the next prompt arrives.
	s, _ := newTestStore()
	s.Apply(planEvent("a", `{"items":[{"content":"A","status":"completed"},{"content":"B","status":"in_progress"}]}`))
	s.Apply(Event{SessionID: "a", HookEventName: "Stop"})
	s.Apply(Event{SessionID: "a", HookEventName: "UserPromptSubmit"})
	if st := snapshot(s)[0].Steps; st != nil {
		t.Fatalf("new prompt should start without the old plan: %+v", st)
	}
}

func TestLastTurnAsks(t *testing.T) {
	const prompt = `{"type":"user","message":{"role":"user","content":"fix the bug"}}` + "\n"
	const toolResult = `{"type":"user","message":{"role":"user","content":[{"type":"tool_result","content":"ok"}]}}` + "\n"
	text := func(s string) string {
		return `{"type":"assistant","message":{"content":[{"type":"text","text":` + strconvQuote(s) + `}]}}` + "\n"
	}
	wrapUp := func(ask string) string {
		return `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"mcp__ccd_turn__wrap_up","input":{"recap":"r","ask":` + strconvQuote(ask) + `}}]}}` + "\n"
	}
	cases := []struct {
		name, data string
		want       bool
	}{
		{"terminal question", prompt + text("Done. Should I commit it?"), true},
		{"terminal question in bold", prompt + text("**Commit it?**\n"), true},
		{"terminal statement", prompt + text("Is it fixed? Yes, all tests pass."), false},
		{"desktop ask, closing text after it", prompt + wrapUp("Commit it?") + toolResult + text("Fixed the bug."), true},
		{"desktop no ask beats a question mark", prompt + wrapUp("") + toolResult + text("Want more?"), false},
		{"question from an earlier turn", text("Should I fix it?") + prompt + text("Fixed."), false},
		{"wrap_up from an earlier turn", wrapUp("Fix it?") + toolResult + prompt + text("Fixed."), false},
		{"subagent text ignored", prompt + text("Commit it?") + `{"type":"assistant","isSidechain":true,"message":{"content":[{"type":"text","text":"Found it."}]}}` + "\n", true},
	}
	for _, c := range cases {
		if got := parseTranscriptTail([]byte(c.data)).Asks; got != c.want {
			t.Errorf("%s: want asks=%v, got %v", c.name, c.want, got)
		}
	}
}

func strconvQuote(s string) string { b, _ := json.Marshal(s); return string(b) }

func TestWaitingForAnswer(t *testing.T) {
	s, _ := newTestStore()
	info := transcriptInfo{Asks: true}
	s.readInfo = func(string) transcriptInfo { return info }

	// A turn that ends with a question waits for the user instead of being done.
	s.Apply(Event{SessionID: "a", Cwd: "/p", TranscriptPath: "/t.jsonl", HookEventName: "Stop"})
	if got := snapshot(s)[0]; got.Status != StatusWaiting || got.Action.Kind != ActQuestion {
		t.Fatalf("want waiting on a question, got %+v", got)
	}

	info.Asks = false
	s.Apply(Event{SessionID: "a", TranscriptPath: "/t.jsonl", HookEventName: "Stop"})
	if got := snapshot(s)[0]; got.Status != StatusDone {
		t.Fatalf("want done without a question, got %+v", got)
	}
	// The idle reminder a minute later doesn't turn a finished turn into a waiting one.
	s.Apply(Event{SessionID: "a", HookEventName: "Notification", Message: "Claude is waiting for your input"})
	if got := snapshot(s)[0]; got.Status != StatusDone {
		t.Fatalf("want still done after the idle reminder, got %+v", got)
	}

	// Tools that ask the user wait until they are answered.
	for tool, kind := range map[string]string{"AskUserQuestion": ActQuestion, "ExitPlanMode": ActPlanApproval} {
		s.Apply(Event{SessionID: "a", HookEventName: "PreToolUse", ToolName: tool})
		if got := snapshot(s)[0]; got.Status != StatusWaiting || got.Action.Kind != kind {
			t.Fatalf("%s: want waiting, got %+v", tool, got)
		}
		s.Apply(Event{SessionID: "a", HookEventName: "PostToolUse", ToolName: tool})
		if got := snapshot(s)[0]; got.Status != StatusWorking {
			t.Fatalf("%s: want working once answered, got %+v", tool, got)
		}
	}
}

func TestLastEntryInterrupted(t *testing.T) {
	const prompt = `{"type":"user","message":{"role":"user","content":"fix the bug"}}` + "\n"
	const toolUse = `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash","input":{}}]}}` + "\n"
	const toolResult = `{"type":"user","message":{"role":"user","content":[{"type":"tool_result","content":"ok"}]}}` + "\n"
	const stopped = `{"type":"user","message":{"role":"user","content":[{"type":"text","text":"[Request interrupted by user]"}]}}` + "\n"
	const stoppedTool = `{"type":"user","message":{"role":"user","content":[{"type":"text","text":"[Request interrupted by user for tool use]"}]}}` + "\n"
	const other = `{"type":"file-history-snapshot","snapshot":{}}` + "\n"
	cases := []struct {
		name, data string
		want       bool
	}{
		{"stopped while thinking", prompt + stopped + other, true},
		{"stopped during a tool", prompt + toolUse + toolResult + stoppedTool, true},
		{"tool still running", prompt + toolUse, false},
		{"tool just finished", prompt + toolUse + toolResult, false},
		{"new prompt after a stop", prompt + stopped + prompt, false},
	}
	for _, c := range cases {
		if got := lastEntryInterrupted(bytes.Split([]byte(c.data), []byte("\n"))); got != c.want {
			t.Errorf("%s: want %v, got %v", c.name, c.want, got)
		}
	}
}

func TestInterruptedWhenQuiet(t *testing.T) {
	s, now := newTestStore()
	stopped := true
	s.interrupted = func(string) bool { return stopped }

	s.Apply(Event{SessionID: "a", Cwd: "/p", TranscriptPath: "/t.jsonl", HookEventName: "UserPromptSubmit"})
	s.CheckInterrupts()
	if got := snapshot(s)[0]; got.Status != StatusWorking {
		t.Fatalf("want working while events still arrive, got %+v", got)
	}

	*now = now.Add(interruptQuiet)
	s.CheckInterrupts()
	if got := snapshot(s)[0]; got.Status != StatusInterrupted || got.Action.Kind != ActInterrupted || !got.FinishedAt.Equal(*now) {
		t.Fatalf("want interrupted once quiet, got %+v", got)
	}
	s.Apply(Event{SessionID: "a", HookEventName: "Notification", Message: "Claude is waiting for your input"})
	if got := snapshot(s)[0]; got.Status != StatusInterrupted {
		t.Fatalf("want still interrupted after the idle reminder, got %+v", got)
	}

	// A new prompt starts work again; a long-running tool is quiet but not stopped.
	s.Apply(Event{SessionID: "a", HookEventName: "UserPromptSubmit"})
	stopped = false
	*now = now.Add(time.Minute)
	s.CheckInterrupts()
	if got := snapshot(s)[0]; got.Status != StatusWorking {
		t.Fatalf("want working during a long tool, got %+v", got)
	}
}
