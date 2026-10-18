package gitlab

import (
	"strings"
	"testing"
)

func TestTokenResolutionOrder(t *testing.T) {
	saved := tokenSource
	defer func() { tokenSource = saved }()
	var asked string
	tokenSource = func(host string) string { asked = host; return "from-glab" }

	t.Setenv("GITLAB_TOKEN", "from-env")
	if tok, err := ResolveToken("https://gitlab.example.com/api/v4"); err != nil || tok != "from-env" || asked != "" {
		t.Fatalf("env: tok=%q err=%v asked=%q", tok, err, asked)
	}
	t.Setenv("GITLAB_TOKEN", "")
	if tok, err := ResolveToken("https://gitlab.example.com/api/v4"); err != nil || tok != "from-glab" || asked != "gitlab.example.com" {
		t.Fatalf("glab: tok=%q err=%v asked=%q", tok, err, asked)
	}
	tokenSource = func(string) string { return "" }
	if _, err := ResolveToken("https://gitlab.example.com/api/v4"); err == nil || !strings.Contains(err.Error(), "glab auth login --hostname gitlab.example.com") {
		t.Fatalf("none: err=%v", err)
	}
}

func TestBaseURLDefaultsToGitLabCom(t *testing.T) {
	t.Setenv("GITLAB_BASE_URL", "")
	if BaseURL() != DefaultBaseURL {
		t.Fatal(BaseURL())
	}
	t.Setenv("GITLAB_BASE_URL", "https://git.example.org/api/v4")
	if BaseURL() != "https://git.example.org/api/v4" {
		t.Fatal(BaseURL())
	}
}

func TestIssueHasLabel(t *testing.T) {
	i := Issue{Labels: []string{"plane::data", "needs-triage"}}
	if !i.HasLabel("needs-triage") || i.HasLabel("plane::control") {
		t.Fatal(i.Labels)
	}
}
