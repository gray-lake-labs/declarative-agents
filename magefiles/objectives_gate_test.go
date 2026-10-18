package main

import (
	"path/filepath"
	"testing"

	"github.com/gray-lake-labs/scrum-master/internal/objectives"
)

// TestObjectiveDocumentsAreValid gates every objective document in the
// repository (srd034 R2.1). The parent's corpus audit does not read
// docs/specs/objectives/ (SPECIFICATIONS GAP8), so mage audit checks it here.
func TestObjectiveDocumentsAreValid(t *testing.T) {
	docs, err := objectives.LoadDir(filepath.Join("..", objectives.Dir))
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range docs {
		for _, f := range objectives.Validate(d) {
			t.Error(f)
		}
	}
}

// TestObjectiveGateRejectsMalformedFixtures proves the gate fails on each
// malformed fixture and passes the valid one.
func TestObjectiveGateRejectsMalformedFixtures(t *testing.T) {
	valid, err := objectives.LoadDir(filepath.Join("..", "testdata", "objectives", "valid"))
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range valid {
		if f := objectives.Validate(d); len(f) != 0 {
			t.Errorf("valid fixture %s: %v", d.Path, f)
		}
	}
	malformed, err := objectives.LoadDir(filepath.Join("..", "testdata", "objectives", "malformed"))
	if err != nil {
		t.Fatal(err)
	}
	if len(malformed) == 0 {
		t.Fatal("no malformed fixtures")
	}
	for _, d := range malformed {
		if len(objectives.Validate(d)) == 0 {
			t.Errorf("malformed fixture %s passed", d.Path)
		}
	}
}
