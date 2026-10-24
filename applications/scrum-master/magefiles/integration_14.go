// Copyright (c) 2026 Gray Lake Labs
// SPDX-License-Identifier: MIT-0
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/release"
)

// Image builds the release for Linux on this machine's architecture, builds the release image from it as mage publish does, and proves the image's agent is the release's, the profiles are unpacked at the release, it carries no Go toolchain or mage and runs as a non-root user, the init profile runs over a mounted checkout with only the image's agent, and the profiles it vendored validate against the vendored core root (test-rel04.3-distribution, GH-34). It needs docker with buildx, and no rig.
func (Integration) Image() error {
	root, err := repositoryRoot()
	if err != nil {
		return err
	}
	tmp, err := os.MkdirTemp("", "scrum-master-image-")
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

	// A CI job's checkout: an empty repository the image's user can write.
	if err := os.MkdirAll(repo, 0o777); err != nil {
		return err
	}
	if out, err := exec.Command("git", "init", "-q", "-b", "main", repo).CombinedOutput(); err != nil {
		return fmt.Errorf("git init: %w: %s", err, out)
	}
	os.Chmod(repo, 0o777)
	inImage := func(script string) (string, error) {
		out, err := exec.Command("docker", "run", "--rm", "-v", repo+":/work", image, "sh", "-c", script).CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}

	c := &check{}
	want, err := fileSHA256(filepath.Join(dist, release.AssetName("agent", platform)))
	if err != nil {
		return err
	}
	got, err := inImage(`sha256sum "$(command -v agent)" | cut -d" " -f1`)
	c.expect(err == nil && got == want, "the image's agent is %q, the release's %q: %v", got, want, err)
	rel, err := inImage(`cat "$SCRUM_MASTER_HOME/release"; test -f "$SCRUM_MASTER_HOME/agents/init/profile.yaml" && echo init`)
	c.expect(err == nil && rel == "dev\ninit", "the image's profiles: %q, %v", rel, err)
	bare, err := inImage(`id -u; command -v go || echo no-go; command -v mage || echo no-mage`)
	c.expect(err == nil && bare == "1000\nno-go\nno-mage", "user and toolchain: %q, %v", bare, err)

	initRun, err := inImage(`agent --profile "$SCRUM_MASTER_HOME/agents/init/profile.yaml" --core-root "$SCRUM_MASTER_HOME/agent-core" --directory /work --output /tmp/out 2>&1 | tail -3`)
	c.expect(err == nil && strings.Contains(initRun, "final machine state: Done"), "the init profile in the image: %q, %v", initRun, err)
	valid, err := inImage(`agent --validate-config --profile /work/.scrum-master/agents/pop/profile.yaml --core-root /work/.scrum-master/agent-core`)
	c.expect(err == nil && strings.Contains(valid, "config valid"), "a vendored profile in the image: %q, %v", valid, err)
	return c.err("image")
}

// fileSHA256 is a file's SHA-256, as sha256sum prints it.
func fileSHA256(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}
