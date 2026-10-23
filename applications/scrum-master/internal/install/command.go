// Copyright (c) 2026 Gray Lake Labs
// SPDX-License-Identifier: MIT-0
package install

import (
	"flag"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/cli"
	"github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/runtime"
	"github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/version"
)

// Version is the tpm version stamped into installed files: the release the
// binary was published at, else its module version, or dev for a local
// build (internal/version).
func Version() string { return version.Current() }

func gitRoot() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("not inside a git repository; name one with --dir")
	}
	return strings.TrimSpace(string(out)), nil
}

func rel(dir, path string) string {
	if r, err := filepath.Rel(dir, path); err == nil {
		return filepath.ToSlash(r)
	}
	return path
}

func init() {
	var (
		agents             []string
		dir                string
		list, check, force bool
		withRuntime        bool
	)
	cli.Default.Register(&cli.Command{
		Group:   "install",
		Summary: "write the embedded skill commands for an agent, or fetch the agent runtime",
		Flags: func(fs *flag.FlagSet) {
			agents = nil
			fs.Func("agent", "cursor, claude, copilot, or all (repeatable)", func(v string) error {
				agents = append(agents, v)
				return nil
			})
			fs.StringVar(&dir, "dir", "", "repository to install into (default: the git root)")
			fs.BoolVar(&list, "list", false, "name the embedded skills and write nothing")
			fs.BoolVar(&check, "check", false, "report installed files that differ from the binary; write nothing")
			fs.BoolVar(&force, "force", false, "replace files a person edited")
			fs.BoolVar(&withRuntime, "runtime", false, "fetch the declarative-agents runtime at the release tpm pins into the user cache; write nothing in --dir")
		},
		Run: func(ctx *cli.Context) error {
			if withRuntime {
				return installRuntime(ctx)
			}
			skills, err := EmbeddedSkills()
			if err != nil {
				return err
			}
			if list {
				var b strings.Builder
				for _, s := range skills {
					fmt.Fprintf(&b, "%-20s %s\n", s.Name, s.Description)
				}
				return ctx.Emit(skills, b.String())
			}
			targets, err := ParseAgents(agents)
			if err != nil {
				return fmt.Errorf("%w: %v", cli.ErrUsage, err)
			}
			if dir == "" {
				if dir, err = gitRoot(); err != nil {
					return err
				}
			}
			version := Version()
			if check {
				findings, err := Check(dir, targets, skills, version)
				if err != nil {
					return err
				}
				var b strings.Builder
				for _, f := range findings {
					fmt.Fprintf(&b, "%-9s %s\n", f.Problem, rel(dir, f.Path))
				}
				if len(findings) == 0 {
					b.WriteString("installed skills match tpm " + version)
				}
				if err := ctx.Emit(findings, b.String()); err != nil {
					return err
				}
				if len(findings) > 0 {
					return fmt.Errorf("%d installed skill file(s) differ from tpm %s; run tpm install", len(findings), version)
				}
				return nil
			}
			results, err := Install(dir, targets, skills, version, force, ctx.DryRun)
			if err != nil {
				return err
			}
			var b strings.Builder
			for _, r := range results {
				fmt.Fprintf(&b, "%-9s %s", r.Action, rel(dir, r.Path))
				if r.Reason != "" {
					fmt.Fprintf(&b, " (%s)", r.Reason)
				}
				b.WriteString("\n")
			}
			return ctx.Emit(results, b.String())
		},
	})
}

// installRuntime lays out the runtime (srd037 R7). Inside the repository
// that ships agent-core, the sibling is the release: the agent builds from
// source and nothing downloads (GH-190). Elsewhere it fetches the release
// the repository pins in docs/external-pins.yaml, else the default pin.
func installRuntime(ctx *cli.Context) error {
	var build func(dir, name string, args ...string) error
	if !ctx.DryRun {
		build = runtime.RunCommand
	}
	if rel, ok, err := runtime.Sibling(".", build); ok {
		if err != nil {
			return err
		}
		rep := runtime.Report{Release: rel.Tag, Agent: rel.Agent, From: "repository", CoreRoot: rel.CoreRoot(), DryRun: ctx.DryRun}
		var b strings.Builder
		fmt.Fprintf(&b, "release    %s (this repository)\n", rep.Release)
		fmt.Fprintf(&b, "agent      %s (built from agent-core)\n", rep.Agent)
		fmt.Fprintf(&b, "core root  %s\n", rep.CoreRoot)
		return ctx.Emit(rep, b.String())
	}
	tag := runtime.RepoPin(".")
	rep, err := runtime.Fetch(tag, ctx.DryRun)
	if err != nil {
		return err
	}
	if err := runtime.BuildSpecTool(&rep); err != nil {
		return err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "release    %s\n", rep.Release)
	fmt.Fprintf(&b, "agent      %s (%s)\n", rep.Agent, rep.From)
	fmt.Fprintf(&b, "core root  %s\n", rep.CoreRoot)
	switch {
	case rep.Built && rep.DryRun:
		fmt.Fprintf(&b, "spec tool  %s (would build)\n", rep.SpecTool)
	case rep.Built:
		fmt.Fprintf(&b, "spec tool  %s (built)\n", rep.SpecTool)
	case rep.SpecTool != "":
		fmt.Fprintf(&b, "spec tool  %s\n", rep.SpecTool)
	default:
		fmt.Fprintf(&b, "spec tool  not built: %s\n", rep.Skipped)
	}
	verb := "downloaded"
	if rep.DryRun {
		verb = "would download"
	}
	if len(rep.Downloaded) == 0 {
		b.WriteString("nothing downloaded; the cache is current\n")
	} else {
		fmt.Fprintf(&b, "%s %s\n", verb, strings.Join(rep.Downloaded, ", "))
	}
	return ctx.Emit(rep, b.String())
}
