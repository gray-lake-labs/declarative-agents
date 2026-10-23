// Copyright (c) 2026 Gray Lake Labs
// SPDX-License-Identifier: MIT-0
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/gray-lake-labs/declarative-agents/applications/tpm/internal/runtime"
)

func init() {
	integrationTargets = append(integrationTargets, Integration{}.Runtime)
}

// Runtime builds tpm and runs tpm install --runtime against the release tpm pins in an empty cache, and proves the dry run writes nothing, the run leaves a verified agent, the source, and the spec tool where mage audit looks, a second run downloads nothing, a tampered agent is fetched again, and a TPM_AGENT that is not the release's binary is refused (test-rel04.3-distribution). It downloads the release, so it needs network access to github.com, and no rig.
func (Integration) Runtime() error {
	root, err := repositoryRoot()
	if err != nil {
		return err
	}
	tmp, err := os.MkdirTemp("", "tpm-runtime-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	if tmp, err = filepath.EvalSymlinks(tmp); err != nil {
		return err
	}
	bin, cache := filepath.Join(tmp, "tpm"), filepath.Join(tmp, "cache")
	if err := runStreaming(root, "go", "build", "-o", bin, "./cmd/tpm"); err != nil {
		return fmt.Errorf("build tpm: %w", err)
	}
	// An empty user cache directory, with Go's own caches kept where they
	// are so building the spec tool downloads no modules.
	goEnv := map[string]string{}
	for _, v := range []string{"GOCACHE", "GOMODCACHE", "GOPATH"} {
		out, err := exec.Command("go", "env", v).Output()
		if err != nil {
			return err
		}
		goEnv[v] = strings.TrimSpace(string(out))
	}
	env := append(os.Environ(), "HOME="+cache, "XDG_CACHE_HOME="+cache, runtime.AgentEnv+"=")
	for k, v := range goEnv {
		env = append(env, k+"="+v)
	}
	install := func(extra []string, args ...string) (runtime.Report, error) {
		cmd := exec.Command(bin, append([]string{"install", "--runtime", "--json"}, args...)...)
		cmd.Dir, cmd.Env = tmp, append(slices.Clone(env), extra...)
		out, err := cmd.Output()
		var rep runtime.Report
		if err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				return rep, fmt.Errorf("%w: %s", err, ee.Stderr)
			}
			return rep, err
		}
		return rep, json.Unmarshal(out, &rep)
	}

	dry, err := install(nil, "--dry-run")
	if err != nil {
		return fmt.Errorf("dry run: %w", err)
	}
	if !dry.DryRun || len(dry.Downloaded) != 3 {
		return fmt.Errorf("the dry run named %v to download, want SHA256SUMS, the agent, and the source", dry.Downloaded)
	}
	if entries, _ := os.ReadDir(cache); len(entries) > 0 {
		return fmt.Errorf("the dry run wrote into the cache: %v", entries)
	}

	rep, err := install(nil)
	if err != nil {
		return fmt.Errorf("install --runtime: %w", err)
	}
	pin := runtime.DefaultPin()
	if rep.Release != pin || rep.From != runtime.FromCache || !strings.HasPrefix(rep.Agent, cache) || len(rep.Downloaded) != 3 {
		return fmt.Errorf("install --runtime reported %+v", rep)
	}
	if rep.SpecTool == "" || !rep.Built {
		return fmt.Errorf("with Go on PATH the spec tool was not built: %+v", rep)
	}
	if _, err := os.Stat(filepath.Join(rep.CoreRoot, "go.mod")); err != nil {
		return fmt.Errorf("the source root: %w", err)
	}
	// mage audit resolves the same runtime in the same cache.
	os.Setenv("HOME", cache)
	os.Setenv("XDG_CACHE_HOME", cache)
	rel, err := runtime.Locate(pin)
	if err != nil {
		return err
	}
	if rel.Agent != rep.Agent || rel.CoreRoot() != rep.CoreRoot {
		return fmt.Errorf("mage audit's runtime is %s at %s, tpm's %s at %s", rel.Agent, rel.CoreRoot(), rep.Agent, rep.CoreRoot)
	}

	again, err := install(nil)
	if err != nil || len(again.Downloaded) != 0 || again.Built {
		return fmt.Errorf("a second run downloaded %v or rebuilt the spec tool: %v", again.Downloaded, err)
	}

	if err := os.WriteFile(rep.Agent, []byte("tampered"), 0o755); err != nil {
		return err
	}
	refetch, err := install(nil)
	if err != nil || len(refetch.Downloaded) != 1 {
		return fmt.Errorf("a tampered agent was not fetched again: %v, %v", refetch.Downloaded, err)
	}

	other := filepath.Join(tmp, "agent")
	if err := os.WriteFile(other, []byte("a locally built agent"), 0o755); err != nil {
		return err
	}
	if _, err := install([]string{runtime.AgentEnv + "=" + other}); err == nil || !strings.Contains(err.Error(), pin) {
		return fmt.Errorf("a TPM_AGENT that is not the release's binary was not refused naming %s: %v", pin, err)
	}
	same := filepath.Join(tmp, "agent-copy")
	data, err := os.ReadFile(rep.Agent)
	if err != nil {
		return err
	}
	if err := os.WriteFile(same, data, 0o755); err != nil {
		return err
	}
	if env, err := install([]string{runtime.AgentEnv + "=" + same}); err != nil || env.Agent != same || env.From != runtime.FromEnv {
		return fmt.Errorf("the release's binary in TPM_AGENT: %+v, %v", env, err)
	}
	fmt.Println("integration:runtime passed")
	return nil
}
