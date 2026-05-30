package test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCheckpointBaselineSavePreservesWorkingTree(t *testing.T) {
	repoDir := initCheckpointRepo(t)
	writeCheckpointFile(t, repoDir, "tracked.txt", "tracked-v2\n")
	writeCheckpointFile(t, repoDir, "untracked.txt", "untracked-v1\n")

	beforeStatus := gitPorcelainStatus(t, repoDir)
	beforeTracked := readCheckpointFile(t, repoDir, "tracked.txt")

	runCheckpointScript(t, repoDir, "baseline", "before-merge")

	if got := gitPorcelainStatus(t, repoDir); got != beforeStatus {
		t.Fatalf("working tree status changed after baseline save:\nbefore: %q\nafter:  %q", beforeStatus, got)
	}
	if got := readCheckpointFile(t, repoDir, "tracked.txt"); got != beforeTracked {
		t.Fatalf("tracked file contents changed after baseline save: got %q, want %q", got, beforeTracked)
	}
	if got := gitStashList(t, repoDir); !strings.Contains(got, "cliproxy-checkpoint:baseline:before-merge") {
		t.Fatalf("stash list does not contain baseline snapshot:\n%s", got)
	}
}

func TestCheckpointRestoreAutosavesCurrentChanges(t *testing.T) {
	repoDir := initCheckpointRepo(t)
	writeCheckpointFile(t, repoDir, "tracked.txt", "snapshot-content\n")
	writeCheckpointFile(t, repoDir, "untracked.txt", "snapshot-untracked\n")
	runCheckpointScript(t, repoDir, "baseline", "restore-target")

	writeCheckpointFile(t, repoDir, "tracked.txt", "dirty-content\n")
	writeCheckpointFile(t, repoDir, "untracked.txt", "dirty-untracked\n")

	runCheckpointScript(t, repoDir, "restore-baseline", "restore-target")

	if got := readCheckpointFile(t, repoDir, "tracked.txt"); got != "snapshot-content\n" {
		t.Fatalf("tracked file not restored from checkpoint: got %q", got)
	}
	if got := readCheckpointFile(t, repoDir, "untracked.txt"); got != "snapshot-untracked\n" {
		t.Fatalf("untracked file not restored from checkpoint: got %q", got)
	}
	if got := gitStashList(t, repoDir); !strings.Contains(got, "cliproxy-checkpoint:autosave:before-restore-") {
		t.Fatalf("restore did not autosave the dirty tree:\n%s", got)
	}
}

func TestCheckpointListShowsSavedSnapshots(t *testing.T) {
	repoDir := initCheckpointRepo(t)
	writeCheckpointFile(t, repoDir, "tracked.txt", "list-check\n")
	runCheckpointScript(t, repoDir, "save", "list-target")

	output := runCheckpointScript(t, repoDir, "list")
	if !strings.Contains(output, "cliproxy-checkpoint:wip:list-target") {
		t.Fatalf("list output does not include saved checkpoint:\n%s", output)
	}
	if strings.Contains(output, "No checkpoints found.") {
		t.Fatalf("list output reported no checkpoints despite an existing snapshot:\n%s", output)
	}
}

func initCheckpointRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	runGitCommand(t, dir, "init")
	runGitCommand(t, dir, "config", "user.name", "Checkpoint Test")
	runGitCommand(t, dir, "config", "user.email", "checkpoint@example.com")

	writeCheckpointFile(t, dir, "tracked.txt", "tracked-v1\n")
	runGitCommand(t, dir, "add", "tracked.txt")
	runGitCommand(t, dir, "commit", "-m", "initial commit")

	return dir
}

func runCheckpointScript(t *testing.T, repoDir string, args ...string) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	scriptPath := filepath.Join(filepath.Dir(filepath.Dir(thisFile)), "checkpoint.sh")
	cmdArgs := append([]string{scriptPath}, args...)
	return runCommand(t, repoDir, "bash", cmdArgs...)
}

func runGitCommand(t *testing.T, repoDir string, args ...string) string {
	t.Helper()
	return runCommand(t, repoDir, "git", args...)
}

func runCommand(t *testing.T, repoDir, name string, args ...string) string {
	t.Helper()

	cmd := exec.Command(name, args...)
	cmd.Dir = repoDir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		t.Fatalf("%s %v failed: %v\noutput:\n%s", name, args, err, buf.String())
	}
	return buf.String()
}

func gitPorcelainStatus(t *testing.T, repoDir string) string {
	t.Helper()
	return strings.TrimSpace(runGitCommand(t, repoDir, "status", "--porcelain", "--untracked-files=all"))
}

func gitStashList(t *testing.T, repoDir string) string {
	t.Helper()
	return runGitCommand(t, repoDir, "stash", "list", "--format=%gs")
}

func writeCheckpointFile(t *testing.T, repoDir, name, content string) {
	t.Helper()
	path := filepath.Join(repoDir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func readCheckpointFile(t *testing.T, repoDir, name string) string {
	t.Helper()
	path := filepath.Join(repoDir, name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}
