package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSplitManaged(t *testing.T) {
	doc := "mine above\n" + BlockBegin + "\ntpm text\n" + BlockEnd + "\nmine below\n"
	before, block, after, ok := splitManaged([]byte(doc))
	if !ok || string(before) != "mine above\n" || string(after) != "mine below\n" ||
		string(block) != BlockBegin+"\ntpm text\n"+BlockEnd+"\n" {
		t.Fatalf("split: %q %q %q %v", before, block, after, ok)
	}
	if _, _, _, ok := splitManaged([]byte(BlockBegin + "\nno end\n")); ok {
		t.Fatal("a block without its end line split")
	}
	reworded := "<!-- tpm:begin: an older note -->\nold\n" + BlockEnd + "\n"
	if _, _, _, ok := splitManaged([]byte(reworded)); !ok {
		t.Fatal("a begin line with another note did not open the block")
	}
}

func TestRefreshManaged(t *testing.T) {
	cur := "notes\n" + BlockBegin + "\nold\n" + BlockEnd + "\nmore notes\n"
	fresh := BlockBegin + "\nnew\n" + BlockEnd + "\n"
	got, ok := refreshManaged([]byte(cur), []byte(fresh))
	if !ok || string(got) != "notes\n"+BlockBegin+"\nnew\n"+BlockEnd+"\nmore notes\n" {
		t.Fatalf("refresh: %q %v", got, ok)
	}
	if _, ok := refreshManaged([]byte("no block\n"), []byte(fresh)); ok {
		t.Fatal("a file without a block was refreshed")
	}
}

// olderBlock rewrites an orientation file's tpm text as an older tpm would
// have written it, and adds the owner's notes around the block.
func olderBlock(t *testing.T, path string) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, ok := splitManaged(b)
	if !ok {
		t.Fatalf("%s has no block", path)
	}
	old := "owner's note above\n" + BlockBegin + "\nan older tpm's text\n" + BlockEnd + "\nowner's note below\n"
	if err := os.WriteFile(path, []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestInitRefreshesBlockKeepsNotes(t *testing.T) {
	dir := repo(t)
	if _, err := Init(InitOptions{Dir: dir}); err != nil {
		t.Fatal(err)
	}
	readme := filepath.Join(dir, "README.md")
	olderBlock(t, readme)
	r, err := Init(InitOptions{Dir: dir})
	if err != nil || actions(r.Files)["README.md"] != "refreshed" {
		t.Fatalf("second init: %v, %v", actions(r.Files)["README.md"], err)
	}
	b, _ := os.ReadFile(readme)
	for _, want := range []string{"owner's note above\n" + BlockBegin, BlockEnd + "\nowner's note below\n", "# Program system repository"} {
		if !strings.Contains(string(b), want) {
			t.Errorf("refreshed README lacks %q:\n%s", want, b)
		}
	}
	if strings.Contains(string(b), "an older tpm's text") {
		t.Errorf("the older block survived:\n%s", b)
	}
	again, err := Init(InitOptions{Dir: dir})
	if err != nil || actions(again.Files)["README.md"] != "kept" {
		t.Fatalf("third init: %v, %v", actions(again.Files)["README.md"], err)
	}
}

func TestInitKeepsOrientationWithoutBlock(t *testing.T) {
	dir := repo(t)
	if _, err := Init(InitOptions{Dir: dir}); err != nil {
		t.Fatal(err)
	}
	readme := filepath.Join(dir, "README.md")
	os.WriteFile(readme, []byte("the owner's own README\n"), 0o644)
	r, err := Init(InitOptions{Dir: dir})
	if err != nil || actions(r.Files)["README.md"] != "kept" {
		t.Fatalf("init over a README without a block: %v, %v", actions(r.Files)["README.md"], err)
	}
	if b, _ := os.ReadFile(readme); string(b) != "the owner's own README\n" {
		t.Fatalf("README changed: %q", b)
	}
}
