package sprintgoal

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// SummaryDir holds a repository's sprint-end summaries, one per release.
const SummaryDir = "docs/sprint-ends"

// storiesHeading starts the section of a sprint-end summary that records the
// listed stories the owner reports done (srd038 R2.4).
const storiesHeading = "## Stories done"

// backlogHeading starts a Sprint Goal's Sprint Backlog section (config
// format issue-body sprint_backlog_section).
const backlogHeading = "## Sprint Backlog"

// SprintBacklog returns the story ids a Sprint Goal's Sprint Backlog lists,
// in order: the first word of each "- STORY title (N points)" line.
func SprintBacklog(description string) []string {
	_, rest, ok := strings.Cut(description, backlogHeading)
	if !ok {
		return nil
	}
	var out []string
	for _, line := range strings.Split(rest, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "## ") || strings.HasPrefix(line, "<!--") {
			break
		}
		if item, ok := strings.CutPrefix(line, "- "); ok {
			if f := strings.Fields(item); len(f) > 0 {
				out = append(out, f[0])
			}
		}
	}
	return out
}

// RecordStoriesDone writes the Stories done section of the summary at path,
// replacing any earlier one, and reports whether the file changed; a dry
// run writes nothing. The summary must exist: the owner's summary comes
// first, the record of what was done with it.
func RecordStoriesDone(path string, ids []string, dryRun bool) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	text := string(data)
	if i := strings.Index(text, "\n"+storiesHeading+"\n"); i >= 0 {
		rest := text[i+1+len(storiesHeading)+1:]
		end := len(rest)
		if j := strings.Index(rest, "\n## "); j >= 0 {
			end = j + 1
		}
		text = text[:i+1] + rest[end:]
	}
	text = strings.TrimRight(text, "\n") + "\n\n" + storiesHeading + "\n\n"
	if len(ids) == 0 {
		text += "None.\n"
	}
	for _, id := range ids {
		text += "- " + id + "\n"
	}
	if text == string(data) {
		return false, nil
	}
	if dryRun {
		return true, nil
	}
	return true, os.WriteFile(path, []byte(text), 0o644)
}

// Done is the stories a repository's sprint-end summaries record as done.
type Done struct {
	// Release maps each done story to the release whose summary records it.
	Release map[string]string
	// Latest is the newest release whose summary has a Stories done section,
	// and Last the stories it records; velocity is read from them.
	Latest string
	Last   []string
}

// StoriesDone reads every summary under SummaryDir in the checkout at dir.
func StoriesDone(dir string) (Done, error) {
	d := Done{Release: map[string]string{}}
	paths, err := filepath.Glob(filepath.Join(dir, SummaryDir, "*.md"))
	if err != nil {
		return d, err
	}
	sort.Slice(paths, func(i, j int) bool { return versionLess(releaseOf(paths[i]), releaseOf(paths[j])) })
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			return d, err
		}
		_, rest, ok := strings.Cut(string(data), "\n"+storiesHeading+"\n")
		if !ok {
			continue
		}
		var ids []string
		for _, line := range strings.Split(rest, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "## ") {
				break
			}
			if id, ok := strings.CutPrefix(line, "- "); ok {
				ids = append(ids, id)
				d.Release[id] = releaseOf(p)
			}
		}
		d.Latest, d.Last = releaseOf(p), ids
	}
	return d, nil
}

func releaseOf(path string) string { return strings.TrimSuffix(filepath.Base(path), ".md") }

// versionLess orders releases like 01.2 before 01.10 by their numbers.
func versionLess(a, b string) bool {
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(pa) && i < len(pb); i++ {
		na, errA := strconv.Atoi(pa[i])
		nb, errB := strconv.Atoi(pb[i])
		if errA != nil || errB != nil {
			return a < b
		}
		if na != nb {
			return na < nb
		}
	}
	return len(pa) < len(pb)
}
