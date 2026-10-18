package main

import (
	"fmt"
	"strings"

	"github.com/gray-lake-labs/scrum-master/internal/gitlab"
	"github.com/gray-lake-labs/scrum-master/internal/program"
	"github.com/gray-lake-labs/scrum-master/internal/release"
)

func init() {
	integrationTargets = append(integrationTargets, Integration{}.Release)
}

// Release runs tpm release next, gate, and tag on a clone of po-1 with the policy seeded in the system repository: no tag before a gate, evidence events on the release issue, peer-read pending while only the author could approve, a pushed tag after a passing gate, no second tag at the same version, and the peer read passing on a reviewer's tpm release approve (test-rel01.2-sprint-end).
func (Integration) Release() error {
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
	if err := r.seed(system, map[string]string{program.PolicyFile: `---
id: policy
title: Rig policy
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

major-and-minor-before-1.0

<!-- field: tag_format -->
### Tag format

{artifact}/v{version}

<!-- field: definition_of_done -->
## Definition of done

<!-- field: grader -->
### Grader

| kind | environment | command |
| --- | --- | --- |
| custom | local | test -d .git |

<!-- field: reviewed -->
### Reviewed

| kind | environment | command |
| --- | --- | --- |
| custom | local | test -d .git |
| peer-read |  |  |
`}); err != nil {
		return err
	}
	clone, err := r.clone(po1, "po-1-release")
	if err != nil {
		return err
	}
	issue, err := r.tr.CreateIssue(po1, gitlab.NewIssue{Title: "Release grader 0.1.0"})
	if err != nil {
		return err
	}
	ref := fmt.Sprintf("%s#%d", po1, issue.IID)
	common := []string{"--control", system, "--ref", seedBranch, "--dir", clone}
	c := &check{}

	var next release.NextResult
	if err := r.tpmJSON(&next, append([]string{"release", "next", "--artifact", "grader"}, common...)...); err != nil {
		return err
	}
	c.expect(next.Latest == "" && next.Candidates[1].Version == "0.1.0", "next before any release: %+v", next)

	_, err = r.runTPM(append([]string{"release", "tag", "--artifact", "grader", "--version", "0.1.0", "--issue", ref}, common...)...)
	c.expect(err != nil && strings.Contains(err.Error(), "no gate has run"), "tag before a gate: %v", err)

	var gate release.ReleaseGateResult
	if err := r.tpmJSON(&gate, append([]string{"release", "gate", "--artifact", "grader", "--version", "0.1.0", "--issue", ref}, common...)...); err != nil {
		return err
	}
	c.expect(gate.Passed && len(gate.Checks) == 1, "grader gate: %+v", gate)

	_, err = r.runTPM("release", "approve", "--project", po1, "--issue", fmt.Sprint(issue.IID))
	c.expect(err != nil && strings.Contains(err.Error(), "author"), "the author approved their own release: %v", err)
	_, err = r.runTPM(append([]string{"release", "gate", "--artifact", "reviewed", "--version", "0.1.0", "--issue", ref}, common...)...)
	c.expect(err != nil && strings.Contains(err.Error(), "did not pass"), "peer-read with no approval from anyone else: %v", err)

	var tagged release.TagResult
	if err := r.tpmJSON(&tagged, append([]string{"release", "tag", "--artifact", "grader", "--version", "0.1.0", "--issue", ref}, common...)...); err != nil {
		return err
	}
	c.expect(tagged.Pushed && tagged.Tag == "grader/v0.1.0" && tagged.Class == "breaking", "tag: %+v", tagged)
	_, terr := r.remoteTag(po1, "grader/v0.1.0")
	c.expect(terr == nil, "grader/v0.1.0 not at the origin: %v", terr)

	notes, err := r.notes(po1, issue.IID)
	if err != nil {
		return err
	}
	var evidence, pending, taggedNote int
	for _, n := range notes {
		switch {
		case strings.Contains(n.Body, "event=evidence") && strings.Contains(n.Body, "result=pending"):
			pending++
			evidence++
		case strings.Contains(n.Body, "event=evidence"):
			evidence++
		case strings.Contains(n.Body, "Tagged grader/v0.1.0"):
			taggedNote++
		}
	}
	c.expect(evidence == 3 && pending == 1 && taggedNote == 1, "release issue notes: %d evidence, %d pending, %d tagged", evidence, pending, taggedNote)

	_, err = r.runTPM(append([]string{"release", "tag", "--artifact", "grader", "--version", "0.1.0", "--issue", ref}, common...)...)
	c.expect(err != nil && strings.Contains(err.Error(), "not greater"), "a second tag at 0.1.0: %v", err)

	// Another person's approval is the peer read (srd015 R1.2).
	if _, err := r.runTPMAs(rigReviewer, "release", "approve", "--project", po1, "--issue", fmt.Sprint(issue.IID)); err != nil {
		return err
	}
	var reviewed release.ReleaseGateResult
	if err := r.tpmJSON(&reviewed, append([]string{"release", "gate", "--artifact", "reviewed", "--version", "0.1.0", "--issue", ref}, common...)...); err != nil {
		return err
	}
	c.expect(reviewed.Passed, "peer-read after the reviewer's approval: %+v", reviewed)
	return c.err("release")
}
