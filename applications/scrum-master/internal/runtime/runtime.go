// Copyright (c) 2026 Gray Lake Labs
// SPDX-License-Identifier: MIT-0
// Package runtime fetches and verifies the declarative-agents runtime: the
// release's agent binary for this host and the release's source, whose
// agent-core tree a profile runs against (srd037 R7). tpm install --runtime
// fetches it at the release tpm pins, tpm doctor reports it (srd043 R4.7),
// and mage audit runs the corpus audit with it at the release the
// repository pins. Every part lives in the release's directory under the
// user cache directory, the one the spec tool builds in, so a machine
// downloads each tag once.
package runtime

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	goruntime "runtime"
	"runtime/debug"
	"strings"

	"github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/spectool"
)

// AgentEnv names an agent binary already on the machine, used in place of
// the cached one when it is the pinned release's (srd037 R7.4).
const AgentEnv = "TPM_AGENT"

// The places an agent comes from.
const (
	FromCache = "cache"
	FromEnv   = AgentEnv
)

// fallbackPin is the newest release of this repository with published agent
// binaries, written into a new corpus when this tpm is not itself a release
// build; a released tpm writes its own version, since tpm and the runtime
// ship from one repository (GH-190, srd043 R3.3).
const fallbackPin = "v0.20261018.1"

var releaseTag = regexp.MustCompile(`^v0\.\d{8}\.\d+$`)

// DefaultPin is the release a new corpus pins and tpm install --runtime
// fetches when the repository pins nothing: this tpm's own release, else
// the newest release with published binaries (GH-190).
func DefaultPin() string {
	if bi, ok := debug.ReadBuildInfo(); ok && releaseTag.MatchString(bi.Main.Version) {
		return bi.Main.Version
	}
	return fallbackPin
}

// RepoPin is the release the repository around dir pins in its
// docs/external-pins.yaml, else DefaultPin (GH-190).
func RepoPin(dir string) string {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return DefaultPin()
	}
	if tag, err := spectool.Pinned(strings.TrimSpace(string(out))); err == nil {
		return tag
	}
	return DefaultPin()
}

// Sibling is the runtime this repository holds beside the application, when
// dir is inside the repository that ships agent-core: the repository root
// is the source and the agent builds from agent-core into agent-core/bin —
// no pin and no download, the sibling is the release (GH-190). With a nil
// builder it only locates; ok reports whether dir is in that repository.
func Sibling(dir string, build func(dir, name string, args ...string) error) (Release, bool, error) {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return Release{}, false, nil
	}
	root := strings.TrimSpace(string(out))
	core := filepath.Join(root, "agent-core")
	if _, err := os.Stat(filepath.Join(core, "cmd", "agent")); err != nil {
		return Release{}, false, nil
	}
	// Build products live in the release cache, not in the tree: a scratch
	// module inside the repository would read as a maintained module.
	dir, derr := spectool.ReleaseDir("in-repo")
	if derr != nil {
		return Release{}, true, derr
	}
	r := Release{Tag: "in-repo", Dir: dir, Src: root, Agent: filepath.Join(dir, "agent")}
	if build == nil {
		return r, true, nil
	}
	if err := build(core, "go", "build", "-o", r.Agent, "./cmd/agent"); err != nil {
		return r, true, err
	}
	return r, true, nil
}

// Release is a release's runtime on this host: its directory under the
// user cache directory, its source tree (a git repository holding the tree
// at the tag), and its agent.
type Release struct {
	Tag, Dir, Src, Agent string
}

// CoreRoot is the agent-core tree a profile's paths resolve against: the
// agent's --core-root.
func (r Release) CoreRoot() string { return filepath.Join(r.Src, "agent-core") }

// Locate names where tag's runtime lives on this host, fetching nothing.
func Locate(tag string) (Release, error) {
	dir, err := spectool.ReleaseDir(tag)
	if err != nil {
		return Release{}, err
	}
	return Release{Tag: tag, Dir: dir, Src: filepath.Join(dir, "src"), Agent: filepath.Join(dir, "agent")}, nil
}

// HostAsset is the name of a release binary for goos and goarch; the parent
// publishes darwin and linux on amd64 and arm64.
func HostAsset(name, goos, goarch string) (string, error) {
	if (goos != "darwin" && goos != "linux") || (goarch != "amd64" && goarch != "arm64") {
		return "", fmt.Errorf("the parent publishes no %s binary for %s/%s; it publishes darwin and linux on amd64 and arm64", name, goos, goarch)
	}
	return name + "_" + goos + "_" + goarch, nil
}

// VerifyChecksum holds data to the SHA-256 SHA256SUMS lists for asset.
func VerifyChecksum(sums []byte, asset string, data []byte) error {
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	sc := bufio.NewScanner(bytes.NewReader(sums))
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == asset {
			if fields[0] != got {
				return fmt.Errorf("%s has checksum %s, but SHA256SUMS lists %s; refusing it", asset, got, fields[0])
			}
			return nil
		}
	}
	return fmt.Errorf("SHA256SUMS lists no %s", asset)
}

// Report is what tpm install --runtime did, or on a dry run would do
// (srd037 R7.5).
type Report struct {
	Release string `json:"release"`
	Agent   string `json:"agent"`
	// From is where the agent came from: the cache or TPM_AGENT.
	From     string `json:"from"`
	CoreRoot string `json:"core_root"`
	SpecTool string `json:"spec_tool,omitempty"`
	// Built says this run built the spec tool, or on a dry run would.
	Built      bool     `json:"spec_tool_built,omitempty"`
	Skipped    string   `json:"spec_tool_skipped,omitempty"`
	Downloaded []string `json:"downloaded,omitempty"`
	DryRun     bool     `json:"dry_run"`
}

// lookGo finds the Go toolchain the spec tool is built with; tests replace it.
var lookGo = func() error { _, err := exec.LookPath("go"); return err }

// Fetch lays tag's runtime out on this host: SHA256SUMS, the agent for the
// host held to it (or TPM_AGENT, when it is that binary), and the release's
// source (srd037 R7.1, R7.2, R7.4). A part already cached and still
// matching is not downloaded again. A dry run names what it would download
// and writes nothing.
func Fetch(tag string, dryRun bool) (Report, error) {
	r, err := Locate(tag)
	if err != nil {
		return Report{}, err
	}
	rep := Report{Release: tag, Agent: r.Agent, From: FromCache, CoreRoot: r.CoreRoot(), DryRun: dryRun}
	asset, err := HostAsset("agent", goruntime.GOOS, goruntime.GOARCH)
	if err != nil {
		return rep, err
	}
	sumsPath := filepath.Join(r.Dir, "SHA256SUMS")
	sums, err := os.ReadFile(sumsPath)
	if err != nil {
		rep.Downloaded = append(rep.Downloaded, "SHA256SUMS")
		if !dryRun {
			if sums, err = download(releaseFile(tag, "SHA256SUMS")); err != nil {
				return rep, fmt.Errorf("the checksums of %s: %w", tag, err)
			}
			if err := writeFile(sumsPath, sums, 0o644); err != nil {
				return rep, err
			}
		}
	}
	if env := os.Getenv(AgentEnv); env != "" {
		if sums != nil {
			if err := verifyFile(env, sums, asset, tag); err != nil {
				return rep, err
			}
		}
		rep.Agent, rep.From = env, FromEnv
	} else if !current(r.Agent, sums, asset) {
		rep.Downloaded = append(rep.Downloaded, asset)
		if !dryRun {
			data, err := download(releaseFile(tag, asset))
			if err != nil {
				return rep, fmt.Errorf("the agent of %s: %w", tag, err)
			}
			if err := VerifyChecksum(sums, asset, data); err != nil {
				return rep, err
			}
			if err := writeFile(r.Agent, data, 0o755); err != nil {
				return rep, err
			}
		}
	}
	if !sourceLaidOut(r) {
		rep.Downloaded = append(rep.Downloaded, "source")
		if !dryRun {
			if err := spectool.EnsureSource(r.Src, tag); err != nil {
				return rep, err
			}
		}
	}
	return rep, nil
}

// BuildSpecTool builds the spec tool in tag's directory when Go is on PATH
// and it is not built there yet, so tpm doctor repairs a corpus at tag
// without Go later (srd037 R7.3). Without Go it says why the tool is absent.
// Fetch lays out the source it builds from first.
func BuildSpecTool(rep *Report) error {
	r, err := Locate(rep.Release)
	if err != nil {
		return err
	}
	tool := spectool.Binary(r.Dir)
	switch {
	case exists(tool):
		rep.SpecTool = tool
	case lookGo() != nil:
		rep.Skipped = "no Go toolchain on PATH; tpm doctor repairs at this release without Go only once the spec tool is built here"
	case rep.DryRun:
		rep.SpecTool, rep.Built = tool, true
	default:
		built, err := spectool.Ensure(r.Dir, r.CoreRoot(), run)
		if err != nil {
			return err
		}
		rep.SpecTool, rep.Built = built, true
	}
	return nil
}

// Built reports whether the spec tool is already built in tag's directory.
func Built(tag string) bool {
	r, err := Locate(tag)
	return err == nil && exists(spectool.Binary(r.Dir))
}

// Status is the runtime tpm would use at a release, read without fetching
// anything (srd043 R4.7).
type Status struct {
	Release string `json:"release"`
	Agent   string `json:"agent,omitempty"`
	From    string `json:"from,omitempty"`
	// Problem says why no agent is usable, and the fix.
	Problem string `json:"problem,omitempty"`
}

// Inspect reports tag's agent on this host: TPM_AGENT or the cached one,
// held to the cached SHA256SUMS. It downloads nothing, so a release whose
// checksums are not cached has no agent it can vouch for.
func Inspect(tag string) Status {
	s := Status{Release: tag}
	const fix = "; run tpm install --runtime"
	r, err := Locate(tag)
	if err != nil {
		s.Problem = err.Error()
		return s
	}
	asset, err := HostAsset("agent", goruntime.GOOS, goruntime.GOARCH)
	if err != nil {
		s.Problem = err.Error()
		return s
	}
	path, from := r.Agent, FromCache
	if env := os.Getenv(AgentEnv); env != "" {
		path, from = env, FromEnv
	}
	sums, err := os.ReadFile(filepath.Join(r.Dir, "SHA256SUMS"))
	if err != nil {
		s.Problem = "the checksums of " + tag + " are not cached, so no agent can be checked against them" + fix
		return s
	}
	if err := verifyFile(path, sums, asset, tag); err != nil {
		s.Problem = err.Error()
		if from == FromCache {
			s.Problem += fix
		}
		return s
	}
	s.Agent, s.From = path, from
	return s
}

// verifyFile holds the binary at path to the release's checksum for asset,
// naming the release when it is some other binary.
func verifyFile(path string, sums []byte, asset, tag string) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("no agent at %s", path)
	}
	if err != nil {
		return err
	}
	if err := VerifyChecksum(sums, asset, data); err != nil {
		return fmt.Errorf("%s is not the %s agent %s publishes for this host: %w", path, asset, tag, err)
	}
	return nil
}

// current reports whether the binary at path matches sums for asset.
func current(path string, sums []byte, asset string) bool {
	if sums == nil {
		return false
	}
	data, err := os.ReadFile(path)
	return err == nil && VerifyChecksum(sums, asset, data) == nil
}

// sourceLaidOut reports whether r's source is already a repository at its tag.
func sourceLaidOut(r Release) bool {
	return exec.Command("git", "-C", r.Src, "rev-parse", "--verify", "-q", "refs/tags/"+r.Tag).Run() == nil
}

func releaseFile(tag, name string) string {
	return spectool.ReleaseURL + "/releases/download/" + tag + "/" + name
}

// download fetches url; tests replace it.
var download = spectool.Download

func writeFile(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, mode)
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// RunCommand runs name with args in dir and keeps its output for the error:
// the builder Sibling takes.
func RunCommand(dir, name string, args ...string) error { return run(dir, name, args...) }

// run runs name with args in dir and keeps its output for the error.
func run(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}
