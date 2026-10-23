package scaffold

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/gray-lake-labs/scrum-master"
	"github.com/gray-lake-labs/scrum-master/internal/install"
	"github.com/gray-lake-labs/scrum-master/internal/program"
	"github.com/gray-lake-labs/scrum-master/internal/routing"
	"github.com/gray-lake-labs/scrum-master/internal/runtime"
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
// differ from this tpm, the repair of the documents' format, and the
// declarative-agents runtime at the release tpm pins.
type DoctorResult struct {
	Dir     string            `json:"dir"`
	Level   string            `json:"level"`
	Files   []Result          `json:"files"`
	Skills  []install.Finding `json:"skills,omitempty"`
	Repair  RepairResult      `json:"repair"`
	Runtime runtime.Status    `json:"runtime"`
	// Credentials is, per shipped profile, every environment variable its
	// declarations read and whether each is set here, so a run's model and
	// tracker credentials are checked before a run (srd043 R4.7, GH-135).
	Credentials []ProfileEnv `json:"credentials,omitempty"`
	// Trackers is the program's tracking systems, per control/trackers.md,
	// and each problem reaching them (srd042).
	Trackers *routing.Report `json:"trackers,omitempty"`
	DryRun   bool            `json:"dry_run"`
}

// RepairResult is what the repair did to the corpus documents: those
// rewritten in canonical form (or, on a dry run, those that would be), the
// findings the formatter cannot repair with their fix hints, and why the
// repair was skipped, if it was (srd043 R4.4, R4.5).
type RepairResult struct {
	Repaired []string `json:"repaired,omitempty"`
	Findings []string `json:"findings,omitempty"`
	Skipped  string   `json:"skipped,omitempty"`
	// Untracked are documents under docs/ git does not track: the spec
	// tool reads only tracked ones, so they were not checked.
	Untracked []string `json:"untracked,omitempty"`
}

// ProfileEnv is one shipped profile's environment: the variables its
// declarations read, each with whether it is set here.
type ProfileEnv struct {
	Profile string   `json:"profile"`
	Vars    []EnvVar `json:"vars"`
}

// EnvVar is one environment variable a profile names.
type EnvVar struct {
	Name string `json:"name"`
	Set  bool   `json:"set"`
}

// profileCredentials reads every shipped profile's environment variables and
// checks each against this process's environment.
func profileCredentials() ([]ProfileEnv, error) {
	env, err := install.ProfileEnv(scrummaster.Plugin, scrummaster.AgentsDir)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(env))
	for name := range env {
		names = append(names, name)
	}
	slices.Sort(names)
	var out []ProfileEnv
	for _, name := range names {
		p := ProfileEnv{Profile: name}
		for _, v := range env[name] {
			_, set := os.LookupEnv(v)
			p.Vars = append(p.Vars, EnvVar{Name: v, Set: set})
		}
		out = append(out, p)
	}
	return out, nil
}

// specTool runs the spec tool at a repository's pin, lookGo finds the Go
// toolchain the tool is built with, built reports a tool already built at a
// release, and inspectRuntime reads the runtime at one; tests replace them.
var (
	specTool       = spectool.Run
	lookGo         = func() error { _, err := exec.LookPath("go"); return err }
	built          = runtime.Built
	inspectRuntime = runtime.Inspect
)

// notCanonical ends the line the tool's check prints for a document fmt
// would rewrite.
const notCanonical = ": not-canonical: run mage fmt"

// repair rewrites the corpus documents at dir in the canonical form of the
// parent release the repository pins, with that release's own formatter,
// then reports what the formatter cannot repair. A dry run only checks.
func repair(dir string, dryRun bool) RepairResult {
	var r RepairResult
	if out, err := exec.Command("git", "-C", dir, "ls-files", "--others", "--exclude-standard", "--", "docs").Output(); err == nil {
		for _, p := range strings.Fields(string(out)) {
			if strings.HasSuffix(p, ".md") {
				r.Untracked = append(r.Untracked, p)
			}
		}
	}
	if lookGo() != nil {
		tag, err := spectool.Pinned(dir)
		if err != nil || !built(tag) {
			r.Skipped = "no Go toolchain on PATH and no spec tool built at the pinned release; install Go, or run tpm install --runtime with Go where tpm pins the same release"
			return r
		}
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

// extendLabels adds to a kept control/labels.md the axes and values tpm's
// defaults list that it lacks, and reports the file extended with them; a
// file that does not parse is kept and its reason reported (srd043 R4.2).
// The tracker's labels come from tpm issue labels, which the reason names.
func extendLabels(dir string, files []Result, dryRun bool) error {
	i := slices.IndexFunc(files, func(f Result) bool { return f.Path == program.LabelsFile })
	if i < 0 || files[i].Action != "kept" {
		return nil
	}
	path := filepath.Join(dir, filepath.FromSlash(program.LabelsFile))
	doc, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	out, added, err := ExtendLabels(doc, program.DefaultLabels())
	if err != nil {
		files[i].Reason = "does not parse, so doctor cannot add the labels tpm's defaults list: " + err.Error()
		return nil
	}
	if len(added) == 0 {
		return nil
	}
	files[i].Action, files[i].Labels = "extended", added
	files[i].Reason = "adds " + strings.Join(added, ", ") + "; run tpm issue labels --control <system repository> to create them in the tracker"
	if dryRun {
		return nil
	}
	return os.WriteFile(path, out, 0o644)
}

// skillProblems are the install findings doctor reports (srd043 R4.3).
var skillProblems = map[string]bool{"missing": true, "outdated": true, "edited": true, "stale": true}

// Doctor brings a repository an older tpm laid down up to date: it writes
// the files this tpm scaffolds at the repository's level that are missing,
// refreshes the orientation files' managed blocks, keeps every other file,
// repairs the documents to the pinned format, and reports the installed
// skills tpm install would change and the runtime at the release tpm pins
// (srd043 R4).
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
	if r.Trackers, err = routing.Check(o.Dir); err != nil {
		return r, err
	}
	files, err := corpusFiles(level, o.Dir)
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
	if level == "system" {
		if err := extendLabels(o.Dir, r.Files, o.DryRun); err != nil {
			return r, err
		}
		if err := upgradeSystem(o.Dir, r.Files, o.DryRun); err != nil {
			return r, err
		}
	}
	r.Repair = repair(o.Dir, o.DryRun)
	tag, err := runtime.Pin()
	if err != nil {
		return r, err
	}
	r.Runtime = inspectRuntime(tag)
	if r.Credentials, err = profileCredentials(); err != nil {
		return r, err
	}
	if agents := install.InstalledAgents(o.Dir); len(agents) > 0 {
		skills, err := install.EmbeddedSkills()
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
