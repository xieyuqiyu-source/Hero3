// 本文件测试每日更新公告生成逻辑，避免重复公告和提交范围错误。
package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestParseDailyReleaseSHA 验证发布记录文件的提交号解析。
func TestParseDailyReleaseSHA(t *testing.T) {
	got := parseDailyReleaseSHA("abc1234 deployed from main-core\n")
	if got != "abc1234" {
		t.Fatalf("parseDailyReleaseSHA = %q", got)
	}
}

// TestBuildDailyUpdateAnnouncementPlanUsesCursorRange 验证公告只汇总 cursor 之后的提交。
func TestBuildDailyUpdateAnnouncementPlanUsesCursorRange(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.name", "Hero3 Test")
	runGit(t, dir, "config", "user.email", "hero3@example.test")

	writeFile(t, filepath.Join(dir, "README.md"), "old\n")
	runGit(t, dir, "add", "README.md")
	runGit(t, dir, "commit", "-m", "旧提交")
	oldSHA := gitOutput(t, dir, "rev-parse", "HEAD")

	writeFile(t, filepath.Join(dir, "feature.txt"), "new\n")
	runGit(t, dir, "add", "feature.txt")
	runGit(t, dir, "commit", "-m", "新增每日公告自动化")
	newSHA := gitOutput(t, dir, "rev-parse", "HEAD")

	memoryDir := filepath.Join(dir, "memory")
	if err := os.MkdirAll(memoryDir, 0o755); err != nil {
		t.Fatalf("mkdir memory: %v", err)
	}
	writeFile(t, filepath.Join(memoryDir, "2026-07-03.md"), strings.Join([]string{
		"# 2026-07-03 记忆",
		"- 每日更新公告接入 dbtool 和 systemd timer。",
		"- 已验证：go test ./cmd/dbtool。",
	}, "\n"))
	releaseFile := filepath.Join(dir, "RELEASE")
	writeFile(t, releaseFile, newSHA+" deployed from main-core\n")

	location := time.FixedZone("Asia/Shanghai", 8*60*60)
	plan, err := buildDailyUpdateAnnouncementPlan(dailyUpdateAnnouncementCursor{LastAnnouncedSha: oldSHA[:12]}, dailyUpdateAnnouncementOptions{
		SourceDir:      dir,
		ReleaseFile:    releaseFile,
		MemoryDir:      memoryDir,
		TargetDate:     time.Date(2026, 7, 3, 12, 0, 0, 0, location),
		Location:       location,
		MaxCommits:     10,
		MaxMemoryItems: 10,
	})
	if err != nil {
		t.Fatalf("buildDailyUpdateAnnouncementPlan failed: %v", err)
	}
	if !plan.HasChanges {
		t.Fatal("plan should have changes")
	}
	if len(plan.Commits) != 1 || plan.Commits[0].Subject != "新增每日公告自动化" {
		t.Fatalf("unexpected commits: %#v", plan.Commits)
	}
	if len(plan.MemoryItems) != 1 || !strings.Contains(plan.MemoryItems[0], "systemd timer") {
		t.Fatalf("unexpected memory items: %#v", plan.MemoryItems)
	}
	if !strings.Contains(plan.Content, "本公告只汇总已经发布到线上环境的内容") {
		t.Fatalf("content missing safety note: %s", plan.Content)
	}
}

// TestBuildDailyUpdateAnnouncementPlanSkipsSameSHA 验证同一提交不会重复发布。
func TestBuildDailyUpdateAnnouncementPlanSkipsSameSHA(t *testing.T) {
	dir := t.TempDir()
	releaseFile := filepath.Join(dir, "RELEASE")
	writeFile(t, releaseFile, "abcdef123456 deployed from main-core\n")
	plan, err := buildDailyUpdateAnnouncementPlan(dailyUpdateAnnouncementCursor{LastAnnouncedSha: "abcdef1"}, dailyUpdateAnnouncementOptions{
		SourceDir:   dir,
		ReleaseFile: releaseFile,
	})
	if err != nil {
		t.Fatalf("buildDailyUpdateAnnouncementPlan failed: %v", err)
	}
	if plan.HasChanges {
		t.Fatalf("same sha should not publish: %#v", plan)
	}
}

// runGit 执行测试 Git 命令。
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_DATE=2026-07-03T10:00:00+08:00",
		"GIT_COMMITTER_DATE=2026-07-03T10:00:00+08:00",
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(output))
	}
}

// gitOutput 执行 Git 命令并返回去空白输出。
func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(output))
	}
	return strings.TrimSpace(string(output))
}

// writeFile 写入测试文件。
func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
