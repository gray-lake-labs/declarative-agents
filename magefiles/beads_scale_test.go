package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/gray-lake-labs/scrum-master/internal/beads"
)

// TestScaleSeedIsATrackerFile writes the seed the way the store keeps its
// file: one issue per line, sorted by ID, every ID and IID once, the
// reserved issues ready to pop.
func TestScaleSeedIsATrackerFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), beads.IssuesFile)
	size, err := writeScaleIssues(path, 500, 40)
	if err != nil || size == 0 {
		t.Fatalf("seed: %d bytes, %v", size, err)
	}
	b, _ := os.ReadFile(path)
	var ids []string
	iids := map[int]bool{}
	for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		var i beads.Issue
		if err := json.Unmarshal([]byte(line), &i); err != nil {
			t.Fatal(err)
		}
		if iids[i.IID] {
			t.Fatalf("IID %d twice", i.IID)
		}
		iids[i.IID] = true
		ids = append(ids, i.ID)
		if i.IID <= 40 && i.State != beads.Ready {
			t.Errorf("reserved issue %d is %s, not ready", i.IID, i.State)
		}
		if len(i.Comments) != 5 {
			t.Errorf("issue %d has %d comments", i.IID, len(i.Comments))
		}
	}
	if len(ids) != 500 || !slices.IsSorted(ids) || len(slices.Compact(slices.Clone(ids))) != 500 {
		t.Fatalf("%d issues, sorted %v, unique %d", len(ids), slices.IsSorted(ids), len(slices.Compact(slices.Clone(ids))))
	}
}

func TestScaleSizesReadsTheEnvironment(t *testing.T) {
	t.Setenv("BEADS_SCALE_TEST", "")
	if got, err := scaleSizes("BEADS_SCALE_TEST", "1,2"); err != nil || !slices.Equal(got, []int{1, 2}) {
		t.Fatalf("default: %v, %v", got, err)
	}
	t.Setenv("BEADS_SCALE_TEST", "100, 300")
	if got, err := scaleSizes("BEADS_SCALE_TEST", "1"); err != nil || !slices.Equal(got, []int{100, 300}) {
		t.Fatalf("set: %v, %v", got, err)
	}
	t.Setenv("BEADS_SCALE_TEST", "x")
	if _, err := scaleSizes("BEADS_SCALE_TEST", "1"); err == nil {
		t.Fatal("a non-number was accepted")
	}
}
