// Copyright (c) 2026 Gray Lake Labs
// SPDX-License-Identifier: MIT-0
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gray-lake-labs/declarative-agents/applications/tpm/internal/release"
	tpmruntime "github.com/gray-lake-labs/declarative-agents/applications/tpm/internal/runtime"
	"github.com/gray-lake-labs/declarative-agents/applications/tpm/internal/scaffold"
)

// Image builds tpm for Linux on this machine's architecture, builds the release image from it as mage publish does, and proves the image's tpm is that binary, its agent is the pinned release's, it carries no Go toolchain or mage and runs as a non-root user, tpm backlog check and tpm doctor --dry-run run over a mounted checkout with doctor finding the runtime and repairing with the image's spec tool, and the agent validates a profile against the image's source root (test-rel04.3-distribution). It needs docker with buildx and network access, and no rig.
func (Integration) Image() error {
	root, err := repositoryRoot()
	if err != nil {
		return err
	}
	tmp, err := os.MkdirTemp("", "tpm-image-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	if tmp, err = filepath.EvalSymlinks(tmp); err != nil {
		return err
	}
	dist, repo := filepath.Join(tmp, "dist"), filepath.Join(tmp, "system")
	platform := release.Platform{OS: "linux", Arch: runtime.GOARCH}
	p := release.Publisher{Root: root, Run: release.ExecCommand, Out: os.Stdout}
	if _, err := p.Build(dist, []release.Platform{platform}); err != nil {
		return err
	}
	image := fmt.Sprintf("scrum-master-integration:%d", os.Getpid())
	if _, err := p.BuildImage(release.ImageBuild{Dist: dist, Platforms: []release.Platform{platform}, Refs: []string{image}, Version: "integration", Revision: "integration"}); err != nil {
		return fmt.Errorf("build the image: %w", err)
	}
	defer exec.Command("docker", "image", "rm", "-f", image).Run()

	// A program repository on this machine, as a CI job's checkout.
	bin := filepath.Join(tmp, "tpm")
	if err := runStreaming(root, "go", "build", "-o", bin, "./cmd/tpm"); err != nil {
		return err
	}
	if err := os.MkdirAll(repo, 0o755); err != nil {
		return err
	}
	for _, args := range [][]string{
		{"git", "init", "-q", "-b", "main"},
		{bin, "init", "--dir", repo},
		{"git", "add", "-A"},
		{"git", "-c", "user.name=ci", "-c", "user.email=ci@example.invalid", "commit", "-q", "-m", "init"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("%v: %w: %s", args, err, out)
		}
	}
	inImage := func(script string) (string, error) {
		out, err := exec.Command("docker", "run", "--rm", "-v", repo+":/work", image, "sh", "-c", script).CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}

	c := &check{}
	version, err := exec.Command("go", "version", "-m", filepath.Join(dist, release.AssetName("tpm", platform))).Output()
	if err != nil {
		return err
	}
	var want string
	for _, line := range strings.Split(string(version), "\n") {
		if f := strings.Fields(line); len(f) >= 3 && f[0] == "mod" {
			want = "tpm " + f[2]
		}
	}
	got, err := inImage("tpm --version")
	c.expect(err == nil && got == want, "the image's tpm reports %q, the release binary %q: %v", got, want, err)

	pin := tpmruntime.DefaultPin()
	asset, _ := tpmruntime.HostAsset("agent", "linux", runtime.GOARCH)
	sums, err := inImage(`sha256sum "$(readlink -f "$(command -v agent)")" | cut -d" " -f1; grep " ` + asset + `$" "$XDG_CACHE_HOME/scrum-master/release/` + pin + `/SHA256SUMS" | cut -d" " -f1`)
	lines := strings.Fields(sums)
	c.expect(err == nil && len(lines) == 2 && lines[0] == lines[1], "the image's agent is not the %s %s: %q, %v", pin, asset, sums, err)

	bare, err := inImage(`id -u; command -v go || echo no-go; command -v mage || echo no-mage`)
	c.expect(err == nil && bare == "1000\nno-go\nno-mage", "user and toolchain: %q, %v", bare, err)

	backlog, err := inImage("tpm backlog check")
	c.expect(err == nil && strings.Contains(backlog, "pass"), "tpm backlog check over the checkout: %q, %v", backlog, err)

	out, err := inImage("tpm doctor --dry-run --json")
	var doctor scaffold.DoctorResult
	if err == nil {
		err = json.Unmarshal([]byte(out), &doctor)
	}
	c.expect(err == nil && doctor.Runtime.Release == pin && doctor.Runtime.Agent != "" && doctor.Repair.Skipped == "", "tpm doctor in the image: runtime %+v, repair %+v, %v\n%s", doctor.Runtime, doctor.Repair, err, out)

	profile := "applications/catalog/agents/specification-critic/corpus-audit-profile.yaml"
	src := "$XDG_CACHE_HOME/scrum-master/release/" + pin + "/src"
	valid, err := inImage(`agent --validate-config --profile "` + src + "/" + profile + `" --core-root "` + src + `/agent-core"`)
	c.expect(err == nil && strings.Contains(valid, "config valid"), "agent --validate-config in the image: %q, %v", valid, err)

	return c.err("image")
}
