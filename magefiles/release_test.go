package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sumLine(data []byte, asset string) string {
	s := sha256.Sum256(data)
	return hex.EncodeToString(s[:]) + "  " + asset + "\n"
}

func TestReleaseBinaryChecksumIsVerified(t *testing.T) {
	good := []byte("the agent")
	sums := []byte(sumLine([]byte("other"), "spectool_linux_amd64") + sumLine(good, "agent_linux_amd64"))
	if err := verifyChecksum(sums, "agent_linux_amd64", good); err != nil {
		t.Fatalf("a matching binary was refused: %v", err)
	}
	if err := verifyChecksum(sums, "agent_linux_amd64", []byte("tampered")); err == nil || !strings.Contains(err.Error(), "refusing") {
		t.Fatalf("a binary whose checksum differs was accepted: %v", err)
	}
	if err := verifyChecksum(sums, "agent_darwin_arm64", good); err == nil || !strings.Contains(err.Error(), "lists no") {
		t.Fatalf("a binary SHA256SUMS does not list was accepted: %v", err)
	}
}

func TestHostAssetNamesUnpublishedHosts(t *testing.T) {
	for _, h := range [][2]string{{"darwin", "arm64"}, {"darwin", "amd64"}, {"linux", "arm64"}, {"linux", "amd64"}} {
		if got, err := hostAsset("agent", h[0], h[1]); err != nil || got != "agent_"+h[0]+"_"+h[1] {
			t.Errorf("%s/%s: %q, %v", h[0], h[1], got, err)
		}
	}
	if _, err := hostAsset("agent", "windows", "amd64"); err == nil || !strings.Contains(err.Error(), "windows/amd64") {
		t.Fatalf("an unpublished host was not named: %v", err)
	}
}

func TestCheckoutsFileLastsOnlyForTheRun(t *testing.T) {
	root := t.TempDir()
	restore, err := withCheckouts(root, "/cache/src", parentRepository)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(root, checkoutsFile))
	if !strings.Contains(string(b), parentRepository+": /cache/src") {
		t.Fatalf("checkouts during the run:\n%s", b)
	}
	restore()
	if _, err := os.Stat(filepath.Join(root, checkoutsFile)); !os.IsNotExist(err) {
		t.Fatalf("the checkouts file outlived the run: %v", err)
	}
	writeFile(t, filepath.Join(root, checkoutsFile), "checkouts: {}\n")
	restore, _ = withCheckouts(root, "/cache/src", parentRepository)
	restore()
	if b, _ := os.ReadFile(filepath.Join(root, checkoutsFile)); string(b) != "checkouts: {}\n" {
		t.Fatalf("a person's checkouts file was changed: %q", b)
	}
}
