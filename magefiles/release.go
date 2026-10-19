package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// releaseURL is where the parent publishes each tag's binaries and source.
const releaseURL = "https://github.com/gray-lake-labs/declarative-agents"

// parentRelease is the pinned parent release on this host: its source tree,
// a git repository holding that tree at the tag (the clone cache
// derived_from entries resolve against), and its agent.
type parentRelease struct {
	Tag, Dir, Src, Agent string
}

// releaseDir is where a release is downloaded and verified, once per tag,
// under the user cache directory.
func releaseDir(tag string) (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "scrum-master", "release", tag), nil
}

// hostAsset is the name of a release binary for goos and goarch; the parent
// publishes darwin and linux on amd64 and arm64.
func hostAsset(name, goos, goarch string) (string, error) {
	if (goos != "darwin" && goos != "linux") || (goarch != "amd64" && goarch != "arm64") {
		return "", fmt.Errorf("the parent publishes no %s binary for %s/%s; it publishes darwin and linux on amd64 and arm64", name, goos, goarch)
	}
	return name + "_" + goos + "_" + goarch, nil
}

// verifyChecksum holds data to the SHA-256 SHA256SUMS lists for asset.
func verifyChecksum(sums []byte, asset string, data []byte) error {
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

func download(url string) ([]byte, error) {
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// fetchRelease downloads the release's agent for this host,
// verifies them against SHA256SUMS, and lays out its source as a git
// repository at the tag. Each part is fetched once per tag; a cached binary
// is used only while it still matches SHA256SUMS.
func fetchRelease(tag string) (parentRelease, error) {
	dir, err := releaseDir(tag)
	if err != nil {
		return parentRelease{}, err
	}
	r := parentRelease{Tag: tag, Dir: dir, Src: filepath.Join(dir, "src"), Agent: filepath.Join(dir, "agent")}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return r, err
	}
	sums, err := cached(filepath.Join(dir, "SHA256SUMS"), releaseURL+"/releases/download/"+tag+"/SHA256SUMS", 0o644)
	if err != nil {
		return r, fmt.Errorf("the checksums of %s: %w", tag, err)
	}
	for name, path := range map[string]string{"agent": r.Agent} {
		asset, err := hostAsset(name, runtime.GOOS, runtime.GOARCH)
		if err != nil {
			return r, err
		}
		if data, err := os.ReadFile(path); err == nil && verifyChecksum(sums, asset, data) == nil {
			continue
		}
		data, err := download(releaseURL + "/releases/download/" + tag + "/" + asset)
		if err != nil {
			return r, fmt.Errorf("the %s of %s: %w", name, tag, err)
		}
		if err := verifyChecksum(sums, asset, data); err != nil {
			return r, err
		}
		if err := os.WriteFile(path, data, 0o755); err != nil {
			return r, err
		}
	}
	return r, ensureSource(r)
}

// cached reads path, downloading url into it first when it is absent.
func cached(path, url string, mode os.FileMode) ([]byte, error) {
	if data, err := os.ReadFile(path); err == nil {
		return data, nil
	}
	data, err := download(url)
	if err != nil {
		return nil, err
	}
	return data, os.WriteFile(path, data, mode)
}

// ensureSource extracts the tag's source archive into r.Src and commits it
// there as a repository tagged r.Tag, so the audit resolves derived_from
// entries offline against it.
func ensureSource(r parentRelease) error {
	if gitQuiet(r.Src, "rev-parse", "--verify", "-q", "refs/tags/"+r.Tag) == nil {
		return nil
	}
	archive, err := download(releaseURL + "/archive/refs/tags/" + r.Tag + ".tar.gz")
	if err != nil {
		return fmt.Errorf("the source of %s: %w", r.Tag, err)
	}
	if err := os.RemoveAll(r.Src); err != nil {
		return err
	}
	if err := os.MkdirAll(r.Src, 0o755); err != nil {
		return err
	}
	untar := exec.Command("tar", "-xzf", "-", "--strip-components=1")
	untar.Dir, untar.Stdin = r.Src, bytes.NewReader(archive)
	if out, err := untar.CombinedOutput(); err != nil {
		return fmt.Errorf("extract the source of %s: %w: %s", r.Tag, err, out)
	}
	if _, err := os.Stat(filepath.Join(r.Src, "agent-core", "go.mod")); err != nil {
		return fmt.Errorf("the source of %s has no agent-core/go.mod: %w", r.Tag, err)
	}
	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"add", "-A"},
		{"-c", "user.name=scrum-master", "-c", "user.email=scrum-master@example.invalid", "commit", "-q", "-m", "release " + r.Tag},
		{"tag", r.Tag},
	} {
		if err := gitQuiet(r.Src, args...); err != nil {
			return fmt.Errorf("lay out the source of %s as a repository: %w", r.Tag, err)
		}
	}
	return nil
}

func gitQuiet(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

// withCheckouts writes docs/external-checkouts.yaml in root pointing each
// parent repository at src while the audit runs, and returns what removes it.
// A checkouts file the person wrote is left as it is.
func withCheckouts(root, src string, repos ...string) (func(), error) {
	path := filepath.Join(root, checkoutsFile)
	if _, err := os.Stat(path); err == nil {
		return func() {}, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	var b strings.Builder
	b.WriteString("# Written by mage audit for this run; it points at the downloaded release.\ncheckouts:\n")
	for _, repo := range repos {
		fmt.Fprintf(&b, "  %s: %s\n", repo, src)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return nil, err
	}
	return func() { os.Remove(path) }, nil
}
