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

	announcementDir := filepath.Join(dir, "announcements", "daily")
	if err := os.MkdirAll(announcementDir, 0o755); err != nil {
		t.Fatalf("mkdir announcement dir: %v", err)
	}
	writeFile(t, filepath.Join(announcementDir, "2026-07-03.md"), strings.Join([]string{
		"# 2026-07-03 每日更新公告",
		"## 摘要",
		"本次更新优化了公告展示和战斗体验。",
		"## 玩家公告",
		"- 修复扫荡结算后偶发服务器异常提示的问题。",
		"- 公告详情支持更清晰的富文本排版。",
	}, "\n"))
	releaseFile := filepath.Join(dir, "RELEASE")
	writeFile(t, releaseFile, newSHA+" deployed from main-core\n")

	location := time.FixedZone("Asia/Shanghai", 8*60*60)
	plan, err := buildDailyUpdateAnnouncementPlan(dailyUpdateAnnouncementCursor{LastAnnouncedSha: oldSHA[:12]}, dailyUpdateAnnouncementOptions{
		SourceDir:            dir,
		ReleaseFile:          releaseFile,
		AnnouncementDir:      announcementDir,
		TargetDate:           time.Date(2026, 7, 3, 12, 0, 0, 0, location),
		Location:             location,
		MaxCommits:           10,
		MaxAnnouncementItems: 10,
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
	if len(plan.PlayerItems) != 2 || !strings.Contains(strings.Join(plan.PlayerItems, "\n"), "扫荡") {
		t.Fatalf("unexpected player items: %#v", plan.PlayerItems)
	}
	for _, forbidden := range []string{"代码变更", "新增每日公告自动化", "当前版本", "上线范围", "dbtool", "systemd"} {
		if strings.Contains(plan.Content, forbidden) {
			t.Fatalf("content should not expose technical detail %q: %s", forbidden, plan.Content)
		}
	}
	for _, want := range []string{"<article", "<ul>", "<li>", "修复扫荡结算后偶发服务器异常提示的问题", "不展示内部代码与文件修改记录"} {
		if !strings.Contains(plan.Content, want) {
			t.Fatalf("content missing %q: %s", want, plan.Content)
		}
	}
	if plan.Summary != "本次更新优化了公告展示和战斗体验。" {
		t.Fatalf("unexpected summary: %s", plan.Summary)
	}
}

// TestBuildDailyUpdateAnnouncementPlanRequiresPlayerSource 验证没有玩家公告源时不发布公告。
func TestBuildDailyUpdateAnnouncementPlanRequiresPlayerSource(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.name", "Hero3 Test")
	runGit(t, dir, "config", "user.email", "hero3@example.test")
	writeFile(t, filepath.Join(dir, "README.md"), "new\n")
	runGit(t, dir, "add", "README.md")
	runGit(t, dir, "commit", "-m", "fix: internal technical change")
	sha := gitOutput(t, dir, "rev-parse", "HEAD")
	releaseFile := filepath.Join(dir, "RELEASE")
	writeFile(t, releaseFile, sha+" deployed from main-core\n")

	plan, err := buildDailyUpdateAnnouncementPlan(dailyUpdateAnnouncementCursor{}, dailyUpdateAnnouncementOptions{
		SourceDir:            dir,
		ReleaseFile:          releaseFile,
		AnnouncementDir:      filepath.Join(dir, "announcements", "daily"),
		TargetDate:           time.Date(2026, 7, 3, 12, 0, 0, 0, time.FixedZone("Asia/Shanghai", 8*60*60)),
		MaxCommits:           10,
		MaxAnnouncementItems: 10,
	})
	if err != nil {
		t.Fatalf("buildDailyUpdateAnnouncementPlan failed: %v", err)
	}
	if plan.HasChanges {
		t.Fatalf("missing player announcement source should not publish: %#v", plan)
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
