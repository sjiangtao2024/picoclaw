package scripts_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyncUpstreamReleaseCreatesWorktreeFromTag(t *testing.T) {
	tmpDir := t.TempDir()
	upstreamBare := filepath.Join(tmpDir, "upstream.git")
	originBare := filepath.Join(tmpDir, "origin.git")
	seedRepo := filepath.Join(tmpDir, "seed")
	repoRoot := filepath.Join(tmpDir, "repo")
	worktreeParent := filepath.Join(tmpDir, "worktrees")

	mustRun(t, tmpDir, "git", "init", "--bare", upstreamBare)
	mustRun(t, tmpDir, "git", "init", "--bare", originBare)
	mustRun(t, tmpDir, "git", "init", seedRepo)
	mustRun(t, seedRepo, "git", "config", "user.name", "Test User")
	mustRun(t, seedRepo, "git", "config", "user.email", "test@example.com")
	mustWriteFile(t, filepath.Join(seedRepo, "README.md"), []byte("seed\n"), 0o644)
	mustRun(t, seedRepo, "git", "add", "README.md")
	mustRun(t, seedRepo, "git", "commit", "-m", "seed commit")
	mustRun(t, seedRepo, "git", "branch", "-M", "main")
	mustRun(t, seedRepo, "git", "tag", "v0.2.4")
	mustRun(t, seedRepo, "git", "remote", "add", "upstream", upstreamBare)
	mustRun(t, seedRepo, "git", "remote", "add", "origin", originBare)
	mustRun(t, seedRepo, "git", "push", "upstream", "main", "--tags")
	mustRun(t, seedRepo, "git", "push", "origin", "main")

	mustRun(t, tmpDir, "git", "clone", upstreamBare, repoRoot)
	mustRun(t, repoRoot, "git", "remote", "rename", "origin", "upstream")
	mustRun(t, repoRoot, "git", "remote", "add", "origin", originBare)
	mustRun(t, repoRoot, "git", "config", "user.name", "Test User")
	mustRun(t, repoRoot, "git", "config", "user.email", "test@example.com")
	mustRun(t, repoRoot, "git", "fetch", "upstream", "--tags")
	mustRun(t, repoRoot, "git", "fetch", "origin")
	mustMkdirAll(t, filepath.Join(repoRoot, "scripts"), 0o755)

	scriptData, err := os.ReadFile("/home/yukun/dev/picobox-ai/github_repos/picoclaw-v0.2.4-h618/scripts/sync-upstream-release.sh")
	if err != nil {
		t.Fatalf("ReadFile(script): %v", err)
	}
	mustWriteFile(t, filepath.Join(repoRoot, "scripts", "sync-upstream-release.sh"), scriptData, 0o755)

	cmd := exec.Command("bash", "./scripts/sync-upstream-release.sh", "v0.2.4")
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "WORKTREE_PARENT="+worktreeParent)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("sync script failed: %v\n%s", err, out)
	}

	worktreeDir := filepath.Join(worktreeParent, "picoclaw-v0.2.4-h618")
	if _, err := os.Stat(worktreeDir); err != nil {
		t.Fatalf("worktree missing: %v", err)
	}

	branchRef := mustRun(t, repoRoot, "git", "rev-parse", "--verify", "custom/release-v0.2.4-h618")
	tagRef := mustRun(t, repoRoot, "git", "rev-list", "-n", "1", "v0.2.4")
	if branchRef != tagRef {
		t.Fatalf("branch ref = %q, want tag ref %q", branchRef, tagRef)
	}

	branchName := mustRun(t, worktreeDir, "git", "branch", "--show-current")
	if branchName != "custom/release-v0.2.4-h618" {
		t.Fatalf("worktree branch = %q, want %q", branchName, "custom/release-v0.2.4-h618")
	}

	if !strings.Contains(string(out), "git push -u origin custom/release-v0.2.4-h618") {
		t.Fatalf("script output missing push hint:\n%s", out)
	}
}

func mustRun(t *testing.T, dir string, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s failed: %v\n%s", name, strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

func mustWriteFile(t *testing.T, path string, data []byte, perm os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, data, perm); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}

func mustMkdirAll(t *testing.T, path string, perm os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(path, perm); err != nil {
		t.Fatalf("MkdirAll(%s): %v", path, err)
	}
}
