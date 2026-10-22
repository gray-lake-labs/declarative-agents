package release

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// fakeImageTools is docker, cosign, sh, and gh for an image publish: it
// records each command, and answers as the registry, the signatures, and
// the release's notes say.
type fakeImageTools struct {
	fakeTools
	ran      []string
	pushed   string // the digest at the tag in the registry; empty is none
	signed   bool
	notes    string
	noCosign bool
}

func (f *fakeImageTools) run(dir string, env []string, name string, args ...string) (string, error) {
	line := name + " " + strings.Join(args, " ")
	f.ran = append(f.ran, line)
	switch {
	case name == "cosign" && f.noCosign:
		return "", errors.New("cosign: not found")
	case name == "cosign" && args[0] == "version", name == "sh":
		return "", nil
	case name == "cosign" && args[0] == "verify":
		if !f.signed {
			return "", errors.New("no signatures found")
		}
		return "", nil
	case name == "cosign" && args[0] == "sign":
		f.signed = true
		return "", nil
	case name == "docker" && args[1] == "imagetools":
		if f.pushed == "" {
			return "", errors.New("not found")
		}
		return f.pushed, nil
	case name == "docker" && args[1] == "build":
		meta := args[slices.Index(args, "--metadata-file")+1]
		f.pushed = "sha256:abc"
		return "", os.WriteFile(meta, []byte(`{"containerimage.digest":"sha256:abc"}`), 0o644)
	case name == "docker":
		return "", nil
	case name == "gh" && len(args) > 1 && args[1] == "view" && slices.Contains(args, "body"):
		return f.notes, nil
	case name == "gh" && len(args) > 1 && args[1] == "edit":
		b, err := os.ReadFile(args[slices.Index(args, "--notes-file")+1])
		f.notes = string(b)
		return "", err
	}
	return f.fakeTools.run(dir, env, name, args...)
}

func (f *fakeImageTools) count(prefix string) int {
	n := 0
	for _, l := range f.ran {
		if strings.HasPrefix(l, prefix) {
			n++
		}
	}
	return n
}

// imageRelease is a pushed tag whose repository holds cosign.pub, with the
// signing key in the environment.
func imageRelease(t *testing.T) (Publisher, *fakeImageTools) {
	t.Helper()
	dir, git := tagged(t, "v0.20261015.1")
	os.WriteFile(filepath.Join(dir, CosignKeyFile), []byte("public key"), 0o644)
	t.Setenv(CosignKeyEnv, "private key")
	f := &fakeImageTools{fakeTools: fakeTools{login: "org-owner"}, notes: "Release v0.20261015.1"}
	p := publisher(dir, git, &f.fakeTools, io.Discard)
	p.Run, p.ImageRepo = f.run, "ghcr.io/org/tpm"
	return p, f
}

func TestPublishPushesSignsAndListsTheImage(t *testing.T) {
	p, f := imageRelease(t)
	if err := p.Publish(); err != nil {
		t.Fatalf("%v\n%s", err, strings.Join(f.ran, "\n"))
	}
	i := slices.IndexFunc(f.ran, func(l string) bool { return strings.HasPrefix(l, "docker buildx build") })
	if i < 0 {
		t.Fatalf("no image built:\n%s", strings.Join(f.ran, "\n"))
	}
	build := f.ran[i]
	for _, want := range []string{"--platform linux/amd64,linux/arm64", "--sbom=true", "--provenance=mode=min", "--tag ghcr.io/org/tpm:v0.20261015.1", "--tag ghcr.io/org/tpm:latest", "--push", "--build-arg VERSION=v0.20261015.1"} {
		if !strings.Contains(build, want) {
			t.Errorf("the build lacks %q: %s", want, build)
		}
	}
	if !f.signed || f.count("cosign sign --yes --key env://"+CosignKeyEnv+" ghcr.io/org/tpm@sha256:abc") != 1 {
		t.Fatalf("the digest was not signed with the organization's key:\n%s", strings.Join(f.ran, "\n"))
	}
	if !strings.Contains(f.notes, "Image: ghcr.io/org/tpm@sha256:abc") {
		t.Fatalf("the release notes do not list the digest: %q", f.notes)
	}
	if f.count("sh -c") != 1 {
		t.Fatalf("ghcr.io was not logged in to once:\n%s", strings.Join(f.ran, "\n"))
	}
}

func TestPublishKeepsAPushedImageAndSignsItOnce(t *testing.T) {
	p, f := imageRelease(t)
	f.pushed = "sha256:abc"
	if err := p.PublishImage("v0.20261015.1"); err != nil {
		t.Fatal(err)
	}
	if f.count("docker buildx build") != 0 || !f.signed {
		t.Fatalf("a pushed, unsigned image was rebuilt or left unsigned:\n%s", strings.Join(f.ran, "\n"))
	}
	f.ran = nil
	if err := p.PublishImage("v0.20261015.1"); err != nil {
		t.Fatal(err)
	}
	if f.count("cosign sign") != 0 || f.count("gh release edit") != 0 {
		t.Fatalf("a signed, listed image was signed or listed again:\n%s", strings.Join(f.ran, "\n"))
	}
}

func TestImageNeedsItsTools(t *testing.T) {
	p, f := imageRelease(t)
	f.noCosign = true
	os.Remove(filepath.Join(p.Root, CosignKeyFile))
	t.Setenv(CosignKeyEnv, "")
	err := p.Check()
	for _, want := range []string{"cosign", CosignKeyFile, CosignKeyEnv} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("the preflight did not name %s: %v", want, err)
		}
	}
	if err := p.PublishImage("v0.20261015.1"); err == nil || f.count("docker buildx build") != 0 {
		t.Fatalf("an image was built without its tools: %v", err)
	}
}

func TestImageDryRunRunsNothing(t *testing.T) {
	p, f := imageRelease(t)
	p.DryRun = true
	if err := p.Publish(); err != nil {
		t.Fatal(err)
	}
	if f.count("docker") != 0 || f.count("cosign") != 0 || f.count("sh") != 0 {
		t.Fatalf("a dry run ran:\n%s", strings.Join(f.ran, "\n"))
	}
}
