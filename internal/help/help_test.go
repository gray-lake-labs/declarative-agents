package help_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/gray-lake-labs/scrum-master/internal/cli"
	_ "github.com/gray-lake-labs/scrum-master/internal/commands"
	"github.com/gray-lake-labs/scrum-master/internal/help"
	"github.com/gray-lake-labs/scrum-master/internal/tracker"
)

func noTracker() (tracker.Tracker, error) { return nil, errors.New("help reads no tracker") }

func tpm(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := cli.Default.Run(args, &out, noTracker)
	return out.String(), err
}

func TestHelpProcessPage(t *testing.T) {
	out, err := tpm(t, "help")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Three planes", "Management", "Control", "Data", "docs/VISION.md", "control/policy.md",
		"tpm plan", "tpm goals", "tpm issue pop", "tpm release", "tpm sprint-end", "track-goals"} {
		if !strings.Contains(out, want) {
			t.Errorf("the process page lacks %q", want)
		}
	}
	for step := 1; step <= 10; step++ {
		if !strings.Contains(out, fmt.Sprintf("\n%4d ", step)) {
			t.Errorf("the process page lacks release-cycle step %d", step)
		}
	}
}

func TestHelpGroupPages(t *testing.T) {
	for _, g := range cli.Default.Groups() {
		if g == "help" {
			continue
		}
		out, err := tpm(t, "help", g)
		if err != nil || !strings.HasPrefix(out, g+": ") {
			t.Errorf("tpm help %s: %v\n%s", g, err, out)
		}
	}
	// The backlog page says where the document formats are (#166).
	if out, _ := tpm(t, "help", "backlog"); !strings.Contains(out, "docs/constitutions/design.md") {
		t.Errorf("tpm help backlog names no format document:\n%s", out)
	}
}

func TestHelpConceptPages(t *testing.T) {
	pages, err := help.Pages()
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]help.Page{}
	for _, p := range pages {
		byName[p.Name] = p
	}
	for _, c := range []string{"sprint-goal", "backlog", "objective", "admission", "approval", "milestone", "tracker", "evidence", "release", "recurring"} {
		p, ok := byName[c]
		if !ok {
			t.Errorf("no concept page %s", c)
			continue
		}
		if len(help.Mentions(cli.Default, p.Text)) == 0 || len(p.SRDs) == 0 {
			t.Errorf("concept page %s names no command or no SRD", c)
		}
	}
}

func TestHelpListAndUnknown(t *testing.T) {
	out, err := tpm(t, "help", "--list")
	if err != nil {
		t.Fatal(err)
	}
	pages, _ := help.Pages()
	for _, p := range pages {
		if !strings.Contains(out, p.Name+" ") {
			t.Errorf("--list lacks %s", p.Name)
		}
	}
	out, err = tpm(t, "help", "no-such-page")
	if err == nil || !errors.Is(err, cli.ErrUsage) || !strings.Contains(err.Error(), "objective") {
		t.Fatalf("unknown page: %v", err)
	}
}

func TestHelpJSON(t *testing.T) {
	out, err := tpm(t, "help", "issue", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var p help.Page
	if err := json.Unmarshal([]byte(out), &p); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if p.Name != "issue" || p.Summary == "" || !strings.Contains(p.Text, "Subcommands:") || len(p.Commands) == 0 || len(p.SRDs) == 0 {
		t.Fatalf("issue page as JSON: %+v", p)
	}
	for _, c := range p.Commands {
		if !strings.HasPrefix(c, "tpm ") {
			t.Errorf("command %q", c)
		}
	}
}

func TestHelpPageLength(t *testing.T) {
	pages, err := help.Pages()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range pages {
		if n := strings.Count(strings.TrimRight(p.Text, "\n"), "\n") + 1; n > help.MaxLines {
			t.Errorf("%s is %d lines, more than %d", p.Name, n, help.MaxLines)
		}
	}
}
