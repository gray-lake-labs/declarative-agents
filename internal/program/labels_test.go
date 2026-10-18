package program

import (
	"os"
	"strings"
	"testing"

	"github.com/gray-lake-labs/scrum-master/internal/gitlab"
	"github.com/gray-lake-labs/scrum-master/internal/gitlab/gitlabtest"
)

func actions(res []LabelResult) map[string]int {
	m := map[string]int{}
	for _, r := range res {
		m[r.Action]++
	}
	return m
}

func TestLabelsCreatesOnceAndNeverDeletes(t *testing.T) {
	f := gitlabtest.New()
	f.Labels["g"] = []gitlab.Label{{Name: "team-only", Color: "#000000"}}
	want := DefaultLabels()

	res, err := SyncLabels(f, "g", want, false, false)
	if err != nil || actions(res)["created"] != len(want) {
		t.Fatalf("first run: %v %v", actions(res), err)
	}
	writes := f.Writes
	res, err = SyncLabels(f, "g", want, false, false)
	if err != nil || actions(res)["unchanged"] != len(want) || f.Writes != writes {
		t.Fatalf("second run: %v writes=%d err=%v", actions(res), f.Writes-writes, err)
	}
	if len(f.Labels["g"]) != len(want)+1 || f.Labels["g"][0].Name != "team-only" {
		t.Fatalf("labels: %+v", f.Labels["g"])
	}
}

func TestDifferingLabelIsReportedThenUpdatedOnRequest(t *testing.T) {
	f := gitlabtest.New()
	f.Labels["g"] = []gitlab.Label{{Name: "plane::data", Color: "#2CA02C", Description: "managed by terraform"}}
	want := []gitlab.Label{{Name: "plane::data", Color: "#1f75cb", Description: "Work"}}

	res, _ := SyncLabels(f, "g", want, false, false)
	if res[0].Action != "differs" || f.Labels["g"][0].Color != "#2CA02C" {
		t.Fatalf("without update: %+v %+v", res, f.Labels["g"])
	}
	res, _ = SyncLabels(f, "g", want, true, false)
	if res[0].Action != "updated" || f.Labels["g"][0].Color != "#1f75cb" {
		t.Fatalf("with update: %+v %+v", res, f.Labels["g"])
	}
}

func TestLabelsColorCompareIgnoresCaseAndDryRunWritesNothing(t *testing.T) {
	f := gitlabtest.New()
	f.Labels["g"] = []gitlab.Label{{Name: "kind::spec", Color: "#8E7CC3", Description: "Changes requirements or design documents under docs/"}}
	res, err := SyncLabels(f, "g", DefaultLabels(), true, true)
	if err != nil || f.Writes != 0 {
		t.Fatalf("dry run wrote: %d %v", f.Writes, err)
	}
	for _, r := range res {
		if r.Name == "kind::spec" && r.Action != "unchanged" {
			t.Fatalf("case-different color treated as change: %+v", r)
		}
	}
}

func TestLoadLabelsReadsControlFileOrDefaults(t *testing.T) {
	f := gitlabtest.New()
	if l, err := LoadLabels(f, "g/system", "main"); err != nil || len(l) != len(DefaultLabels()) {
		t.Fatalf("defaults: %d %v", len(l), err)
	}
	f.AddFile("g/system", "main", "control/labels.md", []byte(`---
id: labels
title: Labels
---

<!-- field: axes -->
## Axes

<!-- field: plane -->
### Plane

<!-- field: prefix -->
#### Prefix

plane::

<!-- field: values -->
#### Values

| name | color | description |
| --- | --- | --- |
| data | #1f75cb | Work |

<!-- field: kind -->
### Kind

<!-- field: prefix -->
#### Prefix

kind::

<!-- field: values -->
#### Values

| name | color | description |
| --- | --- | --- |
| spec | #8e7cc3 | Design |

<!-- field: component -->
### Component

<!-- field: prefix -->
#### Prefix

component::

<!-- field: values_from -->
#### Values from

subsystem SRDs
`))
	l, err := LoadLabels(f, "g/system", "main")
	if err != nil || len(l) != 2 || l[0].Name != "plane::data" || l[1].Name != "kind::spec" {
		t.Fatalf("control file: %+v %v", l, err)
	}
}

// TestConfigFormatExamplesDecode holds the labels and policy config formats
// to their own examples: each example is a config document tpm reads.
func TestConfigFormatExamplesDecode(t *testing.T) {
	f := gitlabtest.New()
	f.AddFile("g/system", "main", LabelsFile, specExample(t, "labels"))
	f.AddFile("g/system", "main", PolicyFile, specExample(t, "policy"))
	labels, err := LoadLabels(f, "g/system", "main")
	if err != nil || len(labels) == 0 || labels[0].Name != "plane::management" {
		t.Fatalf("labels = %v, %v", labels, err)
	}
	p, err := LoadPolicy(f, "g/system", "main")
	if err != nil {
		t.Fatal(err)
	}
	if p.AuditCommand != "mage audit" || p.Escalation.BlockerAgeDays != 14 || len(p.Gate("agent-profile")) != 2 || p.Gate("corpus")[0].Command != "mage audit" {
		t.Fatalf("policy = %+v", p)
	}
}

// specExample is the config document a config format shows in its
// config_file_layout field.
func specExample(t *testing.T, format string) []byte {
	t.Helper()
	b, err := os.ReadFile("../../docs/specs/config-formats/" + format + ".md")
	if err != nil {
		t.Fatal(err)
	}
	_, rest, ok := strings.Cut(string(b), "<!-- field: config_file_layout -->")
	if !ok {
		t.Fatalf("%s has no config_file_layout", format)
	}
	_, rest, _ = strings.Cut(rest, "```markdown\n")
	doc, _, _ := strings.Cut(rest, "\n```")
	return []byte(doc + "\n")
}
