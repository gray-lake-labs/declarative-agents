// Copyright (c) 2026 Gray Lake Labs
// SPDX-License-Identifier: MIT-0
package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/gray-lake-labs/declarative-agents/applications/tpm/internal/release"
)

// TestInstallScriptInstallsAVerifiedRelease builds tpm for this machine at a
// tag in a clone of this repository, as mage publish builds it, serves it as
// a release, and proves install.sh, run with no Go toolchain on PATH,
// installs a tpm whose version is the tag, and refuses a binary whose
// checksum SHA256SUMS does not list (eng03).
func TestInstallScriptInstallsAVerifiedRelease(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("install.sh serves Linux and macOS")
	}
	// The application lives inside the repository (GH-190): the test clones
	// the repository and publishes from the application's directory in it.
	top, err := exec.Command("git", "-C", mustAbs(t, ".."), "rev-parse", "--show-toplevel").Output()
	if err != nil {
		t.Fatal(err)
	}
	root := strings.TrimSpace(string(top))
	head, err := exec.Command("git", "-C", root, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	const tag = "v0.20990101.1"
	clone := filepath.Join(t.TempDir(), "monorepo")
	for _, args := range [][]string{
		{"clone", "-q", "--no-checkout", root, clone},
		{"-C", clone, "checkout", "-q", strings.TrimSpace(string(head))},
		{"-C", clone, "-c", "user.name=t", "-c", "user.email=t@example.invalid", "tag", "-a", tag, "-m", "Release " + tag},
	} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	dist := t.TempDir()
	appRoot := filepath.Join(clone, "applications", "tpm")
	p := release.Publisher{Root: appRoot, Git: release.ExecGit(appRoot), Run: release.ExecCommand}
	assets, err := p.Build(dist, []release.Platform{{OS: runtime.GOOS, Arch: runtime.GOARCH}})
	if err != nil {
		t.Fatal(err)
	}
	served := map[string]string{}
	for _, a := range assets {
		served["/download/"+tag+"/"+a.Name] = a.Path
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, ok := served[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, path)
	}))
	defer srv.Close()

	install := func(prefix string) (string, error) {
		cmd := exec.Command("sh", filepath.Join(root, "applications", "tpm", "install.sh"), tag)
		cmd.Env = []string{"PATH=/usr/bin:/bin", "HOME=" + t.TempDir(), "TPM_PREFIX=" + prefix, "TPM_RELEASE_URL=" + srv.URL}
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
	prefix := filepath.Join(t.TempDir(), "bin")
	out, err := install(prefix)
	if err != nil {
		t.Fatalf("install.sh: %v\n%s", err, out)
	}
	version, err := exec.Command(filepath.Join(prefix, "tpm"), "--version").Output()
	if err != nil || strings.TrimSpace(string(version)) != "tpm "+tag {
		t.Fatalf("the installed tpm reports %q, want tpm %s: %v", version, tag, err)
	}
	if !strings.Contains(out, "tpm install --runtime") {
		t.Errorf("install.sh did not name the next step:\n%s", out)
	}

	tampered := filepath.Join(t.TempDir(), "tampered")
	os.WriteFile(tampered, []byte("not tpm"), 0o755)
	served["/download/"+tag+"/"+assets[0].Name] = tampered
	other := filepath.Join(t.TempDir(), "bin")
	if out, err := install(other); err == nil || !strings.Contains(out, "refusing") {
		t.Fatalf("a tampered binary was installed: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(other, "tpm")); !os.IsNotExist(err) {
		t.Fatalf("a refused binary was written: %v", err)
	}
}
