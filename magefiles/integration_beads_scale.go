package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gray-lake-labs/scrum-master/internal/beads"
)

// The beads scale target is opt-in, like regression:gitlab: it is slow and
// measures rather than proves, so regression:all does not run it. Its sizes
// come from the environment, each a comma-separated list.
const (
	scaleIssuesEnv  = "BEADS_SCALE_ISSUES"  // default 1000,5000,10000
	scaleWritersEnv = "BEADS_SCALE_WRITERS" // default 2,5,10
	scaleOpsEnv     = "BEADS_SCALE_OPS"     // default 10: file and pop pairs per writer
	scaleProject    = "g/po-1"
)

// scaleCase is one tracker size measured alone and under one number of
// concurrent writers.
type scaleCase struct {
	Issues       int                `json:"issues"`
	FileBytes    int                `json:"issues_file_bytes"`
	Sequential   map[string]float64 `json:"sequential_seconds"` // median of three runs per command
	Writers      int                `json:"writers"`
	Ops          int                `json:"ops"`
	Failed       int                `json:"failed"`
	Errors       []string           `json:"errors,omitempty"`
	Latency      map[string]float64 `json:"contended_p50_seconds"`
	LatencyP95   map[string]float64 `json:"contended_p95_seconds"`
	WallSeconds  float64            `json:"contended_wall_seconds"`
	Lost         int                `json:"lost_writes"`
	Duplicates   int                `json:"duplicate_ids"`
	RepoKB       int                `json:"tracker_repository_kb"`
	CloneSeconds float64            `json:"fresh_clone_seconds"`
}

// BeadsScale builds tpm, seeds a tracker repository with thousands of issues, times the common commands against it, runs concurrent writers filing and popping issues, and reports command times, failures, the tracker's size, and a fresh clone's time; it fails only on a lost write or a duplicate ID (GH-129). Opt-in: set BEADS_SCALE_ISSUES, BEADS_SCALE_WRITERS, and BEADS_SCALE_OPS to change the sizes.
func (Integration) BeadsScale() error {
	root, err := repositoryRoot()
	if err != nil {
		return err
	}
	issues, err := scaleSizes(scaleIssuesEnv, "1000,5000,10000")
	if err != nil {
		return err
	}
	writers, err := scaleSizes(scaleWritersEnv, "2,5,10")
	if err != nil {
		return err
	}
	ops, err := scaleSizes(scaleOpsEnv, "10")
	if err != nil {
		return err
	}
	tmp, err := os.MkdirTemp("", "tpm-beads-scale-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	if tmp, err = filepath.EvalSymlinks(tmp); err != nil {
		return err
	}
	bin := filepath.Join(tmp, "tpm")
	if err := runStreaming(root, "go", "build", "-o", bin, "./cmd/tpm"); err != nil {
		return fmt.Errorf("build tpm: %w", err)
	}
	var cases []scaleCase
	var failures []string
	stamp := time.Now().UTC().Format("20060102T150405Z")
	for _, n := range issues {
		for _, k := range writers {
			c, err := scaleRun(filepath.Join(tmp, fmt.Sprintf("n%d-k%d", n, k)), bin, n, k, ops[0])
			if err != nil {
				return err
			}
			cases = append(cases, c)
			fmt.Println(scaleRow(c))
			for _, e := range c.Errors {
				fmt.Println("  failed:", e)
			}
			// The report is rewritten after every case, so a run cut short
			// keeps what it measured.
			if err := writeScaleReport(root, stamp, cases); err != nil {
				return err
			}
			if c.Lost > 0 || c.Duplicates > 0 {
				failures = append(failures, fmt.Sprintf("%d issues, %d writers: %d lost writes, %d duplicate IDs", n, k, c.Lost, c.Duplicates))
			}
		}
	}
	fmt.Println("report:", filepath.Join(root, rigStateDir, "beads-scale", stamp+".md"))
	if len(failures) > 0 {
		return fmt.Errorf("integration:beadsScale failed:\n  %s", strings.Join(failures, "\n  "))
	}
	fmt.Println("integration:beadsScale passed")
	return nil
}

func scaleSizes(env, def string) ([]int, error) {
	v := strings.TrimSpace(os.Getenv(env))
	if v == "" {
		v = def
	}
	var out []int
	for _, f := range strings.Split(v, ",") {
		n, err := strconv.Atoi(strings.TrimSpace(f))
		if err != nil || n <= 0 {
			return nil, fmt.Errorf("%s=%q: want positive integers separated by commas", env, v)
		}
		out = append(out, n)
	}
	return out, nil
}

// scaleRun seeds a tracker of n issues under dir, times the commands with
// one writer, then runs k writers doing ops file-and-pop pairs each, and
// checks the tracker afterwards.
func scaleRun(dir, bin string, n, k, ops int) (scaleCase, error) {
	c := scaleCase{Issues: n, Writers: k, Ops: k * ops * 2, Sequential: map[string]float64{},
		Latency: map[string]float64{}, LatencyP95: map[string]float64{}}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return c, err
	}
	git := func(in string, args ...string) error {
		cmd := exec.Command("git", args...)
		cmd.Dir = in
		cmd.Env = scaleEnv("seed")
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, out)
		}
		return nil
	}
	tracker, code := filepath.Join(dir, "tracker.git"), filepath.Join(dir, "po-1.git")
	for _, bare := range []string{tracker, code} {
		if err := git(dir, "init", "-q", "--bare", "-b", "main", bare); err != nil {
			return c, err
		}
	}
	// The code repository holds one commit; pop branches from it.
	seedCode := filepath.Join(dir, "code-seed")
	if err := git(dir, "clone", "-q", code, seedCode); err != nil {
		return c, err
	}
	os.WriteFile(filepath.Join(seedCode, "README.md"), []byte("# po-1\n"), 0o644)
	for _, args := range [][]string{{"add", "-A"}, {"commit", "-q", "-m", "first"}, {"push", "-q", "origin", "HEAD:main"}} {
		if err := git(seedCode, args...); err != nil {
			return c, err
		}
	}
	// The tracker's first commit holds the n generated issues.
	seed := filepath.Join(dir, "tracker-seed")
	if err := git(dir, "clone", "-q", tracker, seed); err != nil {
		return c, err
	}
	reserved := k * ops * 2
	size, err := writeScaleIssues(filepath.Join(seed, beads.IssuesFile), n, reserved)
	if err != nil {
		return c, err
	}
	c.FileBytes = size
	for _, args := range [][]string{{"add", "-A"}, {"commit", "-q", "-m", "seed"}, {"push", "-q", "origin", "HEAD:main"}} {
		if err := git(seed, args...); err != nil {
			return c, err
		}
	}

	// Every writer, the sequential one first, has its own code checkout,
	// named for the project as tpm expects, and its own tracker clone.
	writer := func(name string) (string, []string, error) {
		home := filepath.Join(dir, name)
		checkout := filepath.Join(home, "po-1")
		if err := os.MkdirAll(home, 0o755); err != nil {
			return "", nil, err
		}
		// --no-local copies objects instead of hardlinking them, so a clone
		// made while others push is not torn.
		if err := git(home, "clone", "-q", "--no-local", code, checkout); err != nil {
			return "", nil, err
		}
		if err := git(home, "clone", "-q", "--no-local", tracker, filepath.Join(home, "tracker-clone")); err != nil {
			return "", nil, err
		}
		env := append(scaleEnv(name), "TPM_TRACKER="+tracker, "TPM_TRACKER_CLONE="+filepath.Join(home, "tracker-clone"), "TPM_USER="+name)
		return checkout, env, nil
	}
	body := filepath.Join(dir, "body.md")
	os.WriteFile(body, []byte(scaleBody), 0o644)
	run := func(checkout string, env []string, args ...string) (time.Duration, error) {
		cmd := exec.Command(bin, args...)
		cmd.Dir, cmd.Env = checkout, env
		start := time.Now()
		out, err := cmd.CombinedOutput()
		if err != nil {
			return time.Since(start), fmt.Errorf("tpm %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
		}
		return time.Since(start), nil
	}
	file := func(title string) []string {
		return []string{"issue", "file", "--json", "--project", scaleProject, "--title", title, "--body-file", body,
			"--plane", "data", "--kind", "build", "--component", "po-1", "--milestone", "01.0"}
	}

	// Sequential: each command three times, the median kept. The pops use
	// the last reserved issues, which no writer takes.
	solo, env, err := writer("solo")
	if err != nil {
		return c, err
	}
	commands := map[string]func(i int) []string{
		"issue file": func(i int) []string { return file(fmt.Sprintf("solo %d", i)) },
		"issue pop": func(i int) []string {
			return []string{"issue", "pop", "--json", "--project", scaleProject, "--issue", strconv.Itoa(n - i), "--dir", solo}
		},
		"issue lint --since --dry-run": func(int) []string {
			return []string{"issue", "lint", "--json", "--dry-run", "--project", scaleProject, "--since", "2026-01-01"}
		},
		"sprint-end facts": func(int) []string {
			return []string{"sprint-end", "facts", "--json", "--project", scaleProject, "--reports", filepath.Join(dir, "reports"),
				"--from", "2026-W43", "--to", "2026-W43", "--artifact", "po-1"}
		},
	}
	if err := scaleReports(filepath.Join(dir, "reports"), bin, solo, env); err != nil {
		return c, err
	}
	for name, args := range commands {
		var times []float64
		for i := range 3 {
			d, err := run(solo, env, args(i)...)
			if err != nil {
				return c, fmt.Errorf("%d issues, sequential %s: %w", n, name, err)
			}
			times = append(times, d.Seconds())
		}
		sort.Float64s(times)
		c.Sequential[name] = times[1]
	}

	// Contended: k writers at once, each filing new issues and popping its
	// own reserved ones.
	type sample struct {
		op  string
		d   time.Duration
		err error
	}
	var mu sync.Mutex
	var samples []sample
	var filed []string
	// Every writer is set up before any starts writing.
	type seat struct {
		name, checkout string
		env            []string
	}
	var seats []seat
	for w := range k {
		name := fmt.Sprintf("writer-%d", w)
		checkout, env, err := writer(name)
		if err != nil {
			return c, err
		}
		seats = append(seats, seat{name, checkout, env})
	}
	var wg sync.WaitGroup
	start := time.Now()
	for w, s := range seats {
		name, checkout, env := s.name, s.checkout, s.env
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range ops {
				title := fmt.Sprintf("%s filed %d", name, j)
				d, err := run(checkout, env, file(title)...)
				mu.Lock()
				samples = append(samples, sample{"issue file", d, err})
				if err == nil {
					filed = append(filed, title)
				}
				mu.Unlock()
				iid := 1 + w*ops*2 + j
				d, err = run(checkout, env, "issue", "pop", "--json", "--project", scaleProject, "--issue", strconv.Itoa(iid), "--dir", checkout)
				mu.Lock()
				samples = append(samples, sample{"issue pop", d, err})
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	c.WallSeconds = time.Since(start).Seconds()
	byOp := map[string][]float64{}
	for _, s := range samples {
		if s.err != nil {
			c.Failed++
			if len(c.Errors) < 5 {
				c.Errors = append(c.Errors, s.err.Error())
			}
			continue
		}
		byOp[s.op] = append(byOp[s.op], s.d.Seconds())
	}
	for op, ts := range byOp {
		sort.Float64s(ts)
		c.Latency[op] = ts[len(ts)/2]
		c.LatencyP95[op] = ts[min(len(ts)-1, len(ts)*95/100)]
	}

	// Correctness: every filed issue is in the tracker once, and no ID or
	// IID repeats.
	out, err := exec.Command("git", "--git-dir", tracker, "show", "main:"+beads.IssuesFile).Output()
	if err != nil {
		return c, fmt.Errorf("read the tracker after the run: %w", err)
	}
	ids, iids, titles := map[string]int{}, map[int]int{}, map[string]int{}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		var i beads.Issue
		if err := json.Unmarshal([]byte(line), &i); err != nil {
			return c, fmt.Errorf("the tracker after the run: %w", err)
		}
		ids[i.ID]++
		iids[i.IID]++
		titles[i.Title]++
	}
	for _, n := range ids {
		c.Duplicates += n - 1
	}
	for _, n := range iids {
		c.Duplicates += n - 1
	}
	for _, t := range filed {
		if titles[t] != 1 {
			c.Lost++
		}
	}

	// The tracker repository's size, packed as a remote would hold it, and
	// a fresh clone's time.
	if err := git(dir, "--git-dir", tracker, "gc", "-q"); err != nil {
		return c, err
	}
	if kb, err := dirKB(tracker); err == nil {
		c.RepoKB = kb
	}
	cloneStart := time.Now()
	if err := git(dir, "clone", "-q", tracker, filepath.Join(dir, "fresh")); err != nil {
		return c, err
	}
	c.CloneSeconds = time.Since(cloneStart).Seconds()
	return c, nil
}

// scaleEnv is the environment of one person's git and tpm runs.
func scaleEnv(user string) []string {
	return append(os.Environ(), "GIT_AUTHOR_NAME="+user, "GIT_AUTHOR_EMAIL="+user+"@example.invalid",
		"GIT_COMMITTER_NAME="+user, "GIT_COMMITTER_EMAIL="+user+"@example.invalid")
}

const scaleBody = `Score one agent's trajectories against the written rubric and store the score with the agent's version.

## Requirements

1. Read the agent's trajectories for the run.
2. Score each against the rubric.
3. Store the scores with the agent version and the rubric version.

## Acceptance Criteria

- One agent's trajectories are scored and stored with its version (test: grader_scores_fixture_agent; environment: local)
`

// writeScaleIssues writes n issues to path, the first reserved of them
// ready for pops, a fifth of the rest closed, each with five comments the
// size of ledger events, and returns the file's size.
func writeScaleIssues(path string, n, reserved int) (int, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return 0, err
	}
	var lines []string
	for iid := 1; iid <= n; iid++ {
		id := fmt.Sprintf("tpm-%08x", iid*2654435761%(1<<32))
		i := beads.Issue{ID: id, IID: iid, Repository: scaleProject, Title: fmt.Sprintf("Seeded issue %d", iid), Description: scaleBody,
			Status: "open", State: beads.Ready, Labels: []string{"plane::data", "kind::build", "component::po-1", "op::file"},
			Release: "01.0", CreatedAt: "2026-10-01T09:00:00Z", UpdatedAt: "2026-10-02T09:00:00Z", CreatedBy: "seed"}
		if iid > reserved && iid <= n-3 && iid%5 == 0 {
			i.Status, i.State, i.ClosedAt = "closed", beads.Closed, "2026-10-03T09:00:00Z"
		}
		for e := range 5 {
			i.Comments = append(i.Comments, beads.Comment{ID: fmt.Sprintf("%s-c%d", id, e), IssueID: id, Author: "seed",
				CreatedAt: "2026-10-02T09:00:00Z",
				Text:      fmt.Sprintf("Planned for release 01.0; evidence counts in local for 1 requirement(s).\n\n<!-- tpm program event=planned release=01.0 requirement=https://example.invalid/g/system/-/blob/corpus/v0.1.0/docs/specs/initiatives/in001-x.md#R1.%d environment=local -->", e)})
		}
		b, err := json.Marshal(i)
		if err != nil {
			return 0, err
		}
		lines = append(lines, string(b))
	}
	slices.Sort(lines) // the store keeps the file sorted by ID, and every ID starts its line
	data := []byte(strings.Join(lines, "\n") + "\n")
	return len(data), os.WriteFile(path, data, 0o644)
}

// scaleReports lays down the reports checkout sprint-end facts reads.
func scaleReports(dir, bin, cwd string, env []string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	cmd := exec.Command(bin, "reports", "init", "--dir", dir, "--system", "g/system", "--member", "solo="+scaleProject)
	cmd.Dir, cmd.Env = cwd, env
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("tpm reports init: %w: %s", err, out)
	}
	return nil
}

func dirKB(dir string) (int, error) {
	var total int64
	err := filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			total += info.Size()
		}
		return err
	})
	return int(total / 1024), err
}

var scaleOps = []string{"issue file", "issue pop", "issue lint --since --dry-run", "sprint-end facts"}

func scaleRow(c scaleCase) string {
	var seq []string
	for _, op := range scaleOps {
		seq = append(seq, fmt.Sprintf("%s %.2fs", op, c.Sequential[op]))
	}
	return fmt.Sprintf("%d issues (%d KB file), %d writers: sequential [%s]; contended file p50 %.2fs p95 %.2fs, pop p50 %.2fs p95 %.2fs, wall %.1fs, %d/%d failed; repository %d KB, clone %.2fs; lost %d, duplicates %d",
		c.Issues, c.FileBytes/1024, c.Writers, strings.Join(seq, ", "), c.Latency["issue file"], c.LatencyP95["issue file"],
		c.Latency["issue pop"], c.LatencyP95["issue pop"], c.WallSeconds, c.Failed, c.Ops, c.RepoKB, c.CloneSeconds, c.Lost, c.Duplicates)
}

// writeScaleReport keeps the run's numbers under .regression, as JSON and
// as a Markdown table.
func writeScaleReport(root, stamp string, cases []scaleCase) error {
	dir := filepath.Join(root, rigStateDir, "beads-scale")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(cases, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, stamp+".json"), b, 0o644); err != nil {
		return err
	}
	var md strings.Builder
	md.WriteString("| issues | file KB | writers | file | pop | lint --since | facts | file p50/p95 contended | pop p50/p95 contended | wall s | failed | repo KB | clone s | lost | dup |\n")
	md.WriteString("| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |\n")
	for _, c := range cases {
		fmt.Fprintf(&md, "| %d | %d | %d | %.2f | %.2f | %.2f | %.2f | %.2f / %.2f | %.2f / %.2f | %.1f | %d/%d | %d | %.2f | %d | %d |\n",
			c.Issues, c.FileBytes/1024, c.Writers, c.Sequential["issue file"], c.Sequential["issue pop"], c.Sequential["issue lint --since --dry-run"],
			c.Sequential["sprint-end facts"], c.Latency["issue file"], c.LatencyP95["issue file"], c.Latency["issue pop"], c.LatencyP95["issue pop"],
			c.WallSeconds, c.Failed, c.Ops, c.RepoKB, c.CloneSeconds, c.Lost, c.Duplicates)
	}
	return os.WriteFile(filepath.Join(dir, stamp+".md"), []byte(md.String()), 0o644)
}
