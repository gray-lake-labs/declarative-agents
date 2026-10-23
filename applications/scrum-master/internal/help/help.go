// Copyright (c) 2026 Gray Lake Labs
// SPDX-License-Identifier: MIT-0
// Package help is tpm help: the process, each group, and each load-bearing
// concept, as pages embedded in the binary (srd044).
package help

import (
	"embed"
	"flag"
	"fmt"
	"path"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/cli"
)

//go:embed pages/*.txt
var pages embed.FS

// ProcessPage is the page tpm help prints with no argument.
const ProcessPage = "process"

// MaxLines is the most lines a page holds (srd044 R2.1).
const MaxLines = 60

// Page is one help page. Commands are the tpm commands it names and SRDs
// the SRDs it cites, read from its text.
type Page struct {
	Name     string   `json:"name"`
	Summary  string   `json:"summary"`
	Text     string   `json:"text"`
	Commands []string `json:"commands"`
	SRDs     []string `json:"srds"`
}

var (
	commandPattern = regexp.MustCompile(`\btpm ([a-z][a-z-]*)(?: ([a-z][a-z-]*))?`)
	srdPattern     = regexp.MustCompile(`\bsrd[0-9]{3}\b`)
)

// Pages are every page, sorted by name. A page's first line is
// "name: summary".
func Pages() ([]Page, error) {
	entries, err := pages.ReadDir("pages")
	if err != nil {
		return nil, err
	}
	var out []Page
	for _, e := range entries {
		b, err := pages.ReadFile(path.Join("pages", e.Name()))
		if err != nil {
			return nil, err
		}
		name := strings.TrimSuffix(e.Name(), ".txt")
		text := string(b)
		first, _, _ := strings.Cut(text, "\n")
		head, summary, ok := strings.Cut(first, ": ")
		if !ok || head != name {
			return nil, fmt.Errorf("help page %s opens with %q, not %q", name, first, name+": <summary>")
		}
		out = append(out, Page{Name: name, Summary: summary, Text: text, SRDs: unique(srdPattern.FindAllString(text, -1))})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// Mention is a tpm command a page's text names: its group, its subcommand
// when the word after the group is one, and that word.
type Mention struct{ Group, Sub, Word string }

// Mentions reads every "tpm <group> [<subcommand>]" in text, resolving the
// subcommand against r.
func Mentions(r *cli.Registry, text string) []Mention {
	var out []Mention
	for _, m := range commandPattern.FindAllStringSubmatch(text, -1) {
		mention := Mention{Group: m[1], Word: m[2]}
		for _, c := range r.Commands(m[1]) {
			if m[2] != "" && c.Name == m[2] {
				mention.Sub = m[2]
			}
		}
		out = append(out, mention)
	}
	return out
}

// Listed are the commands a group page lists: the lines of its
// "Subcommands:" or "Command:" block, up to the next blank line, each a
// command, two or more spaces, and its sentence.
func Listed(text string) []Mention {
	var out []Mention
	in := false
	for _, l := range strings.Split(text, "\n") {
		switch t := strings.TrimSpace(l); {
		case t == "Subcommands:" || t == "Command:":
			in = true
		case in && t == "":
			in = false
		case in:
			f := strings.Fields(columns.Split(t, 2)[0])
			if len(f) >= 2 && f[0] == "tpm" {
				m := Mention{Group: f[1]}
				if len(f) >= 3 {
					m.Sub = f[2]
				}
				out = append(out, m)
			}
		}
	}
	return out
}

var columns = regexp.MustCompile(`\s{2,}`)

func unique(xs []string) []string {
	var out []string
	for _, x := range xs {
		if !slices.Contains(out, x) {
			out = append(out, x)
		}
	}
	return out
}

func (p Page) withCommands(r *cli.Registry) Page {
	p.Commands = nil
	for _, m := range Mentions(r, p.Text) {
		c := strings.TrimSpace("tpm " + m.Group + " " + m.Sub)
		if !slices.Contains(p.Commands, c) {
			p.Commands = append(p.Commands, c)
		}
	}
	return p
}

func init() {
	ps, err := Pages()
	if err != nil {
		panic("tpm: " + err.Error())
	}
	var list bool
	show := func(name string) func(ctx *cli.Context) error {
		return func(ctx *cli.Context) error {
			all, _ := Pages()
			if name == ProcessPage && list {
				var b strings.Builder
				for _, p := range all {
					fmt.Fprintf(&b, "%-12s %s\n", p.Name, p.Summary)
				}
				for i := range all {
					all[i] = all[i].withCommands(cli.Default)
				}
				return ctx.Emit(all, b.String())
			}
			for _, p := range all {
				if p.Name == name {
					return ctx.Emit(p.withCommands(cli.Default), p.Text)
				}
			}
			return fmt.Errorf("no help page %s", name)
		}
	}
	for _, p := range ps {
		c := &cli.Command{Group: "help", Name: p.Name, Summary: p.Summary, Run: show(p.Name)}
		if p.Name == ProcessPage {
			c.Name = ""
			c.Flags = func(fs *flag.FlagSet) {
				list = false
				fs.BoolVar(&list, "list", false, "name every page with its summary")
			}
		}
		cli.Default.Register(c)
		if p.Name == ProcessPage {
			cli.Default.Register(&cli.Command{Group: "help", Name: ProcessPage, Summary: p.Summary, Run: show(p.Name)})
		}
	}
}
