package scaffold

import (
	"fmt"
	"strings"

	"github.com/gray-lake-labs/scrum-master"
	"github.com/gray-lake-labs/scrum-master/internal/install"
	"github.com/gray-lake-labs/scrum-master/internal/program"
	"github.com/gray-lake-labs/scrum-master/internal/tracker"
)

// OnboardOptions are what tpm onboard is given.
type OnboardOptions struct {
	Dir     string
	Agents  []string
	Version string
	// System is the system repository the registration is filed in, read at
	// Ref; Subsystem is the subsystem SRD whose components table gains the
	// repository.
	System, Ref, Subsystem string
	// Repository is the joining repository's URL, Path the path of the
	// component in it.
	Repository, Path string
	Force, DryRun    bool
}

// OnboardResult is what tpm onboard reports.
type OnboardResult struct {
	Dir          string           `json:"dir"`
	Files        []Result         `json:"files"`
	Skills       []install.Result `json:"skills,omitempty"`
	Registration Registration     `json:"registration"`
	DryRun       bool             `json:"dry_run"`
}

// Registration is the issue onboard filed in the system repository, or why
// it filed none (srd043 R2.2, R2.3).
type Registration struct {
	Filed  bool                `json:"filed"`
	Issue  *program.FileResult `json:"issue,omitempty"`
	Reason string              `json:"reason,omitempty"`
}

// registrationMarker names the repository a registration issue registers, so
// a second onboard finds it.
func registrationMarker(repository string) string {
	return "<!-- tpm onboard registers " + program.ProjectPath(repository) + " -->"
}

// Onboard lays down a joining repository's corpus skeleton in o.Dir and
// files its registration in the system repository (srd043 R2): one issue
// asking the subsystem SRD's owner to list the repository, never a commit
// there.
func Onboard(t tracker.Tracker, o OnboardOptions) (OnboardResult, error) {
	r := OnboardResult{Dir: o.Dir, DryRun: o.DryRun}
	if err := requireRepository(o.Dir); err != nil {
		return r, err
	}
	if o.System == "" || o.Subsystem == "" || o.Repository == "" {
		return r, fmt.Errorf("onboard needs the system repository, the subsystem SRD, and the repository's URL")
	}
	if o.Path == "" {
		o.Path = "/"
	}
	files, err := corpusFiles("subsystem")
	if err != nil {
		return r, err
	}
	if r.Files, err = write(o.Dir, files, o.Force, o.DryRun); err != nil {
		return r, err
	}
	if len(o.Agents) > 0 {
		skills, err := install.Skills(scrummaster.Plugin, scrummaster.SkillsDir)
		if err != nil {
			return r, err
		}
		if r.Skills, err = install.Install(o.Dir, o.Agents, skills, o.Version, o.Force, o.DryRun); err != nil {
			return r, err
		}
	}
	r.Registration, err = register(t, o)
	return r, err
}

func register(t tracker.Tracker, o OnboardOptions) (Registration, error) {
	subs, err := program.LoadSubsystems(t, o.System, o.Ref)
	if err != nil {
		return Registration{}, err
	}
	if _, sub, ok := subs.ComponentFor(o.Repository); ok {
		return Registration{Reason: fmt.Sprintf("%s already lists %s", sub.ID, program.ProjectPath(o.Repository))}, nil
	}
	marker := registrationMarker(o.Repository)
	open, err := t.ListIssues(o.System, tracker.IssueQuery{State: "opened"})
	if err != nil {
		return Registration{}, err
	}
	for _, i := range open {
		if strings.Contains(i.Description, marker) {
			return Registration{Reason: fmt.Sprintf("%s#%d already asks to register %s", o.System, i.IID, program.ProjectPath(o.Repository))}, nil
		}
	}
	policy, err := program.LoadPolicy(t, o.System, o.Ref)
	if err != nil {
		return Registration{}, err
	}
	in := tracker.NewIssue{
		Title:       fmt.Sprintf("Register %s in %s", program.ProjectPath(o.Repository), o.Subsystem),
		Description: registrationBody(o) + "\n\n" + marker + "\n",
		Labels:      append(program.Labels("management", "", "", ""), program.Op(program.OpOnboard)),
	}
	res, err := program.File(t, o.System, in, policy, o.DryRun)
	if err != nil {
		return Registration{}, err
	}
	return Registration{Filed: !o.DryRun, Issue: &res}, nil
}

// registrationBody asks the subsystem SRD's owner for the components-table
// row, in do-work's input format.
func registrationBody(o OnboardOptions) string {
	repo := program.ProjectPath(o.Repository)
	return fmt.Sprintf(`%[1]s joins the program and asks to be listed in %[2]s (design.md D7: membership is declared, not inferred).

## Requirements

1. In docs/specs/software-requirements/%[2]s.md, add the row | <component name> | %[3]s | %[4]s | to the implementation targets table, naming the component.

## Acceptance Criteria

- %[2]s lists %[1]s, and the corpus audit passes (test: mage_audit; environment: local)`, repo, o.Subsystem, o.Repository, o.Path)
}
