package beads

import (
	"bytes"
	"os/exec"
	"slices"
	"strings"
	"testing"
	"time"
)

// TestFilesLayoutMergesThroughGit creates an issue on two clones that
// cannot see each other, and changes one issue on both: after the later
// clone reconciles, the two new issues have distinct IIDs, the later one
// renumbered, and the shared issue carries both clones' changes (eng04).
func TestFilesLayoutMergesThroughGit(t *testing.T) {
	if !fixtureFiles() {
		t.Skip("the files layout's merge")
	}
	f := newFixture(t, ready("sm-1"))
	alice, bob := f.clone("alice"), f.clone("bob")
	bob.sleep = func(time.Duration) {}
	if _, _, err := bob.Issues(); err != nil {
		t.Fatal(err)
	}
	f.offline(bob)
	theirs, err := bob.Create(Issue{Repository: "g/po-1", Title: "bob's"})
	if err != nil {
		t.Fatal(err)
	}
	if err := bob.Comment("sm-1", "bob was here"); err != nil {
		t.Fatal(err)
	}
	ours, err := alice.Create(Issue{Repository: "g/po-1", Title: "alice's"})
	if err != nil {
		t.Fatal(err)
	}
	if err := alice.Comment("sm-1", "alice was here"); err != nil {
		t.Fatal(err)
	}
	if ours.IID != 1 || theirs.IID != 1 {
		t.Fatalf("before the merge both clones picked IID 1: alice %d, bob %d", ours.IID, theirs.IID)
	}
	f.online(bob)
	if _, err := bob.Reconcile(); err != nil {
		t.Fatal(err)
	}
	got := map[string]Issue{}
	for _, i := range f.remoteIssues() {
		got[i.Title] = i
	}
	if got["alice's"].IID != 1 || got["bob's"].IID != 2 {
		t.Fatalf("IIDs after the merge: alice's %d, bob's %d; want 1 and 2", got["alice's"].IID, got["bob's"].IID)
	}
	var texts []string
	for _, c := range got["work sm-1"].Comments {
		texts = append(texts, c.Text)
	}
	if !slices.Contains(texts, "alice was here") || !slices.Contains(texts, "bob was here") {
		t.Fatalf("sm-1's comments after the merge: %q", texts)
	}
	// Every issue is one file with one line.
	out, _ := exec.Command("git", "--git-dir", f.remote, "ls-tree", "--name-only", "main", IssuesDir+"/").Output()
	if n := len(strings.Fields(string(out))); n != 3 {
		t.Fatalf("%d issue files, want 3", n)
	}
}

// TestMigrateAndExport moves a JSONL tracker to one file per issue in one
// commit, keeps every issue, refuses nothing on a second run, and exports
// bd's JSONL back.
func TestMigrateAndExport(t *testing.T) {
	t.Setenv("BEADS_FIXTURE_LAYOUT", "jsonl")
	f := newFixture(t, ready("sm-1"), ready("sm-2"))
	alice := f.clone("alice")
	if err := alice.Comment("sm-2", "before the move"); err != nil {
		t.Fatal(err)
	}
	before := f.remoteIssues()
	r, err := alice.Migrate()
	if err != nil || r.Issues != 2 || r.Already {
		t.Fatalf("migrate = %+v, %v", r, err)
	}
	if !alice.files() {
		t.Fatal("the config does not say layout: files")
	}
	t.Setenv("BEADS_FIXTURE_LAYOUT", "files")
	after := f.remoteIssues()
	if len(after) != len(before) {
		t.Fatalf("%d issues after the move, %d before", len(after), len(before))
	}
	if out, _ := exec.Command("git", "--git-dir", f.remote, "show", "main:"+IssuesFile).CombinedOutput(); !strings.Contains(string(out), "does not exist") && !strings.Contains(string(out), "exists on disk, but not in") {
		t.Fatalf("the JSONL file is still in the tracker: %s", out)
	}
	again, err := alice.Migrate()
	if err != nil || !again.Already {
		t.Fatalf("second migrate = %+v, %v", again, err)
	}
	b, err := alice.Export()
	if err != nil {
		t.Fatal(err)
	}
	want, _ := encodeJSONL(before)
	if !bytes.Equal(b, want) {
		t.Fatalf("export:\n%s\nwant:\n%s", b, want)
	}
	bob := f.clone("bob")
	if err := bob.Comment("sm-1", "after the move"); err != nil {
		t.Fatal(err)
	}
	if i := f.remoteIssue("sm-1"); len(i.Comments) != 1 {
		t.Fatalf("a write after the move: %+v", i)
	}
}
