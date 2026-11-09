package report

import (
	_ "embed"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"daily-report/internal/browser"
	"daily-report/internal/config"
	"daily-report/internal/storage"
)

//go:embed prompt_daily.txt
var defaultDailyPrompt string

//go:embed prompt_weekly.txt
var defaultWeeklyPrompt string

type Generator struct {
	db  *storage.DB
	cfg *config.Config
}

func NewGenerator(db *storage.DB, cfg *config.Config) *Generator {
	return &Generator{db: db, cfg: cfg}
}

// GenerateDailyReport 生成日报
func (g *Generator) GenerateDailyReport(date time.Time) (string, error) {
	start, end := g.getDayRange(date)

	// 同步浏览器历史
	if g.cfg.Browser.Enabled {
		browser.SyncAll(g.db, g.cfg.Browser.Browsers)
	}

	// 查询数据
	activities, err := g.db.GetWindowActivities(start, end)
	if err != nil {
		return "", fmt.Errorf("查询窗口活动失败: %w", err)
	}

	summaries, err := g.db.GetAppUsageSummary(start, end)
	if err != nil {
		return "", fmt.Errorf("查询应用汇总失败: %w", err)
	}

	visits, err := g.db.GetBrowserVisits(start, end)
	if err != nil {
		log.Printf("查询浏览器记录失败: %v", err)
		// 不中断，浏览器记录是可选的
	}

	if len(activities) == 0 {
		return "", fmt.Errorf("没有找到 %s 的活动记录", date.Format("2006-01-02"))
	}

	// 格式化数据
	userData := g.formatDailyData(date, summaries, visits)

	// 调用 LLM
	systemPrompt := defaultDailyPrompt
	content, err := CallLLM(g.cfg, systemPrompt, userData)
	if err != nil {
		return "", fmt.Errorf("调用 LLM 失败: %w", err)
	}

	// 生成 .docx
	outputDir := g.cfg.GetReportOutputDir()
	fileName := fmt.Sprintf("日报_%s.docx", date.Format("2006-01-02"))
	filePath := filepath.Join(outputDir, fileName)

	if err := SaveDocx(content, filePath); err != nil {
		return "", fmt.Errorf("保存文档失败: %w", err)
	}

	// 记录
	dateStr := date.Format("2006-01-02")
	g.db.InsertReport("daily", dateStr, dateStr, filePath)

	log.Printf("日报已生成: %s", filePath)
	return filePath, nil
}

// GenerateWeeklyReport 生成周报
func (g *Generator) GenerateWeeklyReport(weekEnd time.Time) (string, error) {
	// 计算本周一到周五
	monday := WeekMonday(weekEnd)
	friday := monday.AddDate(0, 0, 4)

	// 同步浏览器历史
	if g.cfg.Browser.Enabled {
		browser.SyncAll(g.db, g.cfg.Browser.Browsers)
	}

	// 收集每天的数据
	var dailySummaries []string
	for d := monday; !d.After(friday); d = d.AddDate(0, 0, 1) {
		start, end := g.getDayRange(d)
		summaries, err := g.db.GetAppUsageSummary(start, end)
		if err != nil {
			continue
		}
		visits, _ := g.db.GetBrowserVisits(start, end)
		if len(summaries) > 0 {
			dailySummaries = append(dailySummaries, g.formatDaySummary(d, summaries, visits))
		}
	}

	if len(dailySummaries) == 0 {
		return "", fmt.Errorf("本周没有活动记录")
	}

	// 格式化数据
	userData := fmt.Sprintf("## 本周活动数据（%s ~ %s）\n\n%s",
		monday.Format("2006-01-02"),
		friday.Format("2006-01-02"),
		strings.Join(dailySummaries, "\n\n"),
	)

	// 调用 LLM
	systemPrompt := defaultWeeklyPrompt
	content, err := CallLLM(g.cfg, systemPrompt, userData)
	if err != nil {
		return "", fmt.Errorf("调用 LLM 失败: %w", err)
	}

	// 生成 .docx
	outputDir := g.cfg.GetReportOutputDir()
	_, weekNum := monday.ISOWeek()
	fileName := fmt.Sprintf("周报_%s_W%02d.docx", monday.Format("2006"), weekNum)
	filePath := filepath.Join(outputDir, fileName)

	if err := SaveDocx(content, filePath); err != nil {
		return "", fmt.Errorf("保存文档失败: %w", err)
	}

	g.db.InsertReport("weekly", monday.Format("2006-01-02"), friday.Format("2006-01-02"), filePath)

	log.Printf("周报已生成: %s", filePath)
	return filePath, nil
}

func (g *Generator) getDayRange(date time.Time) (time.Time, time.Time) {
	startH, startM, _ := g.cfg.GetWorkStart()
	endH, endM, _ := g.cfg.GetWorkEnd()

	start := time.Date(date.Year(), date.Month(), date.Day(), startH, startM, 0, 0, date.Location())
	end := time.Date(date.Year(), date.Month(), date.Day(), endH, endM, 0, 0, date.Location())
	return start, end
}

func (g *Generator) formatDailyData(date time.Time, summaries []storage.AppUsageSummary, visits []storage.BrowserVisit) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## 今日活动数据（%s）\n\n", date.Format("2006-01-02")))

	// 应用使用统计
	sb.WriteString("### 应用使用统计\n\n")
	sb.WriteString("| 应用 | 使用时长 | 主要窗口标题 |\n")
	sb.WriteString("|------|----------|-------------|\n")
	for _, s := range summaries {
		titles := g.summarizeTitles(s.Titles, 3)
		sb.WriteString(fmt.Sprintf("| %s | %s | %s |\n",
			s.ProcessName,
			formatDuration(s.TotalSec),
			titles,
		))
	}

	// 浏览器访问记录
	if len(visits) > 0 {
		sb.WriteString("\n### 浏览器访问记录（Top 20）\n\n")
		sb.WriteString("| 时间 | 网址 | 标题 |\n")
		sb.WriteString("|------|------|------|\n")

		// 按时间倒序，取 Top 20
		sort.Slice(visits, func(i, j int) bool {
			return visits[i].VisitedAt.After(visits[j].VisitedAt)
		})
		limit := 20
		if len(visits) < limit {
			limit = len(visits)
		}
		for i := 0; i < limit; i++ {
			v := visits[i]
			domain := extractDomain(v.URL)
			sb.WriteString(fmt.Sprintf("| %s | %s | %s |\n",
				v.VisitedAt.Format("15:04"),
				domain,
				truncate(v.Title, 50),
			))
		}
	}

	return sb.String()
}

func (g *Generator) formatDaySummary(date time.Time, summaries []storage.AppUsageSummary, visits []storage.BrowserVisit) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("### %s（%s）\n\n", getWeekdayName(date.Weekday()), date.Format("2006-01-02")))

	for _, s := range summaries {
		titles := g.summarizeTitles(s.Titles, 2)
		sb.WriteString(fmt.Sprintf("- **%s**: 使用 %s，主要做: %s\n",
			s.ProcessName,
			formatDuration(s.TotalSec),
			titles,
		))
	}

	if len(visits) > 0 {
		// 统计域名访问频率
		domainCount := make(map[string]int)
		for _, v := range visits {
			domainCount[extractDomain(v.URL)]++
		}
		var domains []string
		for d := range domainCount {
			domains = append(domains, d)
		}
		sort.Slice(domains, func(i, j int) bool {
			return domainCount[domains[i]] > domainCount[domains[j]]
		})
		limit := 5
		if len(domains) < limit {
			limit = len(domains)
		}
		sb.WriteString("- 浏览器主要访问: ")
		for i := 0; i < limit; i++ {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("%s(%d次)", domains[i], domainCount[domains[i]]))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func (g *Generator) summarizeTitles(titles []string, maxTitles int) string {
	seen := make(map[string]bool)
	var unique []string
	for _, t := range titles {
		t = strings.TrimSpace(t)
		if t == "" || t == "(无标题)" {
			continue
		}
		if !seen[t] {
			seen[t] = true
			unique = append(unique, t)
		}
	}

	if len(unique) > maxTitles {
		unique = unique[:maxTitles]
	}

	// []rune 长度才是人眼看到的字符数，也保证不会把多字节字符切一半
	return truncate(strings.Join(unique, ", "), 80)
}

func formatDuration(seconds int) string {
	h := seconds / 3600
	m := (seconds % 3600) / 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}

func extractDomain(url string) string {
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "www.")
	if idx := strings.Index(url, "/"); idx > 0 {
		url = url[:idx]
	}
	return url
}

// truncate 按字符数（rune）截断字符串。
// 不能用 len(s) / s[:n] 直接切字节：中文是 3 字节，切开后会得到
// 非法 UTF-8，写进 DOCX 就变成乱码。
func truncate(s string, maxLen int) string {
	if maxLen <= 3 {
		return ""
	}
	r := []rune(s)
	if len(r) <= maxLen {
		return s
	}
	return string(r[:maxLen-3]) + "..."
}

func getWeekdayName(w time.Weekday) string {
	names := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}
	return names[w]
}

// OpenFile 用系统默认程序打开文件
func OpenFile(path string) error {
	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", path)
	return cmd.Run()
}

// WeekMonday 返回 t 所在自然周的周一（周一为一周起点）。
func WeekMonday(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7 // 周日算第 7 天
	}
	return t.AddDate(0, 0, 1-weekday)
}

// ShouldAutoGenerate 判断自动任务是否还需要生成某周期的报告。
//
// 定时任务原来只靠内存变量 lastDailyDate 去重，进程重启后当天会再生成
// 一份（追加一条 generated_reports 记录 + 覆盖 docx）。这里改成查库。
// 只用于自动触发的场景：用户手动点"生成日报"仍应允许覆盖重生成。
func ShouldAutoGenerate(db *storage.DB, reportType string, periodStart time.Time) bool {
	exists, err := db.HasReport(reportType, periodStart.Format("2006-01-02"))
	if err != nil {
		// 查不到就宁可生成，不能因为查询失败丢掉报告
		log.Printf("查询报告生成记录失败，按未生成处理: %v", err)
		return true
	}
	return !exists
}
