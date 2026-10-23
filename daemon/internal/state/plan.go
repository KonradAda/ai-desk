package state

import (
	"encoding/json"
	"strings"
	"time"
)

// Plan tracking. Agents keep a task list while working on multi-step tasks:
// Claude Code in the terminal uses TodoWrite (always the full list), the Claude
// desktop app uses an update_plan tool that sends the full list once and then
// short updates ("advance", "done", "start"). Both end up as planItems.

type planItem struct {
	Content string
	Status  string // pending, in_progress, completed
}

// planTool reports whether a tool call updates the agent's task list.
func planTool(name string) bool {
	return name == "TodoWrite" || strings.HasSuffix(name, "__update_plan")
}

// applyPlan updates the session's plan from a plan tool call.
func (sess *Session) applyPlan(tool string, raw json.RawMessage, now time.Time) {
	if tool == "TodoWrite" {
		var in struct {
			Todos []struct {
				Content    string `json:"content"`
				ActiveForm string `json:"activeForm"`
				Status     string `json:"status"`
			} `json:"todos"`
		}
		if json.Unmarshal(raw, &in) != nil {
			return
		}
		items := make([]planItem, 0, len(in.Todos))
		for _, t := range in.Todos {
			label := t.Content
			if t.Status == "in_progress" && t.ActiveForm != "" {
				label = t.ActiveForm // "Running tests" reads better than "Run tests" while it happens
			}
			items = append(items, planItem{Content: label, Status: t.Status})
		}
		sess.plan = items
	} else {
		var in struct {
			Items []struct {
				Content string `json:"content"`
				Status  string `json:"status"`
			} `json:"items"`
			Advance bool  `json:"advance"`
			Done    []int `json:"done"`  // 1-based
			Start   int   `json:"start"` // 1-based
		}
		if json.Unmarshal(raw, &in) != nil {
			return
		}
		if in.Items != nil {
			sess.plan = make([]planItem, 0, len(in.Items))
			for _, it := range in.Items {
				sess.plan = append(sess.plan, planItem{Content: it.Content, Status: it.Status})
			}
		}
		if in.Advance {
			sess.advancePlan()
		}
		for _, n := range in.Done {
			if n >= 1 && n <= len(sess.plan) {
				sess.plan[n-1].Status = "completed"
			}
		}
		if in.Start >= 1 && in.Start <= len(sess.plan) {
			sess.plan[in.Start-1].Status = "in_progress"
		}
	}
	sess.updateSteps(now)
}

// advancePlan completes the step in progress and starts the next pending one.
func (sess *Session) advancePlan() {
	from := 0
	for i, it := range sess.plan {
		if it.Status == "in_progress" {
			sess.plan[i].Status = "completed"
			from = i + 1
			break
		}
	}
	for i := from; i < len(sess.plan); i++ {
		if sess.plan[i].Status == "pending" {
			sess.plan[i].Status = "in_progress"
			return
		}
	}
}

// updateSteps recomputes the summary sent to the display. Since only moves
// when the agent reaches a new step, so the display can ease the bar forward
// within a step based on how long it has been running.
func (sess *Session) updateSteps(now time.Time) {
	if len(sess.plan) == 0 {
		sess.Steps = nil
		return
	}
	st := Steps{Total: len(sess.plan)}
	for _, it := range sess.plan {
		switch it.Status {
		case "completed":
			st.Done++
		case "in_progress":
			if st.Current == "" {
				st.Current = it.Content
			}
		}
	}
	st.Since = now
	if prev := sess.Steps; prev != nil && prev.Done == st.Done && prev.Total == st.Total && prev.Current == st.Current {
		st.Since = prev.Since
	}
	sess.Steps = &st
}

// planFinished reports whether every step of the plan is completed, so a new
// prompt starts without the old plan.
func (sess *Session) planFinished() bool {
	return sess.Steps != nil && sess.Steps.Done == sess.Steps.Total
}
