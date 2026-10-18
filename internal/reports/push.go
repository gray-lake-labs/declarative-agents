package reports

import (
	"fmt"
	"strings"

	"github.com/gray-lake-labs/scrum-master/internal/gitlab"
	"github.com/gray-lake-labs/scrum-master/internal/program"
)

// PushResult is one actionable signal and the issue that carries it.
type PushResult struct {
	Key    string `json:"key"`
	Action string `json:"action"` // filed, exists
	IID    int    `json:"iid,omitempty"`
	WebURL string `json:"web_url,omitempty"`
}

func signalMarker(s Signal) string {
	return fmt.Sprintf("<!-- tpm reports signal=%s -->", s.Key())
}

// SignalIssue is the issue an actionable signal becomes, in do-work's input
// format (srd018 R6.3).
func SignalIssue(s Signal) gitlab.NewIssue {
	body := fmt.Sprintf(`%s (%s, %s).

%s

## Requirements

1. Find the cause of the %s signal for %s and decide with the component's owner what changes.
2. Record the decision as a comment on this issue and update the affected objective or blocker issues.

## Acceptance Criteria

- The next tpm reports aggregate no longer reports %s for %s (test: tpm_reports_aggregate; environment: local)
`, s.Detail, s.Kind, s.Week, signalMarker(s), s.Kind, s.Subject, s.Kind, s.Subject)
	return gitlab.NewIssue{Title: fmt.Sprintf("Signal: %s on %s", s.Kind, s.Subject), Description: body,
		Labels: program.Labels("management", "", "", s.Component)}
}

// PushSignals files each actionable signal as an issue in the system
// repository, once: an open issue already carrying the signal is kept
// (srd018 R3.6, R6.3).
func PushSignals(t gitlab.Tracker, system string, signals []Signal, p program.Policy, dryRun bool) ([]PushResult, error) {
	open, err := t.ListIssues(system, gitlab.IssueQuery{State: "opened"})
	if err != nil {
		return nil, err
	}
	var out []PushResult
	for _, s := range signals {
		if !s.Actionable {
			continue
		}
		r := PushResult{Key: s.Key(), Action: "filed"}
		for _, i := range open {
			if strings.Contains(i.Description, signalMarker(s)) {
				r.Action, r.IID, r.WebURL = "exists", i.IID, i.WebURL
				break
			}
		}
		if r.Action == "filed" {
			fr, err := program.File(t, system, SignalIssue(s), p, dryRun)
			if err != nil {
				return out, err
			}
			r.IID, r.WebURL = fr.IID, fr.WebURL
		}
		out = append(out, r)
	}
	return out, nil
}
