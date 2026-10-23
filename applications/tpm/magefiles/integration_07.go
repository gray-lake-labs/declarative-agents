// Copyright (c) 2026 Gray Lake Labs
// SPDX-License-Identifier: MIT-0
package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gray-lake-labs/declarative-agents/applications/tpm/internal/help"
	"github.com/gray-lake-labs/declarative-agents/applications/tpm/internal/standup"
	"github.com/gray-lake-labs/declarative-agents/applications/tpm/internal/tracker"
)

func init() {
	integrationTargets = append(integrationTargets, Integration{}.IssueGroup)
}

// IssueGroup proves tpm help teaches the process from the binary, the issue group serves its subcommands, tpm program runs nothing and names its fix, and tpm issue recur opens the next instance once, finding one an earlier tpm standup recur opened (test-rel01.4-agent-legible-cli).
func (Integration) IssueGroup() error {
	r, err := newRigEnv()
	if err != nil {
		return err
	}
	po1, err := r.project("product-owners/po-1")
	if err != nil {
		return err
	}
	c := &check{}

	usage, err := r.runTPM("issue", "--help")
	if err != nil {
		return err
	}
	for _, sub := range []string{"approve", "close", "file", "labels", "lint", "merge", "pop", "recur"} {
		c.expect(strings.Contains(string(usage), "\n  "+sub+" "), "tpm issue --help lists no %s:\n%s", sub, usage)
	}

	process, err := r.runTPM("help")
	c.expect(err == nil && strings.Contains(string(process), "Three planes") && strings.Contains(string(process), "docs/VISION.md"),
		"tpm help: %v\n%s", err, process)
	var objective help.Page
	err = json.Unmarshal(must(r.runTPM("help", "objective", "--json")), &objective)
	c.expect(err == nil && objective.Name == "objective" && len(objective.Commands) > 0 && len(objective.SRDs) > 0,
		"tpm help objective --json: %+v %v", objective, err)
	_, err = r.runTPM("help", "no-such-page")
	c.expect(err != nil && strings.Contains(err.Error(), "objective"), "tpm help no-such-page: %v, want a refusal listing the pages", err)

	_, err = r.runTPM("program", "pop", "--project", po1, "--issue", "1")
	c.expect(err != nil && strings.Contains(err.Error(), "tpm issue pop") && strings.Contains(err.Error(), "tpm install"),
		"tpm program pop: %v, want a refusal naming tpm issue pop and tpm install", err)

	recurring := func(title, due, body string) (tracker.Issue, error) {
		return r.tr.CreateIssue(po1, tracker.NewIssue{Title: title, Labels: []string{"type::recurring"}, DueDate: due, Description: body})
	}
	recur := func(iid int) (standup.RecurResult, error) {
		var res standup.RecurResult
		return res, r.tpmJSON(&res, "issue", "recur", "--project", po1, "--issue", fmt.Sprint(iid))
	}
	fresh, err := recurring("Recurring: rig audit", "2026-10-15", "Audit the rig.\n\n## Previous Runs\n\n- none\n")
	if err != nil {
		return err
	}
	first, err := recur(fresh.IID)
	if err != nil {
		return err
	}
	again, err := recur(fresh.IID)
	if err != nil {
		return err
	}
	c.expect(!first.Existing && first.Next != 0 && again.Existing && again.Next == first.Next,
		"recur opened the next instance more than once: %+v then %+v", first, again)

	moved, err := recurring("Recurring: rig sweep", "2026-10-15", "Sweep the rig.\n\n## Previous Runs\n\n- none\n")
	if err != nil {
		return err
	}
	earlier, err := recurring("Recurring: rig sweep", "2026-10-22", fmt.Sprintf(
		"Sweep the rig.\n\n## Previous Runs\n\n- %s#%d (due 2026-10-15): closed by tpm standup recur without its work.\n\n<!-- tpm standup recurs %s#%d -->\n",
		po1, moved.IID, po1, moved.IID))
	if err != nil {
		return err
	}
	found, err := recur(moved.IID)
	if err != nil {
		return err
	}
	c.expect(found.Existing && found.Next == earlier.IID, "recur missed the instance an earlier tpm opened: %+v, want #%d", found, earlier.IID)
	return c.err("issueGroup")
}

// must returns out, dropping the error a later check reports.
func must(out []byte, _ error) []byte { return out }
