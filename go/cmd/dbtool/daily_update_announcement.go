// 本文件提供每日更新公告自动生成与发布命令。
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"hero3/internal/app/game"
	"hero3/internal/infrastructure/storage"
)

const dailyUpdateAnnouncementCursorKey = "daily_update_announcement_cursor"

type dailyUpdateAnnouncementCursor struct {
	LastAnnouncedSha   string `json:"lastAnnouncedSha"`
	LastAnnouncementID string `json:"lastAnnouncementId,omitempty"`
	LastPublishedAt    string `json:"lastPublishedAt"`
}

type dailyUpdateAnnouncementOptions struct {
	SourceDir            string
	ReleaseFile          string
	AnnouncementDir      string
	CurrentSha           string
	TargetDate           time.Time
	Location             *time.Location
	MaxCommits           int
	MaxAnnouncementItems int
	TitlePrefix          string
	DisplayMode          string
	ForcePopup           bool
	Pinned               bool
	Priority             int
	EndsAfterHours       int
	Execute              bool
}

type dailyUpdateCommit struct {
	SHA     string
	Subject string
	Date    string
}

type dailyUpdateAnnouncementPlan struct {
	CurrentSha  string
	PreviousSha string
	Title       string
	Summary     string
	Content     string
	Commits     []dailyUpdateCommit
	PlayerItems []string
	HasChanges  bool
}

type dailyPlayerAnnouncementSource struct {
	Summary string
	Items   []string
}

// runPublishDailyUpdateAnnouncement 生成或发布每日更新公告。
func runPublishDailyUpdateAnnouncement(args []string) error {
	flags := flag.NewFlagSet("publish-daily-update-announcement", flag.ContinueOnError)
	dsn := flags.String("dsn", "", "目标数据库 DSN，默认使用 HERO3_DATABASE_DSN")
	sourceDir := flags.String("source-dir", "/opt/hero3/source", "服务器源码目录")
	releaseFile := flags.String("release-file", "/opt/hero3/RELEASE", "当前线上发布记录文件")
	announcementDir := flags.String("announcement-dir", "", "玩家公告源目录，默认使用 source-dir/announcements/daily")
	currentSha := flags.String("current-sha", "", "当前线上提交，默认从 release-file 读取")
	dateText := flags.String("date", "", "公告日期，格式 YYYY-MM-DD，默认北京时间当天")
	maxCommits := flags.Int("max-commits", 20, "用于判断更新范围的最多提交数")
	maxAnnouncementItems := flags.Int("max-announcement-items", 10, "公告中最多展示多少条玩家更新项")
	titlePrefix := flags.String("title-prefix", "更新公告", "公告标题前缀")
	displayMode := flags.String("display-mode", game.AnnouncementDisplayCenterOnly, "公告展示模式：center_only、popup 或 banner")
	forcePopup := flags.Bool("force-popup", false, "是否强制弹窗")
	pinned := flags.Bool("pinned", false, "是否置顶公告")
	priority := flags.Int("priority", 20, "公告优先级")
	endsAfterHours := flags.Int("ends-after-hours", 168, "公告有效期小时数，0 表示不过期")
	execute := flags.Bool("execute", false, "真正发布公告并写入 cursor；默认只预览")
	allowNonTest := flags.Bool("allow-non-test", false, "允许写入非 test_ 前缀数据库")
	if err := flags.Parse(args); err != nil {
		return err
	}

	location := loadDailyAnnouncementLocation()
	targetDate, err := parseDailyAnnouncementDate(*dateText, location)
	if err != nil {
		return err
	}
	resolvedDSN := strings.TrimSpace(*dsn)
	if resolvedDSN == "" {
		resolvedDSN, err = configuredDSN()
		if err != nil {
			return err
		}
	}
	databaseName, err := storage.MySQLDatabaseName(resolvedDSN)
	if err != nil {
		return err
	}
	if *execute && !*allowNonTest && !strings.HasPrefix(databaseName, "test_") {
		return fmt.Errorf("target database must use test_ prefix or pass --allow-non-test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), longCommandTimeout)
	defer cancel()
	db, err := storage.OpenMySQL(ctx, resolvedDSN)
	if err != nil {
		return err
	}
	defer db.Close()
	repo := storage.NewMySQLRepository(db)

	cursor, err := loadDailyUpdateCursor(repo)
	if err != nil {
		return err
	}
	options := dailyUpdateAnnouncementOptions{
		SourceDir:            *sourceDir,
		ReleaseFile:          *releaseFile,
		AnnouncementDir:      *announcementDir,
		CurrentSha:           *currentSha,
		TargetDate:           targetDate,
		Location:             location,
		MaxCommits:           *maxCommits,
		MaxAnnouncementItems: *maxAnnouncementItems,
		TitlePrefix:          *titlePrefix,
		DisplayMode:          *displayMode,
		ForcePopup:           *forcePopup,
		Pinned:               *pinned,
		Priority:             *priority,
		EndsAfterHours:       *endsAfterHours,
		Execute:              *execute,
	}
	plan, err := buildDailyUpdateAnnouncementPlan(cursor, options)
	if err != nil {
		return err
	}
	printDailyUpdateAnnouncementPlan(databaseName, plan, *execute)
	if !plan.HasChanges {
		return nil
	}
	if !*execute {
		fmt.Println("dry-run：未发布公告；加 --execute 后才会写入线上公告和 cursor。")
		return nil
	}

	now := time.Now().UTC()
	announcement, err := game.NewServiceWithRepository(repo).CreateAnnouncement(game.SaveAnnouncementRequest{
		Title:       plan.Title,
		Summary:     plan.Summary,
		Content:     plan.Content,
		Type:        game.AnnouncementTypeUpdate,
		Status:      game.AnnouncementStatusPublished,
		DisplayMode: normalizeDailyAnnouncementDisplayMode(options.DisplayMode),
		Pinned:      options.Pinned,
		Priority:    options.Priority,
		ForcePopup:  options.ForcePopup,
		StartsAt:    now.Format(time.RFC3339),
		EndsAt:      dailyAnnouncementEndsAt(now, options.EndsAfterHours),
		Targets:     []game.AnnouncementTarget{{Type: game.AnnouncementTargetAll}},
	})
	if err != nil {
		return err
	}
	cursor = dailyUpdateAnnouncementCursor{
		LastAnnouncedSha:   plan.CurrentSha,
		LastAnnouncementID: announcement.ID,
		LastPublishedAt:    now.Format(time.RFC3339),
	}
	if err := saveDailyUpdateCursor(repo, cursor); err != nil {
		return err
	}
	fmt.Printf("已发布每日更新公告：%s，cursor=%s\n", announcement.ID, cursor.LastAnnouncedSha)
	return nil
}

// loadDailyUpdateCursor 从 game_configs 读取每日公告发布游标。
func loadDailyUpdateCursor(repo game.GameConfigRepository) (dailyUpdateAnnouncementCursor, error) {
	record, exists, err := repo.GetGameConfig(dailyUpdateAnnouncementCursorKey)
	if err != nil {
		return dailyUpdateAnnouncementCursor{}, err
	}
	if !exists || len(record.ValueJSON) == 0 {
		return dailyUpdateAnnouncementCursor{}, nil
	}
	var cursor dailyUpdateAnnouncementCursor
	if err := json.Unmarshal(record.ValueJSON, &cursor); err != nil {
		return dailyUpdateAnnouncementCursor{}, err
	}
	cursor.LastAnnouncedSha = strings.TrimSpace(cursor.LastAnnouncedSha)
	return cursor, nil
}

// saveDailyUpdateCursor 将每日公告发布游标写入 game_configs。
func saveDailyUpdateCursor(repo game.GameConfigRepository, cursor dailyUpdateAnnouncementCursor) error {
	content, err := json.MarshalIndent(cursor, "", "  ")
	if err != nil {
		return err
	}
	_, err = repo.SaveGameConfig(dailyUpdateAnnouncementCursorKey, content, "daily_update_announcement", time.Now().UTC())
	return err
}

// buildDailyUpdateAnnouncementPlan 根据线上提交范围和玩家公告源生成公告计划。
func buildDailyUpdateAnnouncementPlan(cursor dailyUpdateAnnouncementCursor, options dailyUpdateAnnouncementOptions) (dailyUpdateAnnouncementPlan, error) {
	options = normalizeDailyUpdateAnnouncementOptions(options)
	currentSha, err := resolveDailyAnnouncementCurrentSha(options)
	if err != nil {
		return dailyUpdateAnnouncementPlan{}, err
	}
	if currentSha == "" {
		return dailyUpdateAnnouncementPlan{}, fmt.Errorf("current release sha is empty")
	}
	previousSha := strings.TrimSpace(cursor.LastAnnouncedSha)
	if previousSha != "" && sameDailyAnnouncementSHA(previousSha, currentSha) {
		return dailyUpdateAnnouncementPlan{CurrentSha: currentSha, PreviousSha: previousSha}, nil
	}

	commits, err := collectDailyAnnouncementCommits(options.SourceDir, previousSha, currentSha, options.TargetDate, options.Location, options.MaxCommits)
	if err != nil {
		return dailyUpdateAnnouncementPlan{}, err
	}
	source, ok := collectDailyPlayerAnnouncementSource(options.AnnouncementDir, options.TargetDate, options.MaxAnnouncementItems)
	if len(commits) == 0 || !ok || len(source.Items) == 0 {
		return dailyUpdateAnnouncementPlan{CurrentSha: currentSha, PreviousSha: previousSha}, nil
	}
	titleDate := options.TargetDate.In(options.Location).Format("1月2日")
	title := fmt.Sprintf("%s%s", titleDate, strings.TrimSpace(options.TitlePrefix))
	summary := strings.TrimSpace(source.Summary)
	if summary == "" {
		summary = fmt.Sprintf("本次更新包含 %d 项玩法、体验与问题修复。", len(source.Items))
	}
	content := renderDailyUpdateAnnouncementContent(titleDate, source.Items)
	return dailyUpdateAnnouncementPlan{
		CurrentSha:  currentSha,
		PreviousSha: previousSha,
		Title:       title,
		Summary:     summary,
		Content:     content,
		Commits:     commits,
		PlayerItems: source.Items,
		HasChanges:  true,
	}, nil
}

// normalizeDailyUpdateAnnouncementOptions 补齐每日公告选项默认值。
func normalizeDailyUpdateAnnouncementOptions(options dailyUpdateAnnouncementOptions) dailyUpdateAnnouncementOptions {
	if options.Location == nil {
		options.Location = loadDailyAnnouncementLocation()
	}
	if options.TargetDate.IsZero() {
		options.TargetDate = time.Now().In(options.Location)
	}
	if strings.TrimSpace(options.AnnouncementDir) == "" && strings.TrimSpace(options.SourceDir) != "" {
		options.AnnouncementDir = filepath.Join(options.SourceDir, "announcements", "daily")
	}
	if options.MaxCommits <= 0 {
		options.MaxCommits = 20
	}
	if options.MaxAnnouncementItems < 0 {
		options.MaxAnnouncementItems = 0
	}
	if strings.TrimSpace(options.TitlePrefix) == "" {
		options.TitlePrefix = "更新公告"
	}
	if options.EndsAfterHours < 0 {
		options.EndsAfterHours = 0
	}
	return options
}

// resolveDailyAnnouncementCurrentSha 读取当前线上提交。
func resolveDailyAnnouncementCurrentSha(options dailyUpdateAnnouncementOptions) (string, error) {
	if strings.TrimSpace(options.CurrentSha) != "" {
		return strings.TrimSpace(options.CurrentSha), nil
	}
	release, err := os.ReadFile(options.ReleaseFile)
	if err != nil {
		return "", err
	}
	return parseDailyReleaseSHA(string(release)), nil
}

// parseDailyReleaseSHA 从 /opt/hero3/RELEASE 内容中解析提交号。
func parseDailyReleaseSHA(content string) string {
	fields := strings.Fields(strings.TrimSpace(content))
	if len(fields) == 0 {
		return ""
	}
	return strings.TrimSpace(fields[0])
}

// collectDailyAnnouncementCommits 收集上次公告到当前发布之间的提交。
func collectDailyAnnouncementCommits(sourceDir string, previousSha string, currentSha string, targetDate time.Time, location *time.Location, maxCommits int) ([]dailyUpdateCommit, error) {
	sourceDir = strings.TrimSpace(sourceDir)
	if sourceDir == "" {
		return nil, errors.New("source-dir is empty")
	}
	args := []string{"-C", sourceDir, "log", "--reverse", "--date=short", "--pretty=format:%h%x1f%s%x1f%ad%x1e"}
	if strings.TrimSpace(previousSha) != "" {
		args = append(args, fmt.Sprintf("%s..%s", strings.TrimSpace(previousSha), strings.TrimSpace(currentSha)))
	} else {
		start := startOfDailyAnnouncementDate(targetDate, location)
		args = append(args, "--since", start.Format(time.RFC3339), strings.TrimSpace(currentSha))
	}
	output, err := exec.Command("git", args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("collect git log: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return parseDailyAnnouncementCommits(string(output), maxCommits), nil
}

// parseDailyAnnouncementCommits 解析 git log 的分隔格式输出。
func parseDailyAnnouncementCommits(output string, maxCommits int) []dailyUpdateCommit {
	var commits []dailyUpdateCommit
	for _, record := range strings.Split(output, "\x1e") {
		record = strings.TrimSpace(record)
		if record == "" {
			continue
		}
		parts := strings.Split(record, "\x1f")
		if len(parts) < 2 {
			continue
		}
		commit := dailyUpdateCommit{SHA: strings.TrimSpace(parts[0]), Subject: strings.TrimSpace(parts[1])}
		if len(parts) >= 3 {
			commit.Date = strings.TrimSpace(parts[2])
		}
		if commit.Subject == "" {
			continue
		}
		commits = append(commits, commit)
		if maxCommits > 0 && len(commits) >= maxCommits {
			break
		}
	}
	return commits
}

// collectDailyPlayerAnnouncementSource 读取当天玩家公告源文件。
func collectDailyPlayerAnnouncementSource(announcementDir string, targetDate time.Time, maxItems int) (dailyPlayerAnnouncementSource, bool) {
	path := filepath.Join(strings.TrimSpace(announcementDir), targetDate.Format("2006-01-02")+".md")
	content, err := os.ReadFile(path)
	if err != nil {
		return dailyPlayerAnnouncementSource{}, false
	}
	source := parseDailyPlayerAnnouncementSource(string(content), maxItems)
	return source, len(source.Items) > 0
}

// parseDailyPlayerAnnouncementSource 解析 announcements/daily/YYYY-MM-DD.md。
func parseDailyPlayerAnnouncementSource(content string, maxItems int) dailyPlayerAnnouncementSource {
	if maxItems <= 0 {
		maxItems = 10
	}
	var source dailyPlayerAnnouncementSource
	section := ""
	for _, raw := range strings.Split(content, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "## ") {
			title := strings.TrimSpace(strings.TrimPrefix(line, "## "))
			switch title {
			case "摘要", "公告摘要":
				section = "summary"
			case "玩家公告", "更新内容", "本次更新":
				section = "items"
			default:
				section = ""
			}
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		switch section {
		case "summary":
			if source.Summary == "" {
				source.Summary = strings.TrimSpace(strings.TrimPrefix(line, "- "))
			}
		case "items":
			if len(source.Items) >= maxItems || !strings.HasPrefix(line, "- ") {
				continue
			}
			item := strings.TrimSpace(strings.TrimPrefix(line, "- "))
			if item != "" {
				source.Items = append(source.Items, item)
			}
		}
	}
	return source
}

// renderDailyUpdateAnnouncementContent 生成带有限 HTML 排版的公告正文。
func renderDailyUpdateAnnouncementContent(titleDate string, playerItems []string) string {
	var builder strings.Builder
	builder.WriteString(`<article class="hero3-announcement hero3-update-announcement">`)
	fmt.Fprintf(&builder, `<p class="hero3-announcement-lead">%s更新内容已上线。</p>`, html.EscapeString(titleDate))
	if len(playerItems) > 0 {
		builder.WriteString(`<section class="hero3-announcement-section"><h3>本次更新</h3><ul>`)
		for _, item := range playerItems {
			fmt.Fprintf(&builder, "<li>%s</li>", html.EscapeString(item))
		}
		builder.WriteString(`</ul></section>`)
	}
	builder.WriteString(`<p class="hero3-announcement-note">本公告只汇总已经上线的游戏内容和体验调整，不展示内部代码与文件修改记录。</p>`)
	builder.WriteString(`</article>`)
	return strings.TrimSpace(builder.String())
}

// printDailyUpdateAnnouncementPlan 输出每日公告发布计划。
func printDailyUpdateAnnouncementPlan(databaseName string, plan dailyUpdateAnnouncementPlan, execute bool) {
	mode := "dry-run"
	if execute {
		mode = "execute"
	}
	fmt.Printf("每日更新公告计划：database=%s mode=%s current=%s previous=%s commits=%d playerItems=%d\n",
		databaseName, mode, shortDailyAnnouncementSHA(plan.CurrentSha), shortDailyAnnouncementSHA(plan.PreviousSha), len(plan.Commits), len(plan.PlayerItems))
	if !plan.HasChanges {
		fmt.Println("没有发现需要发布的线上更新公告。")
		return
	}
	fmt.Printf("\n标题：%s\n摘要：%s\n\n%s\n\n", plan.Title, plan.Summary, plan.Content)
}

// dailyAnnouncementEndsAt 根据保留小时数生成公告结束时间。
func dailyAnnouncementEndsAt(now time.Time, hours int) string {
	if hours <= 0 {
		return ""
	}
	return now.Add(time.Duration(hours) * time.Hour).Format(time.RFC3339)
}

// normalizeDailyAnnouncementDisplayMode 规范每日公告展示模式。
func normalizeDailyAnnouncementDisplayMode(value string) string {
	switch strings.TrimSpace(value) {
	case game.AnnouncementDisplayPopup, game.AnnouncementDisplayBanner:
		return strings.TrimSpace(value)
	default:
		return game.AnnouncementDisplayCenterOnly
	}
}

// loadDailyAnnouncementLocation 读取每日公告使用的北京时间时区。
func loadDailyAnnouncementLocation() *time.Location {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	return location
}

// parseDailyAnnouncementDate 解析命令行日期。
func parseDailyAnnouncementDate(value string, location *time.Location) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Now().In(location), nil
	}
	parsed, err := time.ParseInLocation("2006-01-02", value, location)
	if err != nil {
		return time.Time{}, err
	}
	return parsed, nil
}

// startOfDailyAnnouncementDate 返回公告日期的北京时间零点。
func startOfDailyAnnouncementDate(date time.Time, location *time.Location) time.Time {
	local := date.In(location)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
}

// sameDailyAnnouncementSHA 判断短提交和完整提交是否指向同一前缀。
func sameDailyAnnouncementSHA(left string, right string) bool {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	if left == "" || right == "" {
		return false
	}
	return strings.HasPrefix(left, right) || strings.HasPrefix(right, left)
}

// shortDailyAnnouncementSHA 返回公告中展示用短提交。
func shortDailyAnnouncementSHA(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= 7 {
		return value
	}
	return value[:7]
}
