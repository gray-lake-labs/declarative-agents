package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gray-lake-labs/scrum-master/internal/spectool"
)

// parentRelease is the pinned parent release on this host: its source tree,
// a git repository holding that tree at the tag (the clone cache
// derived_from entries resolve against), and its agent.
type parentRelease struct {
	Tag, Dir, Src, Agent string
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

// fetchRelease downloads the release's agent for this host,
// verifies them against SHA256SUMS, and lays out its source as a git
// repository at the tag. Each part is fetched once per tag; a cached binary
// is used only while it still matches SHA256SUMS.
func fetchRelease(tag string) (parentRelease, error) {
	dir, err := spectool.ReleaseDir(tag)
	if err != nil {
		return parentRelease{}, err
	}
	r := parentRelease{Tag: tag, Dir: dir, Src: filepath.Join(dir, "src"), Agent: filepath.Join(dir, "agent")}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return r, err
	}
	sums, err := cached(filepath.Join(dir, "SHA256SUMS"), spectool.ReleaseURL+"/releases/download/"+tag+"/SHA256SUMS", 0o644)
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
		data, err := spectool.Download(spectool.ReleaseURL + "/releases/download/" + tag + "/" + asset)
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
	return r, spectool.EnsureSource(r.Src, r.Tag)
}

// cached reads path, downloading url into it first when it is absent.
func cached(path, url string, mode os.FileMode) ([]byte, error) {
	if data, err := os.ReadFile(path); err == nil {
		return data, nil
	}
	data, err := spectool.Download(url)
	if err != nil {
		return nil, err
	}
	return data, os.WriteFile(path, data, mode)
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
