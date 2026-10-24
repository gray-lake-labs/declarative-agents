// Copyright (c) 2026 Gray Lake Labs
// SPDX-License-Identifier: MIT-0
package checkout

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// TrackersFile is where a program's tracking systems are configured, in
// its system repository (srd042 R1.3); the checkout holding it is the
// system repository's.
const TrackersFile = "control/trackers.md"

// FindTrackers is the trackers document for the checkout at dir: the checkout's
// own control/trackers.md, else the one sibling checkout that holds one, as
// repositories sit side by side under their owner and only the system
// repository has control files. Worktrees of one repository, which pop
// opens beside its checkout, are one sibling, read from the main checkout.
// It returns "" when there is none, and refuses siblings of more than one
// repository.
func FindTrackers(dir string) (string, error) {
	root := dir
	if out, err := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel").Output(); err == nil {
		root = strings.TrimSpace(string(out))
	}
	if own := filepath.Join(root, TrackersFile); exists(own) {
		return own, nil
	}
	siblings := oneEachRepository(globSiblings(root))
	switch len(siblings) {
	case 0:
		return "", nil
	case 1:
		return siblings[0], nil
	}
	return "", fmt.Errorf("several sibling checkouts hold %s (%s); run scrum-master from the system repository's checkout", TrackersFile, strings.Join(siblings, ", "))
}

func exists(p string) bool { _, err := os.Stat(p); return err == nil }

func globSiblings(root string) []string {
	siblings, _ := filepath.Glob(filepath.Join(filepath.Dir(root), "*", TrackersFile))
	return siblings
}

// oneEachRepository keeps one trackers document per git repository among
// files: worktrees share their repository's common git directory, and the
// main checkout's file is the one kept, since a worktree's branch may carry
// an unmerged change to it.
func oneEachRepository(files []string) []string {
	var out []string
	kept := map[string]int{}
	for _, f := range files {
		checkout := filepath.Dir(filepath.Dir(f))
		common := checkout
		if b, err := exec.Command("git", "-C", checkout, "rev-parse", "--path-format=absolute", "--git-common-dir").Output(); err == nil {
			common = strings.TrimSpace(string(b))
		}
		main := filepath.Clean(filepath.Dir(common)) == filepath.Clean(checkout)
		if at, seen := kept[common]; seen {
			if main {
				out[at] = f
			}
			continue
		}
		kept[common] = len(out)
		out = append(out, f)
	}
	return out
}
