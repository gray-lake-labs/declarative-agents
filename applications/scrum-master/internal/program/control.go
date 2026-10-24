// Copyright (c) 2026 Gray Lake Labs
// SPDX-License-Identifier: MIT-0
package program

import (
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/checkout"
)

// ControlFor is the program repository whose control files a word reading
// the policy applies, run from the checkout at dir (GH-46): control when a
// --control (or another flag the word takes for it) names one; else the
// system repository, found as the tracker's routing finds it, so a team
// repository's merge is held to the system's policy; else fallback, which
// is empty for the defaults.
func ControlFor(control, dir, fallback string) string {
	if control != "" {
		return control
	}
	if system := SystemControl(dir); system != "" {
		return system
	}
	return fallback
}

// SystemControl is the system repository of the program the checkout at
// dir belongs to: the checkout holding the trackers document that
// checkout.FindTrackers finds there or beside it, named by its origin's
// project path, or by its directory when it has no origin, as a project's
// checkout is the sibling named for the project's last segment. Empty when
// no checkout holds one.
func SystemControl(dir string) string {
	if dir == "" {
		dir = "."
	}
	file, err := checkout.FindTrackers(dir)
	if err != nil || file == "" {
		return ""
	}
	root := filepath.Dir(filepath.Dir(file))
	out, err := exec.Command("git", "-C", root, "remote", "get-url", "origin").Output()
	if url := strings.TrimSpace(string(out)); err == nil && url != "" {
		if project := ProjectPath(url); filepath.Base(project) == filepath.Base(root) {
			return project
		}
	}
	return filepath.Base(root)
}
