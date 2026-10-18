package gitlab

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gl "gitlab.com/gitlab-org/api/client-go"
)

// fakeGroupMilestones serves the group milestone endpoints EnsureMilestone
// calls, holding at most one milestone, and records each request method.
func fakeGroupMilestones(t *testing.T, existing map[string]any) (*Client, *[]string) {
	t.Helper()
	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/groups/") || !strings.HasSuffix(strings.TrimSuffix(r.URL.Path, "/7"), "/milestones") {
			http.NotFound(w, r)
			return
		}
		calls = append(calls, r.Method)
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			if existing == nil {
				w.Write([]byte("[]"))
				return
			}
			json.NewEncoder(w).Encode([]any{existing})
		default:
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			body["id"] = 7
			existing = body
			json.NewEncoder(w).Encode(body)
		}
	}))
	t.Cleanup(srv.Close)
	api, err := gl.NewClient("token", gl.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	return &Client{api: api}, &calls
}

func TestEnsureMilestoneUpdatesRatherThanDuplicates(t *testing.T) {
	m := Milestone{Title: "01.2", Description: "2026-W43..W45", StartDate: "2026-10-19", DueDate: "2026-11-06"}

	c, calls := fakeGroupMilestones(t, nil)
	if got, err := c.EnsureMilestone("program", m); err != nil || got != MilestoneCreated {
		t.Fatalf("absent: %s %v", got, err)
	}
	if got, err := c.EnsureMilestone("program", m); err != nil || got != MilestoneUnchanged {
		t.Fatalf("second run: %s %v", got, err)
	}
	m.DueDate = "2026-11-13"
	if got, err := c.EnsureMilestone("program", m); err != nil || got != MilestoneUpdated {
		t.Fatalf("moved: %s %v", got, err)
	}
	if strings.Join(*calls, ",") != "GET,POST,GET,GET,PUT" {
		t.Fatalf("calls = %v", *calls)
	}
	if _, err := c.EnsureMilestone("program", Milestone{Title: "x", DueDate: "next week"}); err == nil {
		t.Fatal("a date that is not YYYY-MM-DD was sent")
	}
}
