package main

import (
	"bytes"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/gray-lake-labs/scrum-master/internal/cli"
	"github.com/gray-lake-labs/scrum-master/internal/tracker"
)

func TestIssueGroupServesSubcommands(t *testing.T) {
	var names []string
	for _, c := range cli.Default.Commands("issue") {
		names = append(names, c.Name)
	}
	for _, want := range []string{"approve", "close", "file", "labels", "lint", "merge", "pop", "recur"} {
		if !slices.Contains(names, want) {
			t.Errorf("tpm issue has no %s; it has %v", want, names)
		}
	}
	if groups := cli.Default.Groups(); slices.Contains(groups, "program") {
		t.Errorf("program is still a group: %v", groups)
	}
}

func TestProgramGroupRefusesWithPointer(t *testing.T) {
	opened := false
	tracker := func() (tracker.Tracker, error) { opened = true; return nil, errors.New("no tracker in this test") }
	for _, args := range [][]string{{"program"}, {"program", "pop", "--project", "g/po-1", "--issue", "1"}} {
		var out bytes.Buffer
		err := cli.Default.Run(args, &out, tracker)
		if err == nil || errors.Is(err, cli.ErrUsage) {
			t.Fatalf("tpm %s: want a non-usage error, got %v", strings.Join(args, " "), err)
		}
		for _, want := range []string{"tpm issue", "tpm install"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("tpm %s: %q names no %q", strings.Join(args, " "), err, want)
			}
		}
		if len(args) > 1 && !strings.Contains(err.Error(), "tpm issue pop") {
			t.Errorf("tpm %s: %q does not name the subcommand to run", strings.Join(args, " "), err)
		}
	}
	if opened {
		t.Error("tpm program opened the tracker; it must run nothing")
	}
}
