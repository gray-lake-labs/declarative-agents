package release

import (
	"testing"

	"github.com/gray-lake-labs/scrum-master/internal/gitlab/gitlabtest"
	"github.com/gray-lake-labs/scrum-master/internal/program"
)

func TestNextReportsLatestAndCandidates(t *testing.T) {
	r := program.Defaults().Releases
	n := Next(r, "a", []string{"a/v0.1.0", "a/v0.2.3", "b/v9.0.0", "a/v0.2.3-rc1", "v0.20261015.2"})
	if n.Latest != "0.2.3" || n.LatestTag != "a/v0.2.3" {
		t.Fatalf("latest %+v", n)
	}
	want := []Candidate{
		{"patch", "0.2.4", "a/v0.2.4", "compatible"},
		{"minor", "0.3.0", "a/v0.3.0", "breaking"},
		{"major", "1.0.0", "a/v1.0.0", "breaking"},
	}
	for i, c := range want {
		if n.Candidates[i] != c {
			t.Fatalf("candidate %d: %+v, want %+v", i, n.Candidates[i], c)
		}
	}

	n = Next(r, "fresh", nil)
	if n.Latest != "" || n.Candidates[0].Version != "0.0.1" || n.Candidates[1].Version != "0.1.0" || n.Candidates[2].Version != "1.0.0" {
		t.Fatalf("no tag: %+v", n)
	}
}

func TestBreakingClassFollowsPolicy(t *testing.T) {
	v := func(s string) Version { x, _ := ParseVersion(s); return x }
	for _, c := range []struct {
		rule, from, to string
		breaking       bool
	}{
		{"major-and-minor-before-1.0", "0.2.3", "0.3.0", true},
		{"major-and-minor-before-1.0", "0.2.3", "0.2.4", false},
		{"major-and-minor-before-1.0", "1.2.3", "1.3.0", false},
		{"major", "0.2.3", "0.3.0", false},
		{"major", "1.2.3", "2.0.0", true},
	} {
		if got := Breaking(c.rule, v(c.from), v(c.to)); got != c.breaking {
			t.Errorf("%s %s -> %s: breaking %v", c.rule, c.from, c.to, got)
		}
	}
}

func TestPolicyLoadsReleasesAndDefinitionOfDone(t *testing.T) {
	f := gitlabtest.New()
	f.AddFile("g/system", "main", "control/policy.md", []byte(`---
id: policy
title: Policy
---

<!-- field: version -->
## Version

1

<!-- field: releases -->
## Releases

<!-- field: scheme -->
### Scheme

semver

<!-- field: breaking -->
### Breaking

major

<!-- field: tag_format -->
### Tag format

v{version}

<!-- field: definition_of_done -->
## Definition of done

<!-- field: corpus -->
### Corpus

| kind | environment | command |
| --- | --- | --- |
| custom | local | mage audit |

<!-- field: agent-profile -->
### Agent-profile

| kind | environment | against | threshold |
| --- | --- | --- | --- |
| scorecard | dev | current | no-regression |
| peer-read |  |  |  |
`))
	p, err := program.LoadPolicy(f, "g/system", "main")
	if err != nil {
		t.Fatal(err)
	}
	if p.Releases.Breaking != "major" || p.Releases.TagFormat != "v{version}" {
		t.Fatalf("releases %+v", p.Releases)
	}
	if g := p.Gate("corpus"); len(g) != 1 || g[0].Command != "mage audit" {
		t.Fatalf("corpus gate %+v", g)
	}
	if g := p.Gate("agent-profile"); len(g) != 2 || g[1].Kind != "peer-read" || g[0].Threshold != "no-regression" {
		t.Fatalf("agent-profile gate %+v", g)
	}
	if g := p.Gate("other"); g != nil {
		t.Fatalf("an artifact with no entry and no default gets %+v", g)
	}
	if g := program.Defaults().Gate("anything"); len(g) != 1 || g[0].Command != "go test ./..." {
		t.Fatalf("default gate %+v", g)
	}
}
