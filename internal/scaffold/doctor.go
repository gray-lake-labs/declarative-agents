package scaffold

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/gray-lake-labs/scrum-master"
	"github.com/gray-lake-labs/scrum-master/internal/install"
	"github.com/gray-lake-labs/scrum-master/internal/program"
)

// DoctorOptions are what tpm doctor is given.
type DoctorOptions struct {
	Dir     string
	Version string // the running tpm's version, which installed skills are checked against
	DryRun  bool
}

// DoctorResult is what tpm doctor reports: the repository's level, what
// happened to each file tpm scaffolds there, and the installed skills that
// differ from this tpm.
type DoctorResult struct {
	Dir    string            `json:"dir"`
	Level  string            `json:"level"`
	Files  []Result          `json:"files"`
	Skills []install.Finding `json:"skills,omitempty"`
	DryRun bool              `json:"dry_run"`
}

// systemSRDPath and subsystemSRDPath are the SRDs init and onboard write, by which
// doctor knows a repository's level.
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
// and reports the installed skills tpm install would change (srd043 R4).
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
