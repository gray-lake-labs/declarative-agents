package scaffold

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gray-lake-labs/scrum-master"
	"github.com/gray-lake-labs/scrum-master/internal/install"
	"github.com/gray-lake-labs/scrum-master/internal/program"
	"github.com/gray-lake-labs/scrum-master/internal/spectool"
)

// DoctorOptions are what tpm doctor is given.
type DoctorOptions struct {
	Dir     string
	Version string // the running tpm's version, which installed skills are checked against
	DryRun  bool
}

// DoctorResult is what tpm doctor reports: the repository's level, what
// happened to each file tpm scaffolds there, the installed skills that
// differ from this tpm, and the repair of the documents' format.
type DoctorResult struct {
	Dir    string            `json:"dir"`
	Level  string            `json:"level"`
	Files  []Result          `json:"files"`
	Skills []install.Finding `json:"skills,omitempty"`
	Repair RepairResult      `json:"repair"`
	DryRun bool              `json:"dry_run"`
}

// RepairResult is what the repair did to the corpus documents: those
// rewritten in canonical form (or, on a dry run, those that would be), the
// findings the formatter cannot repair with their fix hints, and why the
// repair was skipped, if it was (srd043 R4.4, R4.5).
type RepairResult struct {
	Repaired []string `json:"repaired,omitempty"`
	Findings []string `json:"findings,omitempty"`
	Skipped  string   `json:"skipped,omitempty"`
}

// specTool runs the spec tool at a repository's pin, and lookGo finds the Go
// toolchain the tool is built with; tests replace both.
var (
	specTool = spectool.Run
	lookGo   = func() error { _, err := exec.LookPath("go"); return err }
)

// notCanonical ends the line the tool's check prints for a document fmt
// would rewrite.
const notCanonical = ": not-canonical: run mage fmt"

// repair rewrites the corpus documents at dir in the canonical form of the
// parent release the repository pins, with that release's own formatter,
// then reports what the formatter cannot repair. A dry run only checks.
func repair(dir string, dryRun bool) RepairResult {
	var r RepairResult
	if lookGo() != nil {
		r.Skipped = "no Go toolchain on PATH; the repair builds the pinned release's formatter with it"
		return r
	}
	skip := func(err error) {
		r.Skipped = err.Error()
		if strings.Contains(r.Skipped, "no corpus document matches") {
			r.Skipped = "git tracks no corpus document here; commit the documents, then run tpm doctor again"
		}
	}
	if !dryRun {
		lines, err := specTool(dir, "fmt", "all")
		if err != nil && len(lines) == 0 {
			skip(err)
			return r
		}
		for _, l := range lines {
			if p, ok := strings.CutPrefix(l, "formatted "); ok {
				r.Repaired = append(r.Repaired, p)
			}
		}
	}
	lines, err := specTool(dir, "check", "all")
	if err != nil && len(lines) == 0 {
		skip(err)
		return r
	}
	for _, l := range lines {
		if p, ok := strings.CutSuffix(l, notCanonical); ok && dryRun {
			r.Repaired = append(r.Repaired, p)
			continue
		}
		r.Findings = append(r.Findings, strings.TrimSuffix(l, ": run mage fmt"))
	}
	return r
}

// systemSRDPath and subsystemSRDPath are the SRDs init and onboard write,
// by which doctor knows a repository's level.
const (
	systemSRDPath    = "docs/specs/software-requirements/srd001-system.md"
	subsystemSRDPath = "docs/specs/software-requirements/srd001-subsystem.md"
)

// Level reads a program repository's level from the files tpm wrote there:
// system when the policy or the system SRD is present, subsystem when the
// subsystem SRD is (srd043 R4.1).
func Level(dir string) (string, error) {
	exists := func(p string) bool {
		_, err := os.Stat(filepath.Join(dir, filepath.FromSlash(p)))
		return err == nil
	}
	switch {
	case exists(program.PolicyFile) || exists(systemSRDPath):
		return "system", nil
	case exists(subsystemSRDPath):
		return "subsystem", nil
	}
	return "", fmt.Errorf("%s holds neither %s, %s, nor %s, so it is no program repository tpm laid down; run tpm init or tpm onboard there",
		dir, program.PolicyFile, systemSRDPath, subsystemSRDPath)
}

// skillProblems are the install findings doctor reports (srd043 R4.3).
var skillProblems = map[string]bool{"missing": true, "outdated": true, "edited": true, "stale": true}

// Doctor brings a repository an older tpm laid down up to date: it writes
// the files this tpm scaffolds at the repository's level that are missing,
// refreshes the orientation files' managed blocks, keeps every other file,
// repairs the documents to the pinned format, and reports the installed
// skills tpm install would change (srd043 R4).
// It files nothing and touches no tracker.
func Doctor(o DoctorOptions) (DoctorResult, error) {
	r := DoctorResult{Dir: o.Dir, DryRun: o.DryRun}
	if err := requireRepository(o.Dir); err != nil {
		return r, err
	}
	level, err := Level(o.Dir)
	if err != nil {
		return r, err
	}
	r.Level = level
	files, err := corpusFiles(level)
	if err != nil {
		return r, err
	}
	if level == "system" {
		for n, b := range controlFiles() {
			files[n] = b
		}
	}
	if r.Files, err = write(o.Dir, files, false, o.DryRun); err != nil {
		return r, err
	}
	for i, f := range r.Files {
		if f.Action == "kept" && f.Reason == "edited; --force replaces it" {
			r.Files[i].Reason = "edited; doctor keeps it"
		}
	}
	r.Repair = repair(o.Dir, o.DryRun)
	if agents := install.InstalledAgents(o.Dir); len(agents) > 0 {
		skills, err := install.Skills(scrummaster.Plugin, scrummaster.SkillsDir)
		if err != nil {
			return r, err
		}
		findings, err := install.Check(o.Dir, agents, skills, o.Version)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return r, err
		}
		for _, f := range findings {
			if skillProblems[f.Problem] {
				r.Skills = append(r.Skills, f)
			}
		}
	}
	return r, nil
}
