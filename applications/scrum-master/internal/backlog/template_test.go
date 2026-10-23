// Copyright (c) 2026 Gray Lake Labs
// SPDX-License-Identifier: MIT-0

package backlog

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/hierarchy"
)

// Every level has a template, and laid out together they pass the
// hierarchy gate, so the layout a writer copies is the one the gate reads
// (GH-20).
func TestTemplatesPassTheGate(t *testing.T) {
	dir := t.TempDir()
	folders := map[string]string{"theme": "themes", "initiative": "initiatives", "epic": "epics", "feature": "features", "story": "stories"}
	for _, k := range kinds() {
		folder, ok := folders[k]
		if !ok {
			continue
		}
		doc, err := Template(k)
		if err != nil {
			t.Fatal(err)
		}
		name := map[string]string{"theme": "th001-dependable-agents", "initiative": "in001-reliable-agents", "epic": "ep001-agent-evaluation", "feature": "ft001-grader", "story": "st001-score-trajectories"}[k]
		path := filepath.Join(dir, "docs", "specs", folder, name+".md")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(doc), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	docs, err := hierarchy.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 5 {
		t.Fatalf("loaded %d documents, want 5", len(docs))
	}
	if f := hierarchy.Validate(docs); len(f) != 0 {
		t.Fatalf("the templates fail the gate: %+v", f)
	}
	if _, err := Template("no-such-kind"); err == nil {
		t.Fatal("an unknown kind must be refused")
	}
}
