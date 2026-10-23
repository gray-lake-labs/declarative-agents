// Copyright (c) 2026 Gray Lake Labs
// SPDX-License-Identifier: MIT-0
package main

import (
	"os"

	"github.com/gray-lake-labs/declarative-agents/applications/tpm/internal/release"
)

// The GitHub repository releases are published to, and the organization
// account they are published as (eng03).
const (
	releaseRepo    = "gray-lake-labs/declarative-agents"
	releaseAccount = "gray-lake-labs-owner"
	releaseImage   = "ghcr.io/gray-lake-labs/tpm"
)

// Tag runs the gates in release-gates.md and, when every gate passes,
// creates the next v0.YYYYMMDD.N tag at HEAD, pushes it as the file says,
// and publishes it once it is on origin (eng03). /gh-release-push calls it
// on main.
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

// Publish publishes the tag at HEAD once it is on origin: the release binaries and SHA256SUMS on its GitHub release, then its image; a part already published is checked and kept, so a rerun completes a partial release (eng03).
func Publish() error {
	p, err := scrumMasterPublisher(false)
	if err != nil {
		return err
	}
	return p.Publish()
}

// PublishDryRun builds the release binaries and SHA256SUMS for the tag at HEAD and names what mage publish would upload and push, creating nothing.
func PublishDryRun() error {
	p, err := scrumMasterPublisher(true)
	if err != nil {
		return err
	}
	tag, err := p.Git("describe", "--tags", "--exact-match", "--match", release.DatePrefix+"*", "HEAD")
	if err != nil {
		tag = "untagged"
	}
	if err := p.Binaries(tag); err != nil {
		return err
	}
	return p.PublishImage(tag)
}

// CrossBuild builds tpm for every release platform and keeps nothing, so a release fails its gates before tagging when a platform does not build (eng03).
func CrossBuild() error {
	p, err := scrumMasterPublisher(true)
	if err != nil {
		return err
	}
	dist, err := os.MkdirTemp("", "tpm-cross-build-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dist)
	_, err = p.Build(dist, release.Platforms)
	return err
}

func scrumMasterRelease() (release.Release, error) {
	root, err := repositoryRoot()
	if err != nil {
		return release.Release{}, err
	}
	p, err := scrumMasterPublisher(false)
	if err != nil {
		return release.Release{}, err
	}
	return release.Release{
		Root:      root,
		Git:       release.ExecGit(root),
		Exec:      release.ShellExec,
		Out:       os.Stdout,
		Preflight: p.Check,
		Publish:   func(string) error { return p.Publish() },
	}, nil
}

func scrumMasterPublisher(dryRun bool) (release.Publisher, error) {
	root, err := repositoryRoot()
	if err != nil {
		return release.Publisher{}, err
	}
	return release.Publisher{
		Root:      root,
		Git:       release.ExecGit(root),
		Run:       release.ExecCommand,
		Out:       os.Stdout,
		Repo:      releaseRepo,
		Account:   releaseAccount,
		DryRun:    dryRun,
		ImageRepo: releaseImage,
	}, nil
}
