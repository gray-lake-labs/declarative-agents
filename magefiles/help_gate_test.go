package main

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/gray-lake-labs/scrum-master/internal/cli"
	_ "github.com/gray-lake-labs/scrum-master/internal/commands"
	"github.com/gray-lake-labs/scrum-master/internal/help"
)

// The help pages are prose held to the binary and the corpus: every command
// a page names is in the dispatch table, every SRD it cites is in the
// corpus, every group has a page, and a group page lists exactly the
// group's subcommands (srd044 R3).

func helpPages(t *testing.T) []help.Page {
	t.Helper()
	pages, err := help.Pages()
	if err != nil {
		t.Fatal(err)
	}
	return pages
}

func TestHelpPagesNameRealCommandsAndSRDs(t *testing.T) {
	groups := cli.Default.Groups()
	for _, p := range helpPages(t) {
		for _, m := range help.Mentions(cli.Default, p.Text) {
			if !slices.Contains(groups, m.Group) {
				t.Errorf("help %s names tpm %s, which is no group", p.Name, m.Group)
				continue
			}
			if m.Word != "" && m.Sub == "" && !hasGroupCommand(m.Group) {
				t.Errorf("help %s names tpm %s %s, which is no subcommand", p.Name, m.Group, m.Word)
			}
		}
		for _, srd := range p.SRDs {
			matches, _ := filepath.Glob(filepath.Join("..", "docs", "specs", "software-requirements", srd+"-*.md"))
			if len(matches) == 0 {
				t.Errorf("help %s cites %s, which the corpus does not hold", p.Name, srd)
			}
		}
	}
}

func TestHelpPagesCoverEveryGroup(t *testing.T) {
	var names []string
	for _, p := range helpPages(t) {
		names = append(names, p.Name)
	}
	for _, g := range cli.Default.Groups() {
		if g != "help" && !slices.Contains(names, g) {
			t.Errorf("group %s has no help page", g)
		}
	}
}

func TestHelpPagesListGroupSubcommands(t *testing.T) {
	groups := cli.Default.Groups()
	for _, p := range helpPages(t) {
		if !slices.Contains(groups, p.Name) {
			continue
		}
		var listed, want []string
		for _, m := range help.Listed(p.Text) {
			if m.Group != p.Name {
				t.Errorf("help %s lists tpm %s", p.Name, m.Group)
			}
			listed = append(listed, m.Sub)
		}
		for _, c := range cli.Default.Commands(p.Name) {
			want = append(want, c.Name)
		}
		slices.Sort(listed)
		slices.Sort(want)
		if !slices.Equal(listed, want) {
			t.Errorf("help %s lists %q; the group has %q", p.Name, listed, want)
		}
	}
}

func hasGroupCommand(group string) bool {
	for _, c := range cli.Default.Commands(group) {
		if c.Name == "" {
			return true
		}
	}
	return false
}
