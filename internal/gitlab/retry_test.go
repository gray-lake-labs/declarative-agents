package gitlab

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// stalling answers the first request of each method only after the
// client's timeout, and the rest at once.
func stalling(t *testing.T) (*Client, *atomic.Int32, *atomic.Int32) {
	t.Helper()
	var gets, posts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := &gets
		if r.Method == http.MethodPost {
			n = &posts
		}
		if n.Add(1) == 1 {
			time.Sleep(300 * time.Millisecond)
		}
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/issues") {
			w.Write([]byte(`{"id": 11, "iid": 1, "title": "x"}`))
			return
		}
		w.Write([]byte(`{"id": 11, "iid": 1, "title": "x", "state": "opened"}`))
	}))
	t.Cleanup(srv.Close)
	c, err := newClient("token", srv.URL, 100*time.Millisecond, time.Millisecond, 5*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	return c, &gets, &posts
}

func TestTimedOutGetIsRetried(t *testing.T) {
	c, gets, _ := stalling(t)
	i, err := c.GetIssue("g/p", 1)
	if err != nil || i.IID != 1 {
		t.Fatalf("issue %+v err %v", i, err)
	}
	if gets.Load() != 2 {
		t.Fatalf("GET attempts = %d, want 2", gets.Load())
	}
}

func TestTimedOutWriteIsNotRetried(t *testing.T) {
	c, _, posts := stalling(t)
	if _, err := c.CreateIssue("g/p", NewIssue{Title: "x"}); err == nil {
		t.Fatal("a write that timed out reported success")
	}
	if posts.Load() != 1 {
		t.Fatalf("POST attempts = %d, want 1", posts.Load())
	}
}
