package main

import (
	"os"

	"github.com/gray-lake-labs/scrum-master/internal/release"
)

// Tag runs the gates in release-gates.md and, when every gate passes,
// creates the next v0.YYYYMMDD.N tag at HEAD and pushes it as the file says
// (eng03). /gh-release-push calls it on main.
func Tag() error {
	r, err := scrumMasterRelease()
	if err != nil {
		return err
	}
	return r.Tag()
}

// TagDryRun runs the same checks and gates and creates no tag.
func TagDryRun() error {
	r, err := scrumMasterRelease()
	if err != nil {
		return err
	}
	return r.DryRun()
}

func scrumMasterRelease() (release.Release, error) {
	root, err := repositoryRoot()
	if err != nil {
		return release.Release{}, err
	}
	return release.Release{
		Root: root,
		Git:  release.ExecGit(root),
		Exec: release.ShellExec,
		Out:  os.Stdout,
	}, nil
}
