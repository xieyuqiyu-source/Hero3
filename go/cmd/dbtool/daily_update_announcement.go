// 本文件提供每日更新公告自动生成与发布命令。
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
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
	SourceDir      string
	ReleaseFile    string
	MemoryDir      string
	CurrentSha     string
	TargetDate     time.Time
	Location       *time.Location
	MaxCommits     int
	MaxMemoryItems int
	TitlePrefix    string
	DisplayMode    string
	ForcePopup     bool
	Pinned         bool
	Priority       int
	EndsAfterHours int
	Execute        bool
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
	MemoryItems []string
	HasChanges  bool
}

// runPublishDailyUpdateAnnouncement 生成或发布每日更新公告。
func runPublishDailyUpdateAnnouncement(args []string) error {
	flags := flag.NewFlagSet("publish-daily-update-announcement", flag.ContinueOnError)
	dsn := flags.String("dsn", "", "目标数据库 DSN，默认使用 HERO3_DATABASE_DSN")
	sourceDir := flags.String("source-dir", "/opt/hero3/source", "服务器源码目录")
	releaseFile := flags.String("release-file", "/opt/hero3/RELEASE", "当前线上发布记录文件")
	memoryDir := flags.String("memory-dir", "", "记忆目录，默认使用 source-dir/memory")
	currentSha := flags.String("current-sha", "", "当前线上提交，默认从 release-file 读取")
	dateText := flags.String("date", "", "公告日期，格式 YYYY-MM-DD，默认北京时间当天")
	maxCommits := flags.Int("max-commits", 20, "公告中最多展示多少条提交")
	maxMemoryItems := flags.Int("max-memory-items", 10, "公告中最多展示多少条记忆摘要")
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
		SourceDir:      *sourceDir,
		ReleaseFile:    *releaseFile,
		MemoryDir:      *memoryDir,
		CurrentSha:     *currentSha,
		TargetDate:     targetDate,
		Location:       location,
		MaxCommits:     *maxCommits,
		MaxMemoryItems: *maxMemoryItems,
		TitlePrefix:    *titlePrefix,
		DisplayMode:    *displayMode,
		ForcePopup:     *forcePopup,
		Pinned:         *pinned,
		Priority:       *priority,
		EndsAfterHours: *endsAfterHours,
		Execute:        *execute,
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

// buildDailyUpdateAnnouncementPlan 汇总 git 提交和记忆文件，生成公告计划。
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
	memoryItems := collectDailyAnnouncementMemoryItems(options.MemoryDir, options.TargetDate, options.MaxMemoryItems)
	if len(commits) == 0 && len(memoryItems) == 0 {
		return dailyUpdateAnnouncementPlan{CurrentSha: currentSha, PreviousSha: previousSha}, nil
	}
	titleDate := options.TargetDate.In(options.Location).Format("1月2日")
	title := fmt.Sprintf("%s%s", titleDate, strings.TrimSpace(options.TitlePrefix))
	summary := fmt.Sprintf("今日已上线 %d 条代码更新，整理了 %d 条关键变更。", len(commits), len(memoryItems))
	content := renderDailyUpdateAnnouncementContent(titleDate, currentSha, previousSha, commits, memoryItems)
	return dailyUpdateAnnouncementPlan{
		CurrentSha:  currentSha,
		PreviousSha: previousSha,
		Title:       title,
		Summary:     summary,
		Content:     content,
		Commits:     commits,
		MemoryItems: memoryItems,
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
	if strings.TrimSpace(options.MemoryDir) == "" && strings.TrimSpace(options.SourceDir) != "" {
		options.MemoryDir = filepath.Join(options.SourceDir, "memory")
	}
	if options.MaxCommits <= 0 {
		options.MaxCommits = 20
	}
	if options.MaxMemoryItems < 0 {
		options.MaxMemoryItems = 0
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

// collectDailyAnnouncementMemoryItems 从当天记忆文件提取公告可读摘要。
func collectDailyAnnouncementMemoryItems(memoryDir string, targetDate time.Time, maxItems int) []string {
	if maxItems <= 0 {
		return nil
	}
	path := filepath.Join(strings.TrimSpace(memoryDir), targetDate.Format("2006-01-02")+".md")
	content, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return extractDailyAnnouncementMemoryItems(string(content), maxItems)
}

// extractDailyAnnouncementMemoryItems 提取适合玩家公告展示的记忆条目。
func extractDailyAnnouncementMemoryItems(content string, maxItems int) []string {
	var items []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "- ") {
			continue
		}
		item := strings.TrimSpace(strings.TrimPrefix(line, "- "))
		if item == "" || dailyAnnouncementMemoryItemSkipped(item) {
			continue
		}
		items = append(items, item)
		if maxItems > 0 && len(items) >= maxItems {
			break
		}
	}
	return items
}

// dailyAnnouncementMemoryItemSkipped 判断记忆条目是否不适合进入玩家公告。
func dailyAnnouncementMemoryItemSkipped(item string) bool {
	skipWords := []string{"用户要求", "用户指出", "已验证", "验证通过", "git diff", "go test", "npm run", "pnpm build", "OpenAPI", "README"}
	for _, word := range skipWords {
		if strings.Contains(item, word) {
			return true
		}
	}
	return false
}

// renderDailyUpdateAnnouncementContent 生成最终公告正文。
func renderDailyUpdateAnnouncementContent(titleDate string, currentSha string, previousSha string, commits []dailyUpdateCommit, memoryItems []string) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "%s更新内容已上线。\n\n", titleDate)
	if len(memoryItems) > 0 {
		builder.WriteString("主要更新：\n")
		for idx, item := range memoryItems {
			fmt.Fprintf(&builder, "%d. %s\n", idx+1, item)
		}
		builder.WriteString("\n")
	}
	if len(commits) > 0 {
		builder.WriteString("代码变更：\n")
		for idx, commit := range commits {
			fmt.Fprintf(&builder, "%d. %s\n", idx+1, commit.Subject)
		}
		builder.WriteString("\n")
	}
	if strings.TrimSpace(previousSha) != "" {
		fmt.Fprintf(&builder, "上线范围：%s -> %s\n", shortDailyAnnouncementSHA(previousSha), shortDailyAnnouncementSHA(currentSha))
	} else {
		fmt.Fprintf(&builder, "当前版本：%s\n", shortDailyAnnouncementSHA(currentSha))
	}
	builder.WriteString("本公告只汇总已经发布到线上环境的内容。")
	return strings.TrimSpace(builder.String())
}

// printDailyUpdateAnnouncementPlan 输出每日公告发布计划。
func printDailyUpdateAnnouncementPlan(databaseName string, plan dailyUpdateAnnouncementPlan, execute bool) {
	mode := "dry-run"
	if execute {
		mode = "execute"
	}
	fmt.Printf("每日更新公告计划：database=%s mode=%s current=%s previous=%s commits=%d memory=%d\n",
		databaseName, mode, shortDailyAnnouncementSHA(plan.CurrentSha), shortDailyAnnouncementSHA(plan.PreviousSha), len(plan.Commits), len(plan.MemoryItems))
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
