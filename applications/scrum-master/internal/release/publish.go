// Copyright (c) 2026 Gray Lake Labs
// SPDX-License-Identifier: MIT-0
package release

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/runtime"
)

// Platform is one operating system and architecture a release binary is
// built for.
type Platform struct{ OS, Arch string }

// Platforms are the release binaries' platforms (eng03).
var Platforms = []Platform{
	{"linux", "amd64"}, {"linux", "arm64"},
	{"darwin", "amd64"}, {"darwin", "arm64"},
	{"windows", "amd64"}, {"windows", "arm64"},
}

// AssetName is a release binary's name, the convention the declarative-agents
// releases follow: name_os_arch, with .exe on windows.
func AssetName(name string, p Platform) string {
	n := name + "_" + p.OS + "_" + p.Arch
	if p.OS == "windows" {
		n += ".exe"
	}
	return n
}

// SumsFile lists every release binary's SHA-256.
const SumsFile = "SHA256SUMS"

// Asset is one file on a release: a binary or SHA256SUMS.
type Asset struct {
	Name, Path, Sum string
}

// Command runs name with args in dir, with env added to the process's
// environment, and returns its standard output.
type Command func(dir string, env []string, name string, args ...string) (string, error)

// ExecCommand is Command on this machine.
func ExecCommand(dir string, env []string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(string(out)), nil
}

// Publisher publishes the scrum-master tag at HEAD (eng03): the release
// binaries and SHA256SUMS on the tag's GitHub release, then the image. A part
// already published is checked and kept, so a rerun completes a release a
// failed upload left partial.
type Publisher struct {
	Root    string
	Git     Git
	Run     Command
	Out     io.Writer
	Repo    string // the GitHub repository, owner/name
	Account string // the gh account releases are made as
	DryRun  bool
	// ImageRepo is where the tag's image is pushed, such as
	// ghcr.io/owner/name; empty publishes no image.
	ImageRepo string
}

// VersionPackage is where a release's version is stamped in tpm; the agent's
// is agent-core's internal/version, under the module path its go.mod
// declares. Both are nested modules of the repository, so their build info
// never carries its tag (GH-18).
const VersionPackage = "github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/version"

// agentVersionPackage is agent-core's version package, read from the module
// path in core's go.mod.
func agentVersionPackage(core string) (string, error) {
	b, err := os.ReadFile(filepath.Join(core, "go.mod"))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(b), "\n") {
		if mod, ok := strings.CutPrefix(strings.TrimSpace(line), "module "); ok {
			return strings.TrimSpace(mod) + "/internal/version", nil
		}
	}
	return "", fmt.Errorf("%s/go.mod declares no module", core)
}

// Build cross-builds ./cmd/tpm for each platform into dist and writes
// SHA256SUMS over the binaries. Go stamps the version from the commit's tag.
func (p Publisher) Build(dist string, platforms []Platform) ([]Asset, error) {
	if err := os.MkdirAll(dist, 0o755); err != nil {
		return nil, err
	}
	var assets []Asset
	var sums strings.Builder
	for _, pl := range platforms {
		a := Asset{Name: AssetName("tpm", pl), Path: filepath.Join(dist, AssetName("tpm", pl))}
		env := []string{"CGO_ENABLED=0", "GOOS=" + pl.OS, "GOARCH=" + pl.Arch}
		args := []string{"build", "-trimpath"}
		if tag, err := p.releaseTag(); err == nil && tag != "" {
			// The application is a nested module (GH-190): the repository's
			// release tag is not its buildinfo version, so the build stamps it.
			args = append(args, "-ldflags", "-X "+VersionPackage+".Release="+tag)
		}
		args = append(args, "-o", a.Path, "./cmd/tpm")
		if _, err := p.Run(p.Root, env, "go", args...); err != nil {
			return nil, fmt.Errorf("build tpm for %s/%s: %w", pl.OS, pl.Arch, err)
		}
		sum, err := fileSum(a.Path)
		if err != nil {
			return nil, err
		}
		a.Sum = sum
		fmt.Fprintf(&sums, "%s  %s\n", sum, a.Name)
		assets = append(assets, a)
	}
	// The release also carries the runtime: the sibling agent-core's agent
	// for every platform the parent ecosystem publishes, in the same
	// SHA256SUMS, so a program repository's tpm install --runtime fetches
	// agent and checksums from one release (GH-14; srd037 R7).
	if core, err := filepath.Abs(filepath.Join(p.Root, "..", "..", "agent-core")); err == nil {
		if _, statErr := os.Stat(filepath.Join(core, "cmd", "agent")); statErr == nil {
			// The released agent serves this application's words in process
			// (GH-27).
			comp, err := runtime.Compose(core, p.Root)
			if err != nil {
				return nil, err
			}
			defer comp.Cleanup()
			for _, pl := range []Platform{{OS: "darwin", Arch: "amd64"}, {OS: "darwin", Arch: "arm64"}, {OS: "linux", Arch: "amd64"}, {OS: "linux", Arch: "arm64"}} {
				a := Asset{Name: AssetName("agent", pl), Path: filepath.Join(dist, AssetName("agent", pl))}
				env := append([]string{"CGO_ENABLED=0", "GOOS=" + pl.OS, "GOARCH=" + pl.Arch}, comp.Env...)
				agentArgs := append([]string{"build", "-trimpath"}, comp.Flags...)
				if tag, err := p.releaseTag(); err == nil && tag != "" {
					pkg, err := agentVersionPackage(core)
					if err != nil {
						return nil, err
					}
					agentArgs = append(agentArgs, "-ldflags", "-X "+pkg+".Version="+tag)
				}
				agentArgs = append(agentArgs, "-o", a.Path, "./cmd/agent")
				if _, err := p.Run(comp.Dir, env, "go", agentArgs...); err != nil {
					return nil, fmt.Errorf("build agent for %s/%s: %w", pl.OS, pl.Arch, err)
				}
				sum, err := fileSum(a.Path)
				if err != nil {
					return nil, err
				}
				a.Sum = sum
				fmt.Fprintf(&sums, "%s  %s\n", sum, a.Name)
				assets = append(assets, a)
			}
		}
	}
	path := filepath.Join(dist, SumsFile)
	if err := os.WriteFile(path, []byte(sums.String()), 0o644); err != nil {
		return nil, err
	}
	sum, err := fileSum(path)
	if err != nil {
		return nil, err
	}
	return append(assets, Asset{Name: SumsFile, Path: path, Sum: sum}), nil
}

// Publish publishes HEAD's tag once it is on origin: the binaries, then the
// image.
func (p Publisher) Publish() error {
	tag, err := p.headTag()
	if err != nil {
		return err
	}
	if remote, err := p.Git("ls-remote", "--tags", "--refs", "origin", "refs/tags/"+tag); err != nil {
		return err
	} else if remote == "" {
		return fmt.Errorf("%s is not on origin; push it with git push origin %s, then run mage publish", tag, tag)
	}
	if err := p.Binaries(tag); err != nil {
		return err
	}
	if p.ImageRepo != "" {
		return p.PublishImage(tag)
	}
	return nil
}

// releaseTag is the v0.* tag at HEAD when the Publisher has git, else
// empty: a Build without git stamps nothing.
func (p Publisher) releaseTag() (string, error) {
	if p.Git == nil {
		return "", nil
	}
	return p.headTag()
}

// headTag is the v0.* tag at HEAD.
func (p Publisher) headTag() (string, error) {
	tags, err := p.Git("tag", "--points-at", "HEAD", "--list", DatePrefix+"*")
	if err != nil {
		return "", err
	}
	if tags == "" {
		return "", errors.New("HEAD carries no " + DatePrefix + "* tag; mage publish publishes a tag mage tag created")
	}
	return strings.Fields(tags)[0], nil
}

// CheckAccount fails unless gh, with the organization's token, answers as
// the account releases are made as, so nothing on a release names a person.
func (p Publisher) CheckAccount() error {
	env, err := p.ghEnv()
	if err != nil {
		return err
	}
	login, err := p.Run(p.Root, env, "gh", "api", "user", "-q", ".login")
	if err != nil {
		return err
	}
	if login != p.Account {
		return fmt.Errorf("gh answers as %s, not %s; a release is made only as the organization", login, p.Account)
	}
	return nil
}

func (p Publisher) ghEnv() ([]string, error) {
	token, err := p.Run(p.Root, nil, "gh", "auth", "token", "--user", p.Account)
	if err != nil || token == "" {
		return nil, fmt.Errorf("no gh token for %s: %v", p.Account, err)
	}
	return []string{"GH_TOKEN=" + token}, nil
}

// publishedAsset is an asset as gh release view reports it.
type publishedAsset struct {
	Name   string `json:"name"`
	Digest string `json:"digest"`
}

// Binaries builds the release binaries at tag and creates the tag's GitHub
// release with them, uploading only what it lacks and refusing an asset
// already there whose checksum differs. A dry run builds them and names
// what it would create and upload.
func (p Publisher) Binaries(tag string) error {
	gitDir, err := p.Git("rev-parse", "--git-path", "scrum-master-publish")
	if err != nil {
		return err
	}
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(p.Root, gitDir)
	}
	dist := filepath.Join(gitDir, tag)
	if err := os.RemoveAll(dist); err != nil {
		return err
	}
	assets, err := p.Build(dist, Platforms)
	if err != nil {
		return err
	}
	fmt.Fprintf(p.Out, "publish: built %d binaries and %s in %s\n", len(assets)-1, SumsFile, dist)
	if p.DryRun {
		for _, a := range assets {
			fmt.Fprintf(p.Out, "publish: would upload %s (%s)\n", a.Name, a.Sum)
		}
		fmt.Fprintf(p.Out, "publish: dry run; no release created for %s\n", tag)
		return nil
	}
	if err := p.CheckAccount(); err != nil {
		return err
	}
	env, err := p.ghEnv()
	if err != nil {
		return err
	}
	view, err := p.Run(p.Root, env, "gh", "release", "view", tag, "--repo", p.Repo, "--json", "assets")
	if err != nil {
		if !strings.Contains(err.Error(), "release not found") {
			return err
		}
		notes, err := p.Git("tag", "-l", "--format=%(contents)", tag)
		if err != nil {
			return err
		}
		notesFile := filepath.Join(dist, "notes.md")
		if err := os.WriteFile(notesFile, []byte(notes+"\n"), 0o644); err != nil {
			return err
		}
		if _, err := p.Run(p.Root, env, "gh", "release", "create", tag, "--repo", p.Repo, "--verify-tag", "--title", tag, "--notes-file", notesFile); err != nil {
			return err
		}
		fmt.Fprintf(p.Out, "publish: created the release %s\n", tag)
		view = `{"assets":[]}`
	}
	var published struct {
		Assets []publishedAsset `json:"assets"`
	}
	if err := json.Unmarshal([]byte(view), &published); err != nil {
		return fmt.Errorf("gh release view %s: %w", tag, err)
	}
	var upload []string
	for _, a := range assets {
		i := slices.IndexFunc(published.Assets, func(pa publishedAsset) bool { return pa.Name == a.Name })
		if i < 0 {
			upload = append(upload, a.Path)
			continue
		}
		if got := strings.TrimPrefix(published.Assets[i].Digest, "sha256:"); got != a.Sum {
			return fmt.Errorf("%s on the release %s has checksum %s, but this build's is %s; refusing to replace it", a.Name, tag, got, a.Sum)
		}
		fmt.Fprintf(p.Out, "publish: kept %s, already on the release\n", a.Name)
	}
	if len(upload) == 0 {
		fmt.Fprintf(p.Out, "publish: the release %s already holds every binary\n", tag)
		return nil
	}
	if _, err := p.Run(p.Root, env, "gh", append([]string{"release", "upload", tag, "--repo", p.Repo}, upload...)...); err != nil {
		return fmt.Errorf("upload to the release %s: %w; run mage publish again to complete it", tag, err)
	}
	fmt.Fprintf(p.Out, "publish: uploaded %d assets to the release %s\n", len(upload), tag)
	return nil
}

func fileSum(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	s := sha256.Sum256(data)
	return hex.EncodeToString(s[:]), nil
}
