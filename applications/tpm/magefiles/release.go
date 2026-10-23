package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gray-lake-labs/scrum-master/internal/runtime"
)

// fetchRelease lays out tag's runtime on this host through internal/runtime,
// the cache tpm install --runtime fills (srd037 R7), and returns it with the
// agent Fetch chose: the cached one, or TPM_AGENT when that is the release's.
func fetchRelease(tag string) (runtime.Release, error) {
	rep, err := runtime.Fetch(tag, false)
	if err != nil {
		return runtime.Release{}, err
	}
	rel, err := runtime.Locate(tag)
	rel.Agent = rep.Agent
	return rel, err
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
