package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/magefile/mage/mg"

	"github.com/gray-lake-labs/scrum-master/magefiles/gitlabrig"
)

// Regression creates, uses, and destroys the GitLab regression rig
// (docs/engineering/eng01-gitlab-regression-rig.yaml).
type Regression mg.Namespace

// Aliases makes the full run `mage regression`.
var Aliases = map[string]interface{}{"regression": Regression.All}

// integrationTargets are the integration targets mage regression runs
// between up and down, in order.
var integrationTargets []func() error

func newRig() (gitlabrig.Rig, gitlabrig.API, error) {
	root, err := repositoryRoot()
	if err != nil {
		return gitlabrig.Rig{}, nil, err
	}
	cfg, err := gitlabrig.LoadConfig(root)
	if err != nil {
		return gitlabrig.Rig{}, nil, err
	}
	runner := gitlabrig.ExecRunner{}
	api, token, err := gitlabrig.Preflight(cfg, runner, func(token string) gitlabrig.API {
		return gitlabrig.RESTAPI{Base: cfg.APIURL(), Token: token, Client: &http.Client{Timeout: 30 * time.Second}}
	})
	if err != nil {
		return gitlabrig.Rig{}, nil, err
	}
	return gitlabrig.Rig{Root: root, Config: cfg, Runner: runner, API: api, Token: token}, api, nil
}

// Up creates a rig-<run-id> subgroup with the configured repositories and records it as the current run.
func (Regression) Up() error {
	rig, _, err := newRig()
	if err != nil {
		return err
	}
	run, err := rig.Up()
	if err == nil {
		fmt.Printf("rig %s/rig-%s is up\n", run.Group, run.ID)
	}
	return err
}

// Down destroys the current run's rig subgroup.
func (Regression) Down() error {
	rig, _, err := newRig()
	if err != nil {
		return err
	}
	return rig.Down("")
}

// Sweep deletes rig-* subgroups older than the configured sweep age.
func (Regression) Sweep() error {
	rig, api, err := newRig()
	if err != nil {
		return err
	}
	deleted, err := gitlabrig.Sweep(api, rig.Config.Group, rig.Config.SweepAge, time.Now())
	for _, d := range deleted {
		fmt.Println("deleted", d)
	}
	return err
}

// All runs up, every integration target, then down; down runs even when a target fails.
func (Regression) All() error {
	rig, _, err := newRig()
	if err != nil {
		return err
	}
	return rig.Regression(integrationTargets...)
}
