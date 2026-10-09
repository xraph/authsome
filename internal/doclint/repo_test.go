package doclint

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// No environment file is tracked: an .env holds whatever a developer put
// in it, and the example file is the only shape that belongs in git.
func TestNoTrackedEnvFiles(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(context.Background(), "git", "ls-files")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		t.Skipf("git ls-files unavailable: %v", err)
	}
	var tracked []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		base := filepath.Base(line)
		if base == ".env" || (strings.HasPrefix(base, ".env.") && base != ".env.example") {
			tracked = append(tracked, line)
		}
	}
	if len(tracked) > 0 {
		t.Errorf("environment files must not be tracked:\n  %s", strings.Join(tracked, "\n  "))
	}
}
