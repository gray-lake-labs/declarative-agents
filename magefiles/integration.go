package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/magefile/mage/mg"

	"github.com/gray-lake-labs/scrum-master/internal/events"
	"github.com/gray-lake-labs/scrum-master/internal/gitlab"
	"github.com/gray-lake-labs/scrum-master/internal/goals"
	"github.com/gray-lake-labs/scrum-master/internal/objectives"
	"github.com/gray-lake-labs/scrum-master/internal/program"
)

// Integration holds the release integration targets the test suites name.
// They run against the current rig (mage regression:up), except
// objectivesAudit, which is offline.
type Integration mg.Namespace

func init() {
	integrationTargets = append(integrationTargets,
		Integration{}.ObjectivesAudit,
		Integration{}.CreateGoals,
		Integration{}.Program,
	)
}

const (
	rigRelease   = "01.0"
	rigTag       = "v0.1.0"
	rigMilestone = "rig-01.0"
	rigDue       = "2026-11-06"
)

// ObjectivesAudit proves the agent audit passes a well-formed objective document and the gate fails a malformed one (test-rel01.0-objectives).
func (Integration) ObjectivesAudit() error {
	root, err := repositoryRoot()
	if err != nil {
		return err
	}
	cache, err := parentCheckout(root)
	if err != nil {
		return err
	}
	c := &check{}
	audit := func(fixture string) ([]objectives.Finding, error) {
		tmp, err := os.MkdirTemp("", "scrum-master-objectives-*")
		if err != nil {
			return nil, err
		}
		defer os.RemoveAll(tmp)
		if err := runStreaming(root, "cp", "-R", "docs", tmp); err != nil {
			return nil, err
		}
		writeFileTo(filepath.Join(tmp, checkoutsFile), "checkouts:\n  "+parentRepository+": "+cache+"\n")
		dst := filepath.Join(tmp, objectives.Dir, filepath.Base(fixture))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return nil, err
		}
		b, err := os.ReadFile(filepath.Join(root, fixture))
		if err != nil {
			return nil, err
		}
		writeFileTo(dst, string(b))
		docs, err := objectives.LoadDir(filepath.Join(tmp, objectives.Dir))
		if err != nil {
			return nil, err
		}
		var findings []objectives.Finding
		for _, d := range docs {
			findings = append(findings, objectives.Validate(d)...)
		}
		if len(findings) == 0 {
			return nil, runCorpusAudit(tmp, runStreaming)
		}
		return findings, nil
	}
	f, err := audit("testdata/objectives/valid/obj001-reliable-agents.md")
	c.expect(err == nil && len(f) == 0, "well-formed document: findings=%v err=%v", f, err)
	malformed, _ := filepath.Glob(filepath.Join(root, "testdata/objectives/malformed/*.md"))
	for _, m := range malformed {
		rel, _ := filepath.Rel(root, m)
		f, err := audit(rel)
		c.expect(err == nil && len(f) > 0, "%s: expected a finding, got findings=%v err=%v", rel, f, err)
	}
	return c.err("objectivesAudit")
}

// CreateGoals seeds the rig's system repository and proves create-goals files one dated, pinned objective issue per owner, once (test-rel01.0-objectives).
func (Integration) CreateGoals() error {
	r, err := newRigEnv()
	if err != nil {
		return err
	}
	system, err := r.project("system")
	if err != nil {
		return err
	}
	po1, err := r.project("product-owners/po-1")
	if err != nil {
		return err
	}
	po2, err := r.project("product-owners/po-2")
	if err != nil {
		return err
	}
	if err := r.seedSystem(system, po1, po2); err != nil {
		return err
	}
	c := &check{}
	base := []string{"--system", system, "--release", rigRelease, "--ref", seedBranch}

	var plan goals.Plan
	if err := r.tpmJSON(&plan, append([]string{"goals", "plan"}, base...)...); err != nil {
		return err
	}
	c.expect(len(plan.Objectives) == 2, "plan has %d objectives, want 2", len(plan.Objectives))
	if len(plan.Objectives) != 2 {
		return c.err("createGoals")
	}
	c.expect(plan.Objectives[0].Repository == po1, "R1 recommended %q, want %s", plan.Objectives[0].Repository, po1)
	c.expect(plan.Objectives[0].Environment == "dev", "R1 environment %q", plan.Objectives[0].Environment)
	c.expect(plan.Objectives[1].Repository == "", "R2 recommended %q, want none", plan.Objectives[1].Repository)

	// The person confirms R1 and names po-2 for R2, as the skill relays.
	plan.Objectives[1].Repository = po2
	planFile := filepath.Join(r.run.Dir, "plan.json")
	writeJSONTo(planFile, plan)
	push := append([]string{"goals", "push", "--plan", planFile}, base...)

	if _, err := r.runTPM(append(push, "--dry-run")...); err != nil {
		return err
	}
	for _, p := range []string{po1, po2} {
		is, err := r.issues(p)
		c.expect(err == nil && len(is) == 0, "dry run filed %d issues in %s (err %v)", len(is), p, err)
	}

	var pushed []goals.Pushed
	if err := r.tpmJSON(&pushed, push...); err != nil {
		return err
	}
	for _, p := range pushed {
		c.expect(p.Action == "created" && p.Planned, "%s: action %s planned %v reasons %v", p.Key, p.Action, p.Planned, p.Reasons)
	}
	for _, p := range []string{po1, po2} {
		is, err := r.issues(p)
		if err != nil || len(is) != 1 {
			c.expect(false, "%s has %d objective issues (err %v), want 1", p, len(is), err)
			continue
		}
		i := is[0]
		due := i.DueDate
		c.expect(len(i.Assignees) == 0, "%s#%d is assigned", p, i.IID)
		c.expect(slices.Contains(i.Labels, "plane::management") && !slices.Contains(i.Labels, "needs-triage"), "%s#%d labels %v", p, i.IID, i.Labels)
		c.expect(due != "", "%s#%d has no due date", p, i.IID)
		c.expect(strings.Contains(i.Description, "/-/blob/"+rigTag+"/docs/specs/objectives/obj001-reliable-agents.md#R"), "%s#%d Cites not pinned to %s", p, i.IID, rigTag)
		c.expect(strings.Contains(i.Description, "Environment: dev") && strings.Contains(i.Description, "## Notes"), "%s#%d body lacks Environment or Notes", p, i.IID)
		ns, _ := r.notes(p, i.IID)
		planned := 0
		for _, n := range ns {
			if e, ok := events.Parse(n.Body); ok && e.Name == "planned" {
				planned++
			}
		}
		c.expect(planned == 1, "%s#%d has %d planned events", p, i.IID, planned)
	}

	var again []goals.Pushed
	if err := r.tpmJSON(&again, push...); err != nil {
		return err
	}
	for _, p := range again {
		c.expect(p.Action == "unchanged" && !p.Planned, "second push: %s %s planned %v", p.Key, p.Action, p.Planned)
	}

	plan.Objectives[1].Environment = ""
	writeJSONTo(planFile, plan)
	var refused []goals.Pushed
	if err := r.tpmJSON(&refused, push...); err != nil {
		return err
	}
	c.expect(len(refused) == 2 && refused[1].Action == "refused", "missing environment not refused: %+v", refused)
	return c.err("createGoals")
}

// Program proves file and lint against a rig repository: needs-triage with reasons, cleared once fixed, and nothing written on a dry run (test-rel01.0-objectives).
func (Integration) Program() error {
	r, err := newRigEnv()
	if err != nil {
		return err
	}
	po1, err := r.project("product-owners/po-1")
	if err != nil {
		return err
	}
	if err := r.milestone(rigMilestone, rigDue); err != nil {
		return err
	}
	c := &check{}
	dir := r.run.Dir
	bad := "Unpinned work\n\nCites: " + r.webBase + "/" + po1 + "/-/blob/main/README.md#R1.1\n\n## Requirements\n\n1. Do it.\n\n## Acceptance Criteria\n\n- It is done\n"
	good := strings.Replace(strings.Replace(bad, "/-/blob/main/", "/-/blob/v0.0.1/", 1), "- It is done", "- It is done (test: it_is_done; environment: dev)", 1)
	writeFileTo(filepath.Join(dir, "bad.md"), bad)
	file := []string{"issue", "file", "--project", po1, "--title", "Lint probe", "--body-file", filepath.Join(dir, "bad.md"),
		"--plane", "data", "--kind", "build", "--component", "po-1", "--milestone", rigMilestone}

	before, _ := r.issues(po1)
	if _, err := r.runTPM(append(file, "--dry-run")...); err != nil {
		return err
	}
	after, _ := r.issues(po1)
	c.expect(len(after) == len(before), "dry-run file created an issue")

	var filed program.FileResult
	if err := r.tpmJSON(&filed, file...); err != nil {
		return err
	}
	c.expect(!filed.Lint.Admitted, "bad issue admitted")
	reasons := strings.Join(filed.Lint.Reasons, "\n")
	c.expect(strings.Contains(reasons, "not pinned") && strings.Contains(reasons, "names no (test: T; environment: E)"), "reasons: %s", reasons)
	i, err := r.issue(po1, filed.IID)
	if err != nil {
		return err
	}
	c.expect(slices.Contains(i.Labels, "needs-triage") && len(i.Assignees) == 0, "after file: labels %v assignees %d", i.Labels, len(i.Assignees))
	ns, _ := r.notes(po1, i.IID)
	c.expect(len(ns) == 1 && strings.Contains(ns[0].Body, "Admission failed"), "lint comments: %d", len(ns))

	if _, err := r.tr.UpdateIssue(po1, i.IID, gitlab.IssueUpdate{Description: &good}); err != nil {
		return err
	}
	var linted []program.LintResult
	if err := r.tpmJSON(&linted, "issue", "lint", "--project", po1, "--issue", fmt.Sprint(i.IID)); err != nil {
		return err
	}
	c.expect(len(linted) == 1 && linted[0].Admitted, "fixed issue not admitted: %+v", linted)
	i, _ = r.issue(po1, i.IID)
	c.expect(!slices.Contains(i.Labels, "needs-triage"), "needs-triage not cleared: %v", i.Labels)
	return c.err("program")
}
