// Package scaffold lays down a program repository's corpus: tpm init for the
// system repository and tpm onboard for each repository that joins it
// (srd043).
package scaffold

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/gray-lake-labs/scrum-master/internal/beads"

	"github.com/gray-lake-labs/scrum-master"
	"github.com/gray-lake-labs/scrum-master/internal/program"
	"github.com/gray-lake-labs/scrum-master/internal/runtime"
)

//go:embed templates
var templates embed.FS

// parentRepository is the parent whose format the templates follow.
const parentRepository = "github.com/gray-lake-labs/declarative-agents"

// Result is what happened to one file: created, kept, appended, refreshed,
// or overwritten (srd043 R3.2, R3.5).
type Result struct {
	Path   string `json:"path"`
	Action string `json:"action"`
	Reason string `json:"reason,omitempty"`
	// Labels are the labels doctor added to control/labels.md.
	Labels []string `json:"labels,omitempty"`
}

// ParentRelease is the parent release the templates were written against,
// the one this tpm's own corpus pins.
func ParentRelease() (string, error) { return runtime.Pin() }

// implementationTargets is the system SRD's implementation_targets field
// for the repository in dir: one row naming the repository, at its origin's
// URL, path /. Without an origin there is no URL to declare, and the field
// is the empty list in its canonical form.
func implementationTargets(dir string) string {
	out, err := exec.Command("git", "-C", dir, "remote", "get-url", "origin").Output()
	url := strings.TrimSpace(string(out))
	if err != nil || url == "" {
		return "<!-- field: implementation_targets | form: \"empty list\" -->\n## Implementation targets\n\nNone\n"
	}
	return fmt.Sprintf("<!-- field: implementation_targets -->\n## Implementation targets\n\n| name | repository | path |\n| --- | --- | --- |\n| %s | %s | / |\n",
		path.Base(program.ProjectPath(url)), url)
}

// levels are the owners the templates name, by the level a repository is at.
var levels = map[string]string{
	"system":    "the technical program manager",
	"subsystem": "the Product Owner",
}

// corpusFiles is the corpus skeleton every repository of a program carries
// (design.md D7), for a repository at level: the common templates and the
// level's own orientation files (srd043 R1.6, R2.4) with the level and its
// owner filled in, the design constitution, the charter, and the parent pin.
// At the system level the system SRD declares the system repository in dir
// as a component of the program, so its issues carry a component like any
// team repository's.
func corpusFiles(level, dir string) (map[string][]byte, error) {
	owner, ok := levels[level]
	if !ok {
		return nil, fmt.Errorf("no corpus for level %q", level)
	}
	fill := strings.NewReplacer("{{level}}", level, "{{Level}}", strings.ToUpper(level[:1])+level[1:],
		"{{owner}}", owner, "{{Owner}}", strings.ToUpper(owner[:1])+owner[1:], "{{implementation_targets}}", implementationTargets(dir))
	files := map[string][]byte{}
	for _, root := range []string{path.Join("templates", "corpus"), path.Join("templates", level)} {
		err := fs.WalkDir(templates, root, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			b, err := templates.ReadFile(p)
			files[fill.Replace(strings.TrimSuffix(strings.TrimPrefix(p, root+"/"), ".tmpl"))] = []byte(fill.Replace(string(b)))
			return err
		})
		if err != nil {
			return nil, err
		}
	}
	release, err := ParentRelease()
	if err != nil {
		return nil, err
	}
	charter, err := charterFile()
	if err != nil {
		return nil, err
	}
	agents, err := agentsFile(level)
	if err != nil {
		return nil, err
	}
	files[AgentsFile] = agents
	files["docs/constitutions/design.md"] = scrummaster.Constitution
	files["docs/corpus-charter.yaml"] = charter
	files["docs/external-pins.yaml"] = []byte("# The parent release this corpus's format follows (srd043 R3.3).\npins:\n  " + parentRepository + ": " + release + "\n")
	return files, nil
}

// charterFile is the corpus charter of a new corpus: the checks this
// corpus's own charter gates on, which a new corpus passes as written.
func charterFile() ([]byte, error) {
	var own struct {
		Checks []struct {
			Kind   string   `yaml:"kind"`
			Checks []string `yaml:"checks"`
		} `yaml:"checks"`
	}
	if err := yaml.Unmarshal(scrummaster.Charter, &own); err != nil {
		return nil, err
	}
	var b strings.Builder
	b.WriteString("# The checks the parent's specification-corpus audit gates this corpus on.\n" +
		"# The audit binds docs/corpus-charter.yaml relative to the directory it\n" +
		"# audits, so this file decides what gates here. tpm init wrote it with the\n" +
		"# checks a new corpus passes; add a check here as the corpus grows into it.\n" +
		"id: program-corpus-charter\ntitle: Program Specification Corpus Checks\nchecks:\n")
	for _, c := range own.Checks {
		if c.Kind != "spec_corpus" {
			continue
		}
		b.WriteString("  - id: program-spec-corpus\n    kind: spec_corpus\n    severity: error\n    checks:\n")
		for _, name := range c.Checks {
			b.WriteString("      - " + name + "\n")
		}
	}
	return []byte(b.String()), nil
}

// controlFiles are control/labels.md, control/policy.md, and
// control/states.md holding the built-in defaults (srd043 R1.3).
func controlFiles() map[string][]byte {
	return map[string][]byte{
		program.LabelsFile: []byte(LabelsDocument(program.DefaultLabels())),
		program.PolicyFile: []byte(PolicyDocument(InitPolicy())),
		beads.StatesFile:   []byte(StatesDocument(beads.Table)),
	}
}

// ProgramAudit is the corpus gate a program repository runs: tpm's own
// hierarchy gate, since a program repository has no mage.
const ProgramAudit = program.BacklogAudit

// InitPolicy is the policy tpm init writes: the built-in defaults, with
// tpm backlog check as the audit a docs/ merge runs and as the default
// definition of done, and a gate for the system's corpus release of that
// check and a peer read, since a release of the system needs one (srd015
// R1.2). Every command it names is one a program repository can run: it
// has no Go module, so go test ./... is not one.
func InitPolicy() program.Policy {
	p := program.Defaults()
	p.AuditCommand = ProgramAudit
	p.DefinitionOfDone["default"] = []program.Check{{Kind: "tests", Environment: "local", Suite: "hierarchy", Command: ProgramAudit}}
	p.DefinitionOfDone["corpus"] = []program.Check{
		{Kind: "tests", Environment: "local", Suite: "hierarchy", Command: ProgramAudit},
		{Kind: "peer-read"},
	}
	return p
}

// requireRepository refuses a directory that is not a git repository
// (srd043 R3.1).
func requireRepository(dir string) error {
	if out, err := exec.Command("git", "-C", dir, "rev-parse", "--git-dir").CombinedOutput(); err != nil {
		return fmt.Errorf("%s is not a git repository: %s", dir, strings.TrimSpace(string(out)))
	}
	return nil
}

// write lays files into dir. A file that exists is kept unless force
// replaces it, so a second run changes nothing; an orientation file whose
// managed block differs has the block refreshed and the rest kept, and one
// without a block has it appended after its text (srd043 R3.5). A dry run
// writes nothing.
func write(dir string, files map[string][]byte, force, dryRun bool) ([]Result, error) {
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	var out []Result
	for _, n := range names {
		p := filepath.Join(dir, filepath.FromSlash(n))
		r := Result{Path: n, Action: "created"}
		cur, err := os.ReadFile(p)
		switch {
		case errors.Is(err, fs.ErrNotExist):
		case err != nil:
			return out, err
		case bytes.Equal(cur, files[n]):
			r.Action, r.Reason = "kept", "unchanged"
		case force:
			r.Action = "overwritten"
		default:
			r.Action, r.Reason = "kept", "edited; --force replaces it"
			if merged, how, ok := refreshManaged(cur, files[n]); ok {
				switch {
				case bytes.Equal(merged, cur):
					r.Reason = "unchanged; text outside the tpm block is yours"
				case how == "appended":
					r.Action, r.Reason, files[n] = how, "tpm block added after your text", merged
				default:
					r.Action, r.Reason, files[n] = how, "tpm block replaced; text outside it kept", merged
				}
			}
		}
		out = append(out, r)
		if r.Action == "kept" || dryRun {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return out, err
		}
		if err := os.WriteFile(p, files[n], 0o644); err != nil {
			return out, err
		}
	}
	return out, nil
}
