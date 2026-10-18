package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/gray-lake-labs/scrum-master/internal/corpus"
	"github.com/gray-lake-labs/scrum-master/internal/release"
)

// notMarkdown are the files design.md D2 keeps out of Markdown, each because
// a consumer outside this repository requires its format.
var notMarkdown = []string{
	"docs/external-pins.yaml",      // the parent's audit reads it by path
	"docs/external-checkouts.yaml", // the parent's audit reads it by path
	"docs/corpus-charter.yaml",     // the parent's audit executes it
	"plugin/plugin.json",           // the Agent Plugins 1.0 standard
}

// yamlOutsideAllowed returns the YAML and JSON files among paths that D2
// does not allow; a GitLab CI file is allowed wherever it is, since GitLab
// reads only YAML.
func yamlOutsideAllowed(paths []string) []string {
	var out []string
	for _, p := range paths {
		switch {
		case !strings.HasSuffix(p, ".yaml") && !strings.HasSuffix(p, ".yml") && !strings.HasSuffix(p, ".json"):
		case slices.Contains(notMarkdown, p), strings.HasSuffix(p, ".gitlab-ci.yml"):
		default:
			out = append(out, p)
		}
	}
	return out
}

// TestNoYAMLOutsideTheAllowedFiles holds the repository to design.md D2
// (srd040 R2.1):
// every tracked file is Markdown with tags except the files notMarkdown and
// the GitLab CI files name.
func TestNoYAMLOutsideTheAllowedFiles(t *testing.T) {
	out, err := exec.Command("git", "-C", "..", "ls-files").Output()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range yamlOutsideAllowed(strings.Split(strings.TrimSpace(string(out)), "\n")) {
		t.Errorf("%s is not Markdown; design.md D2 allows only %s and *.gitlab-ci.yml", p, strings.Join(notMarkdown, ", "))
	}
	planted := []string{"docs/VISION.md", "control/policy.yaml", "infra/platform/ci/code.gitlab-ci.yml", "plugin/plugin.json", "x/settings.json"}
	if got := yamlOutsideAllowed(planted); !slices.Equal(got, []string{"control/policy.yaml", "x/settings.json"}) {
		t.Fatalf("planted files found as %v", got)
	}
}

// TestConfigDocumentsRead reads every config document this repository
// holds (design.md D11, srd040 R2.2): the gates file and each config
// format's example.
func TestConfigDocumentsRead(t *testing.T) {
	if _, err := release.LoadGates(filepath.Join("..", release.DefaultGatesFile)); err != nil {
		t.Error(err)
	}
	formats, err := filepath.Glob(filepath.Join("..", "docs", "specs", "config-formats", "*.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range formats {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		_, rest, ok := strings.Cut(string(b), "<!-- field: config_file_layout -->")
		if !ok {
			continue
		}
		_, rest, _ = strings.Cut(rest, "```markdown\n")
		doc, _, _ := strings.Cut(rest, "\n```")
		var v map[string]any
		if err := corpus.DecodeConfig([]byte(doc+"\n"), &v); err != nil || len(v) == 0 {
			t.Errorf("%s: example does not read as a config document: %v", f, err)
		}
	}
}
