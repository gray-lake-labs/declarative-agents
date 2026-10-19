package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gray-lake-labs/scrum-master/internal/scaffold"
)

func init() {
	integrationTargets = append(integrationTargets, Integration{}.Onboard)
}

// Onboard onboards a new repository into the rig's program: its corpus passes the parent's corpus audit as written, the system repository gets one registration issue and no commit, and a second onboard files none (test-rel01.4-agent-legible-cli).
func (Integration) Onboard() error {
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
	cache, err := parentCheckout(r.root)
	if err != nil {
		return err
	}
	store := filepath.Join(r.run.Dir, "joining", "store")
	if err := os.MkdirAll(store, 0o755); err != nil {
		return err
	}
	if err := gitIn(store, "init", "-q", "-b", "main"); err != nil {
		return err
	}
	head := func() (string, error) {
		return gitOut(r.run.Dir, "--git-dir", r.origin(system), "rev-parse", "refs/heads/main")
	}
	before, err := head()
	if err != nil {
		return err
	}
	onboard := func() (scaffold.OnboardResult, error) {
		var res scaffold.OnboardResult
		return res, r.tpmJSON(&res, "onboard", "--dir", store, "--system", system, "--ref", seedBranch,
			"--subsystem", "srd010-po-1", "--repository", r.webBase+"/rig/product-owners/store", "--agent", "claude")
	}
	c := &check{}
	first, err := onboard()
	if err != nil {
		return err
	}
	c.expect(first.Registration.Filed && first.Registration.Issue != nil, "no registration filed: %+v", first.Registration)
	if first.Registration.Issue != nil {
		i, err := r.issue(system, first.Registration.Issue.IID)
		c.expect(err == nil && strings.Contains(i.Title, "rig/product-owners/store") && strings.Contains(i.Description, "srd010-po-1"),
			"registration issue: %+v %v", i, err)
	}

	writeFileTo(filepath.Join(store, checkoutsFile), "checkouts:\n  "+parentRepository+": "+cache+"\n")
	if err := gitIn(store, "add", "-A"); err != nil {
		return err
	}
	c.expect(runCorpusAudit(store, runStreaming) == nil, "the parent's corpus audit fails on the corpus tpm onboard wrote")

	second, err := onboard()
	if err != nil {
		return err
	}
	c.expect(!second.Registration.Filed && strings.Contains(second.Registration.Reason, "already asks"),
		"second onboard: %+v", second.Registration)
	for _, f := range second.Files {
		c.expect(f.Action == "kept", "second onboard: %s %s", f.Path, f.Action)
	}
	open, err := r.issues(system)
	if err != nil {
		return err
	}
	registrations := 0
	for _, i := range open {
		if strings.Contains(i.Description, "tpm onboard registers") {
			registrations++
		}
	}
	c.expect(registrations == 1, "%d registration issues in the system repository, want 1", registrations)
	after, err := head()
	c.expect(err == nil && after == before, "onboard committed to the system repository: %s, was %s (%v)", after, before, err)
	return c.err("onboard")
}
