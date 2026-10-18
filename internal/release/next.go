package release

import (
	"strings"

	"github.com/gray-lake-labs/scrum-master/internal/program"
)

// Candidate is one version tpm release next offers.
type Candidate struct {
	Bump    string `json:"bump"`
	Version string `json:"version"`
	Tag     string `json:"tag"`
	Class   string `json:"class"`
}

// NextResult is what tpm release next reports (srd015 R3.1).
type NextResult struct {
	Artifact   string      `json:"artifact"`
	Latest     string      `json:"latest,omitempty"`
	LatestTag  string      `json:"latest_tag,omitempty"`
	Candidates []Candidate `json:"candidates"`
}

// Next reports the artifact's latest version among tags and the next
// patch, minor, and major versions with their class. An artifact with no
// tag starts from 0.0.0.
func Next(r program.Releases, artifact string, tags []string) NextResult {
	out := NextResult{Artifact: artifact}
	latest, ok := Latest(r.TagFormat, artifact, tags)
	if ok {
		out.Latest, out.LatestTag = latest.String(), TagName(r.TagFormat, artifact, latest)
	}
	for _, bump := range []string{"patch", "minor", "major"} {
		v := latest.Bump(bump)
		out.Candidates = append(out.Candidates, Candidate{bump, v.String(), TagName(r.TagFormat, artifact, v), Class(r.Breaking, latest, v)})
	}
	return out
}

// Human is the text tpm release next prints without --json.
func (n NextResult) Human() string {
	var b strings.Builder
	if n.Latest == "" {
		b.WriteString(n.Artifact + ": no release yet\n")
	} else {
		b.WriteString(n.Artifact + ": latest " + n.Latest + " (" + n.LatestTag + ")\n")
	}
	for _, c := range n.Candidates {
		b.WriteString("  " + c.Bump + " " + c.Version + " " + c.Class + "\n")
	}
	return b.String()
}
