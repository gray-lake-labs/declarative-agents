// Copyright (c) 2026 Gray Lake Labs
// SPDX-License-Identifier: MIT-0
package runtime

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	goruntime "runtime"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/spectool"
	"github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/version"
)

const tag = "v0.20260101.1"

func sumLine(data []byte, asset string) string {
	s := sha256.Sum256(data)
	return hex.EncodeToString(s[:]) + "  " + asset + "\n"
}

// sourceArchive is a release's source as GitHub serves it: one top
// directory holding agent-core/go.mod.
func sourceArchive(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	body := []byte("module example.invalid/agent-core\n")
	if err := tw.WriteHeader(&tar.Header{Name: "declarative-agents-" + tag + "/agent-core/go.mod", Mode: 0o644, Size: int64(len(body))}); err != nil {
		t.Fatal(err)
	}
	tw.Write(body)
	tw.Close()
	gz.Close()
	return buf.Bytes()
}

// release serves a fixture release from a local server, counting requests,
// in a cache directory of the test's own; the agent it serves is agent.
type release struct {
	asset    string
	agent    []byte
	requests atomic.Int32
	serve    map[string][]byte
}

func fixture(t *testing.T) *release {
	t.Helper()
	asset, err := HostAsset("agent", goruntime.GOOS, goruntime.GOARCH)
	if err != nil {
		t.Skip(err)
	}
	cache := t.TempDir()
	t.Setenv("HOME", cache)
	t.Setenv("XDG_CACHE_HOME", cache)
	t.Setenv(AgentEnv, "")
	r := &release{asset: asset, agent: []byte("the agent")}
	r.serve = map[string][]byte{
		"/releases/download/" + tag + "/SHA256SUMS": []byte(sumLine([]byte("other"), "agent_plan9_386") + sumLine(r.agent, asset)),
		"/releases/download/" + tag + "/" + asset:   r.agent,
		"/archive/refs/tags/" + tag + ".tar.gz":     sourceArchive(t),
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r.requests.Add(1)
		data, ok := r.serve[req.URL.Path]
		if !ok {
			http.NotFound(w, req)
			return
		}
		w.Write(data)
	}))
	t.Cleanup(srv.Close)
	prev := spectool.ReleaseURL
	spectool.ReleaseURL = srv.URL
	t.Cleanup(func() { spectool.ReleaseURL = prev })
	return r
}

func TestReleaseBinaryChecksumIsVerified(t *testing.T) {
	good := []byte("the agent")
	sums := []byte(sumLine([]byte("other"), "spectool_linux_amd64") + sumLine(good, "agent_linux_amd64"))
	if err := VerifyChecksum(sums, "agent_linux_amd64", good); err != nil {
		t.Fatalf("a matching binary was refused: %v", err)
	}
	if err := VerifyChecksum(sums, "agent_linux_amd64", []byte("tampered")); err == nil || !strings.Contains(err.Error(), "refusing") {
		t.Fatalf("a binary whose checksum differs was accepted: %v", err)
	}
	if err := VerifyChecksum(sums, "agent_darwin_arm64", good); err == nil || !strings.Contains(err.Error(), "lists no") {
		t.Fatalf("a binary SHA256SUMS does not list was accepted: %v", err)
	}
}

func TestHostAssetNamesUnpublishedHosts(t *testing.T) {
	for _, h := range [][2]string{{"darwin", "arm64"}, {"darwin", "amd64"}, {"linux", "arm64"}, {"linux", "amd64"}} {
		if got, err := HostAsset("agent", h[0], h[1]); err != nil || got != "agent_"+h[0]+"_"+h[1] {
			t.Errorf("%s/%s: %q, %v", h[0], h[1], got, err)
		}
	}
	_, err := HostAsset("agent", "windows", "amd64")
	if err == nil || !strings.Contains(err.Error(), "windows/amd64") || !strings.Contains(err.Error(), "darwin and linux on amd64 and arm64") {
		t.Fatalf("an unpublished host was not named with the published ones: %v", err)
	}
}

func TestFetchLaysOutTheReleaseOnce(t *testing.T) {
	f := fixture(t)
	rep, err := Fetch(tag, false)
	if err != nil {
		t.Fatal(err)
	}
	rel, _ := Locate(tag)
	if rep.Agent != rel.Agent || rep.From != FromCache || rep.CoreRoot != filepath.Join(rel.Src, "agent-core") {
		t.Fatalf("report: %+v", rep)
	}
	if want := []string{"SHA256SUMS", f.asset, "source"}; !slices.Equal(rep.Downloaded, want) {
		t.Fatalf("downloaded %v, want %v", rep.Downloaded, want)
	}
	if data, err := os.ReadFile(rel.Agent); err != nil || !bytes.Equal(data, f.agent) {
		t.Fatalf("agent: %q, %v", data, err)
	}
	if info, _ := os.Stat(rel.Agent); info.Mode()&0o100 == 0 {
		t.Fatalf("the agent is not executable: %v", info.Mode())
	}
	if _, err := os.Stat(filepath.Join(rel.CoreRoot(), "go.mod")); err != nil {
		t.Fatalf("source: %v", err)
	}
	if !strings.HasPrefix(rel.Dir, os.Getenv("XDG_CACHE_HOME")) && !strings.HasPrefix(rel.Dir, os.Getenv("HOME")) {
		t.Fatalf("the release is not under the user cache directory: %s", rel.Dir)
	}
	dir, _ := spectool.ReleaseDir(tag)
	if rel.Dir != dir {
		t.Fatalf("the runtime is at %s, but the spec tool and mage audit use %s", rel.Dir, dir)
	}
	before := f.requests.Load()
	again, err := Fetch(tag, false)
	if err != nil || len(again.Downloaded) != 0 || f.requests.Load() != before {
		t.Fatalf("a second fetch downloaded %v (%d requests), %v", again.Downloaded, f.requests.Load()-before, err)
	}
}

func TestFetchDryRunWritesNothing(t *testing.T) {
	f := fixture(t)
	rep, err := Fetch(tag, true)
	if err != nil || len(rep.Downloaded) != 3 || !rep.DryRun {
		t.Fatalf("dry run: %+v, %v", rep, err)
	}
	rel, _ := Locate(tag)
	if _, err := os.Stat(rel.Dir); !errors.Is(err, os.ErrNotExist) || f.requests.Load() != 0 {
		t.Fatalf("a dry run wrote %s or downloaded (%d requests): %v", rel.Dir, f.requests.Load(), err)
	}
}

func TestFetchReplacesATamperedCache(t *testing.T) {
	f := fixture(t)
	if _, err := Fetch(tag, false); err != nil {
		t.Fatal(err)
	}
	rel, _ := Locate(tag)
	os.WriteFile(rel.Agent, []byte("tampered"), 0o755)
	rep, err := Fetch(tag, false)
	if err != nil || !slices.Equal(rep.Downloaded, []string{f.asset}) {
		t.Fatalf("refetch: %v, %v", rep.Downloaded, err)
	}
	if data, _ := os.ReadFile(rel.Agent); !bytes.Equal(data, f.agent) {
		t.Fatalf("the tampered agent was kept: %q", data)
	}
}

func TestFetchRefusesATamperedDownload(t *testing.T) {
	f := fixture(t)
	f.serve["/releases/download/"+tag+"/"+f.asset] = []byte("tampered in transit")
	_, err := Fetch(tag, false)
	if err == nil || !strings.Contains(err.Error(), f.asset) || strings.Count(err.Error(), "checksum")+strings.Count(err.Error(), "lists") < 2 {
		t.Fatalf("a tampered download was not refused naming both checksums: %v", err)
	}
	rel, _ := Locate(tag)
	if _, err := os.Stat(rel.Agent); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a refused agent was written: %v", err)
	}
}

func TestTPMAgentIsHeldToTheRelease(t *testing.T) {
	f := fixture(t)
	own := filepath.Join(t.TempDir(), "agent")
	os.WriteFile(own, f.agent, 0o755)
	t.Setenv(AgentEnv, own)
	rep, err := Fetch(tag, false)
	if err != nil || rep.Agent != own || rep.From != FromEnv || slices.Contains(rep.Downloaded, f.asset) {
		t.Fatalf("the release's binary in TPM_AGENT: %+v, %v", rep, err)
	}
	if s := Inspect(tag); s.Agent != own || s.From != FromEnv || s.Problem != "" {
		t.Fatalf("inspect with TPM_AGENT: %+v", s)
	}
	other := filepath.Join(t.TempDir(), "agent")
	os.WriteFile(other, []byte("a locally built agent"), 0o755)
	t.Setenv(AgentEnv, other)
	if _, err := Fetch(tag, false); err == nil || !strings.Contains(err.Error(), tag) {
		t.Fatalf("another binary in TPM_AGENT was not refused naming %s: %v", tag, err)
	}
	if s := Inspect(tag); s.Agent != "" || !strings.Contains(s.Problem, tag) {
		t.Fatalf("inspect with another binary: %+v", s)
	}
}

func TestTPMAgentOfflineNamesTheMissingChecksums(t *testing.T) {
	f := fixture(t)
	spectool.ReleaseURL = "http://127.0.0.1:1"
	own := filepath.Join(t.TempDir(), "agent")
	os.WriteFile(own, f.agent, 0o755)
	t.Setenv(AgentEnv, own)
	if _, err := Fetch(tag, false); err == nil || !strings.Contains(err.Error(), "checksums of "+tag) {
		t.Fatalf("offline with no cached checksums: %v", err)
	}
}

func TestInspectFetchesNothing(t *testing.T) {
	f := fixture(t)
	s := Inspect(tag)
	if s.Agent != "" || !strings.Contains(s.Problem, "tpm install --runtime") || f.requests.Load() != 0 {
		t.Fatalf("inspect before a fetch: %+v, %d requests", s, f.requests.Load())
	}
	if _, err := Fetch(tag, false); err != nil {
		t.Fatal(err)
	}
	before := f.requests.Load()
	rel, _ := Locate(tag)
	if s := Inspect(tag); s.Agent != rel.Agent || s.From != FromCache || s.Problem != "" {
		t.Fatalf("inspect after a fetch: %+v", s)
	}
	os.WriteFile(rel.Agent, []byte("tampered"), 0o755)
	if s := Inspect(tag); s.Agent != "" || !strings.Contains(s.Problem, "tpm install --runtime") {
		t.Fatalf("inspect of a tampered cache: %+v", s)
	}
	if f.requests.Load() != before {
		t.Fatalf("inspect downloaded %d times", f.requests.Load()-before)
	}
}

func TestBuildSpecToolNeedsGoOrAPrebuiltTool(t *testing.T) {
	fixture(t)
	prev := lookGo
	t.Cleanup(func() { lookGo = prev })
	rel, _ := Locate(tag)

	lookGo = func() error { return errors.New("not found") }
	rep := Report{Release: tag}
	if err := BuildSpecTool(&rep); err != nil || rep.SpecTool != "" || !strings.Contains(rep.Skipped, "no Go toolchain") {
		t.Fatalf("without Go: %+v, %v", rep, err)
	}

	lookGo = func() error { return nil }
	rep = Report{Release: tag, DryRun: true}
	if err := BuildSpecTool(&rep); err != nil || !rep.Built || rep.SpecTool != spectool.Binary(rel.Dir) {
		t.Fatalf("a dry run with Go: %+v, %v", rep, err)
	}
	if Built(tag) {
		t.Fatal("a dry run built the spec tool")
	}

	lookGo = func() error { return errors.New("not found") }
	writeFile(spectool.Binary(rel.Dir), []byte("built"), 0o755)
	rep = Report{Release: tag}
	if err := BuildSpecTool(&rep); err != nil || rep.Built || rep.SpecTool != spectool.Binary(rel.Dir) || rep.Skipped != "" || !Built(tag) {
		t.Fatalf("a prebuilt tool without Go: %+v, %v", rep, err)
	}
}

// A dev build pins the newest release with published binaries, and a
// release tag is the shape DefaultPin takes from a released build (GH-190).
func TestDefaultPinIsAReleaseTag(t *testing.T) {
	if pin := DefaultPin(); !version.IsRelease(pin) {
		t.Fatalf("DefaultPin %q is not a release tag", pin)
	}
}

// Inside this repository the sibling is the runtime: the source is the
// repository root holding agent-core, and nothing downloads (GH-190).
func TestSiblingLocatesThisRepository(t *testing.T) {
	rel, ok, err := Sibling(".", nil)
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if _, serr := os.Stat(filepath.Join(rel.CoreRoot(), "cmd", "agent")); serr != nil {
		t.Fatalf("core root %q has no agent: %v", rel.CoreRoot(), serr)
	}
	if _, ok, _ := Sibling(t.TempDir(), nil); ok {
		t.Fatal("a directory outside the repository must not be a sibling")
	}
}

// A published tpm pins its own release: the stamp, not the build info,
// decides DefaultPin, so a program repository a release lays down pins
// that release's runtime (GH-18).
func TestDefaultPinFollowsTheStampedRelease(t *testing.T) {
	saved := version.Release
	defer func() { version.Release = saved }()
	version.Release = "v0.20990101.4"
	if got := DefaultPin(); got != "v0.20990101.4" {
		t.Fatalf("DefaultPin = %q, want the stamped release", got)
	}
}
