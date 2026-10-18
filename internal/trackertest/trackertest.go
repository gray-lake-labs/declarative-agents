// Package trackertest runs one test against every tracker tpm serves: the
// GitLab fake and a beads tracker over real git repositories on local disk
// (srd042 R1.1). A test describes its program through Program and asserts
// through the tracker interface, never a backend's internals.
package trackertest

import (
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/gray-lake-labs/scrum-master/internal/beads"
	"github.com/gray-lake-labs/scrum-master/internal/beads/beadstest"
	"github.com/gray-lake-labs/scrum-master/internal/gitlab"
	"github.com/gray-lake-labs/scrum-master/internal/gitlab/gitlabtest"
)

// User is the person every Program's tracker acts as, the fake's Me.
const User = "me"

// Program is a tracker and the repositories it serves. Files and tags are
// declared first; Tracker builds them on its first call.
type Program interface {
	AddFile(project, ref, path string, content []byte)
	AddTag(project, tag string)
	Tracker() gitlab.Tracker
	// Writes counts the writes the tracker has made, so a test can prove a
	// dry run or a second run changed nothing.
	Writes() int
	// As is the program's tracker acting as another person, for a step
	// someone other than User takes, such as a peer read.
	As(user string) gitlab.Tracker
}

// Each runs test once per tracker, as subtests named gitlab and beads.
func Each(t *testing.T, test func(t *testing.T, p Program)) {
	t.Run("gitlab", func(t *testing.T) { test(t, fakeProgram{gitlabtest.New()}) })
	t.Run("beads", func(t *testing.T) { test(t, newBeadsProgram(t)) })
}

type fakeProgram struct{ *gitlabtest.Fake }

func (f fakeProgram) Tracker() gitlab.Tracker { return f.Fake }
func (f fakeProgram) Writes() int             { return f.Fake.Writes }
func (f fakeProgram) As(user string) gitlab.Tracker {
	return fakeAs{f.Fake, user}
}

// fakeAs is the fake acting as another person: it is who CurrentUser
// names and who authors the notes it posts.
type fakeAs struct {
	*gitlabtest.Fake
	user string
}

func (f fakeAs) CurrentUser() (gitlab.User, error) { return gitlab.User{Username: f.user}, nil }

func (f fakeAs) CreateNote(project string, iid int, body string) error {
	me := f.Fake.Me
	f.Fake.Me = gitlab.User{Username: f.user}
	defer func() { f.Fake.Me = me }()
	return f.Fake.CreateNote(project, iid, body)
}

// beadsProgram builds each project as a code repository named for the
// project's last path segment, siblings in one directory, which is how the
// beads tracker finds a project's checkout.
type beadsProgram struct {
	t       *testing.T
	root    string
	remote  string
	files   map[string]map[string]map[string][]byte // project, ref, path
	tags    map[string][]string
	tracker *beads.Tracker
	work    string
	as      map[string]*beads.Tracker
}

func newBeadsProgram(t *testing.T) *beadsProgram {
	root := t.TempDir()
	return &beadsProgram{t: t, root: root, remote: beadstest.TrackerRepository(t, root),
		files: map[string]map[string]map[string][]byte{}, tags: map[string][]string{}}
}

func (b *beadsProgram) AddFile(project, ref, p string, content []byte) {
	if b.tracker != nil {
		b.t.Fatal("trackertest: AddFile after Tracker")
	}
	if b.files[project] == nil {
		b.files[project] = map[string]map[string][]byte{}
	}
	if b.files[project][ref] == nil {
		b.files[project][ref] = map[string][]byte{}
	}
	b.files[project][ref][p] = content
}

func (b *beadsProgram) AddTag(project, tag string) {
	if b.tracker != nil {
		b.t.Fatal("trackertest: AddTag after Tracker")
	}
	b.tags[project] = append(b.tags[project], tag)
}

func (b *beadsProgram) Tracker() gitlab.Tracker {
	if b.tracker != nil {
		return b.tracker
	}
	t := b.t
	t.Helper()
	projects := map[string]bool{}
	for p := range b.files {
		projects[p] = true
	}
	for p := range b.tags {
		projects[p] = true
	}
	work := beadstest.CodeRepository(t, b.root, "work", User)
	for project := range projects {
		b.build(project)
	}
	clone := filepath.Join(b.root, User+"-tracker")
	beadstest.Clone(t, b.remote, clone, User)
	b.tracker, b.work = beads.NewTracker(clone, User, work), work
	return b.tracker
}

// As gives another person their own clone of the tracker repository.
func (b *beadsProgram) As(user string) gitlab.Tracker {
	b.Tracker()
	if tr, ok := b.as[user]; ok {
		return tr
	}
	clone := filepath.Join(b.root, user+"-tracker")
	beadstest.Clone(b.t, b.remote, clone, user)
	if b.as == nil {
		b.as = map[string]*beads.Tracker{}
	}
	b.as[user] = beads.NewTracker(clone, user, b.work)
	return b.as[user]
}

// build makes one project's repository: main holds the files declared at
// main; every other ref is a branch from main holding its own files too,
// and a ref that is also a tag becomes that tag instead of a branch.
func (b *beadsProgram) build(project string) {
	t := b.t
	dir := beadstest.CodeRepository(t, b.root, path.Base(project), User)
	commit := func(files map[string][]byte, msg string) {
		for p, content := range files {
			full := filepath.Join(dir, p)
			if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(full, content, 0o644); err != nil {
				t.Fatal(err)
			}
		}
		beadstest.Git(t, dir, "add", "-A")
		beadstest.Git(t, dir, "commit", "-q", "--allow-empty", "-m", msg)
	}
	if files := b.files[project]["main"]; len(files) > 0 {
		commit(files, "main files")
		beadstest.Git(t, dir, "push", "-q", "origin", "main")
	}
	var refs []string
	for ref := range b.files[project] {
		if ref != "main" {
			refs = append(refs, ref)
		}
	}
	for _, tag := range b.tags[project] {
		if !slices.Contains(refs, tag) {
			refs = append(refs, tag)
		}
	}
	slices.Sort(refs)
	for _, ref := range refs {
		beadstest.Git(t, dir, "checkout", "-q", "-b", "build-"+ref, "main")
		commit(b.files[project][ref], ref+" files")
		if slices.Contains(b.tags[project], ref) {
			beadstest.Git(t, dir, "tag", ref)
			beadstest.Git(t, dir, "push", "-q", "origin", "refs/tags/"+ref)
		} else {
			beadstest.Git(t, dir, "branch", "-q", "-m", ref)
			beadstest.Git(t, dir, "push", "-q", "origin", ref)
		}
		beadstest.Git(t, dir, "checkout", "-q", "main")
	}
}

// Writes counts the commits the tracker repository gained after its seed.
func (b *beadsProgram) Writes() int {
	out, err := exec.Command("git", "--git-dir", b.remote, "rev-list", "--count", "main").Output()
	if err != nil {
		b.t.Fatal(err)
	}
	n, _ := strconv.Atoi(strings.TrimSpace(string(out)))
	return n - 1
}
