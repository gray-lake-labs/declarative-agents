// Package routing opens the tracking systems a program configures in
// control/trackers.md, one per repository (srd042), and routes each
// project to its own. With no such file, the one system TPM_TRACKER names
// serves every project, as before routing existed.
package routing

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/gray-lake-labs/scrum-master/internal/beads"
	"github.com/gray-lake-labs/scrum-master/internal/corpus"
	"github.com/gray-lake-labs/scrum-master/internal/github"
	"github.com/gray-lake-labs/scrum-master/internal/tracker"
)

// File is where a program's tracking systems are configured, in its
// system repository.
const File = "control/trackers.md"

// The tracking systems a repository may use.
const (
	Beads  = "bd"
	GitHub = "gh"
	GitLab = "glab"
)

// System is one tracking system: its name and, for bd, the tracker
// repository it reads.
type System struct {
	Name              string `json:"system"`
	TrackerRepository string `json:"tracker_repository,omitempty"`
	// Dir is the command's checkout, and Paths the projects and groups
	// routed to the system, empty for the default.
	Dir   string   `json:"-"`
	Paths []string `json:"-"`
}

// Config is control/trackers.md: the program's default system and the
// projects or groups that use another.
type Config struct {
	Default   System            `json:"default"`
	Overrides map[string]string `json:"overrides,omitempty"`
}

type configFile struct {
	Version int `yaml:"version"`
	Default struct {
		System            string `yaml:"system"`
		TrackerRepository string `yaml:"tracker_repository"`
	} `yaml:"default"`
	Overrides []string `yaml:"overrides"`
}

// Parse reads a trackers document. A path may be named once: a project
// mapped to two systems is refused, so project#iid always names one issue.
func Parse(data []byte) (Config, error) {
	var f configFile
	if err := corpus.DecodeConfig(data, &f); err != nil {
		return Config{}, fmt.Errorf("%s: %w", File, err)
	}
	c := Config{Default: System{Name: strings.TrimSpace(f.Default.System), TrackerRepository: strings.TrimSpace(f.Default.TrackerRepository)}}
	if err := known(c.Default.Name); err != nil {
		return Config{}, fmt.Errorf("%s default: %w", File, err)
	}
	for _, line := range f.Overrides {
		path, name, ok := strings.Cut(line, ":")
		path, name = strings.TrimSpace(path), strings.TrimSpace(name)
		if !ok || path == "" {
			return Config{}, fmt.Errorf("%s: override %q is not path: system", File, line)
		}
		if err := known(name); err != nil {
			return Config{}, fmt.Errorf("%s: %s: %w", File, path, err)
		}
		if name == Beads {
			return Config{}, fmt.Errorf("%s: %s: an override names gh or glab; bd serves through the default's tracker repository", File, path)
		}
		if c.Overrides == nil {
			c.Overrides = map[string]string{}
		}
		if prev, dup := c.Overrides[path]; dup {
			return Config{}, fmt.Errorf("%s: %s is mapped to both %s and %s; a repository has one tracking system", File, path, prev, name)
		}
		c.Overrides[path] = name
	}
	return c, nil
}

func known(name string) error {
	if !slices.Contains([]string{Beads, GitHub, GitLab}, name) {
		return fmt.Errorf("tracking system %q is not bd, gh, or glab", name)
	}
	return nil
}

// Find is the trackers document for the checkout at dir: the checkout's
// own control/trackers.md, else the one sibling checkout that holds one, as
// repositories sit side by side under their owner and only the system
// repository has control files. It returns "" when there is none, and
// refuses more than one sibling.
func Find(dir string) (string, error) {
	root := dir
	if out, err := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel").Output(); err == nil {
		root = strings.TrimSpace(string(out))
	}
	if own := filepath.Join(root, File); exists(own) {
		return own, nil
	}
	siblings, _ := filepath.Glob(filepath.Join(filepath.Dir(root), "*", File))
	switch len(siblings) {
	case 0:
		return "", nil
	case 1:
		return siblings[0], nil
	}
	return "", fmt.Errorf("several sibling checkouts hold %s (%s); run tpm from the system repository's checkout", File, strings.Join(siblings, ", "))
}

func exists(p string) bool { _, err := os.Stat(p); return err == nil }

// Opener opens one tracking system. bd is built in; gh and glab register
// theirs.
type Opener func(System) (tracker.Tracker, error)

var openers = map[string]Opener{
	Beads: func(s System) (tracker.Tracker, error) {
		if s.TrackerRepository == "" {
			return nil, errors.New("bd needs its tracker repository: set tracker_repository in " + File + " or TPM_TRACKER")
		}
		return beads.OpenRepository(s.TrackerRepository)
	},
	GitHub: func(s System) (tracker.Tracker, error) {
		return github.New(s.Dir, github.CLI{}, s.Paths), nil
	},
}

// cli is the command each hosted system drives, which doctor checks is
// installed and logged in.
var cli = map[string]string{GitHub: "gh", GitLab: "glab"}

// Register makes a tracking system available by name.
func Register(name string, open Opener) { openers[name] = open }

func open(s System) (tracker.Tracker, error) {
	o, ok := openers[s.Name]
	if !ok {
		return nil, fmt.Errorf("tpm has no %s tracking system yet; use bd", s.Name)
	}
	return o(s)
}

// Override is what TPM_TRACKER says about the default system: a git URL or
// path is bd's tracker repository, and gh or glab names that system. It
// overrides the file's default, never its overrides.
func Override(env string) (System, bool) {
	switch env = strings.TrimSpace(env); env {
	case "":
		return System{}, false
	case GitHub, GitLab, Beads:
		return System{Name: env}, true
	}
	return System{Name: Beads, TrackerRepository: env}, true
}

// Open opens the tracking systems for a command run in dir: with no
// trackers document, the one TPM_TRACKER names, exactly as before routing;
// with one, its default (TPM_TRACKER overriding it) and each override,
// behind a router when any override exists.
func Open(dir string) (tracker.Tracker, error) {
	path, err := Find(dir)
	if err != nil {
		return nil, err
	}
	env, set := Override(os.Getenv("TPM_TRACKER"))
	if path == "" {
		if !set {
			return nil, errors.New("no tracker: set TPM_TRACKER to the tracker repository, or write " + File + " in the system repository (eng04-beads-sync)")
		}
		env.Dir = dir
		return open(env)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	c, err := Parse(b)
	if err != nil {
		return nil, err
	}
	if set {
		if env.Name == Beads && env.TrackerRepository == "" {
			env.TrackerRepository = c.Default.TrackerRepository
		}
		c.Default = env
	}
	c.Default.Dir = dir
	def, err := open(c.Default)
	if err != nil {
		return nil, err
	}
	if len(c.Overrides) == 0 {
		return def, nil
	}
	r := &tracker.Router{Default: def, Systems: map[string]tracker.Tracker{}}
	paths := map[string][]string{}
	for p, name := range c.Overrides {
		paths[name] = append(paths[name], p)
	}
	opened := map[string]tracker.Tracker{}
	for p, name := range c.Overrides {
		if opened[name] == nil {
			slices.Sort(paths[name])
			if opened[name], err = open(System{Name: name, Dir: dir, Paths: paths[name]}); err != nil {
				return nil, fmt.Errorf("%s: %w", p, err)
			}
		}
		r.Systems[p] = opened[name]
	}
	return r, nil
}

// Document is c as control/trackers.md holds it.
func Document(c Config) []byte {
	var b strings.Builder
	b.WriteString("---\nid: trackers\ntitle: Tracking systems\n---\n\n")
	b.WriteString("<!-- field: version -->\n## Version\n\n1\n\n")
	b.WriteString("<!-- field: default -->\n## Default\n\n<!-- field: system -->\n### System\n\n" + c.Default.Name + "\n")
	if c.Default.TrackerRepository != "" {
		b.WriteString("\n<!-- field: tracker_repository -->\n### Tracker repository\n\n" + c.Default.TrackerRepository + "\n")
	}
	if len(c.Overrides) > 0 {
		b.WriteString("\n<!-- field: overrides | form: \"tree\" -->\n## Overrides\n\n")
		paths := make([]string, 0, len(c.Overrides))
		for p := range c.Overrides {
			paths = append(paths, p)
		}
		slices.Sort(paths)
		for _, p := range paths {
			b.WriteString("- " + p + ": " + c.Overrides[p] + "\n")
		}
	}
	return []byte(b.String())
}

// Report is what tpm doctor says about a program's tracking systems: the
// document it read, the default and the overrides, and each problem found.
type Report struct {
	File      string            `json:"file"`
	Default   System            `json:"default"`
	Overrides map[string]string `json:"overrides,omitempty"`
	Problems  []string          `json:"problems,omitempty"`
}

// Check reads the trackers document for dir and reports each system's
// reach: bd's tracker repository answers git ls-remote, and gh and glab are
// systems this tpm has. It reports nothing when there is no document.
func Check(dir string) (*Report, error) {
	path, err := Find(dir)
	if err != nil || path == "" {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	c, err := Parse(b)
	if err != nil {
		return &Report{File: path, Problems: []string{err.Error()}}, nil
	}
	r := &Report{File: path, Default: c.Default, Overrides: c.Overrides}
	names := []string{c.Default.Name}
	for _, n := range c.Overrides {
		if !slices.Contains(names, n) {
			names = append(names, n)
		}
	}
	for _, n := range names {
		if _, ok := openers[n]; !ok {
			r.Problems = append(r.Problems, fmt.Sprintf("tpm has no %s tracking system yet", n))
			continue
		}
		if bin := cli[n]; bin != "" {
			if _, err := exec.LookPath(bin); err != nil {
				r.Problems = append(r.Problems, fmt.Sprintf("%s serves through the %s CLI, which is not installed", n, bin))
			} else if out, err := exec.Command(bin, "auth", "status").CombinedOutput(); err != nil {
				r.Problems = append(r.Problems, fmt.Sprintf("%s is not logged in: run %s auth login (%s)", bin, bin, firstLine(string(out))))
			}
		}
	}
	if c.Default.Name == Beads {
		if out, err := exec.Command("git", "ls-remote", "-q", c.Default.TrackerRepository, "HEAD").CombinedOutput(); err != nil {
			r.Problems = append(r.Problems, fmt.Sprintf("bd's tracker repository %s does not answer: %s", c.Default.TrackerRepository, strings.TrimSpace(string(out))))
		}
	}
	return r, nil
}

func firstLine(s string) string {
	s, _, _ = strings.Cut(strings.TrimSpace(s), "\n")
	return s
}
