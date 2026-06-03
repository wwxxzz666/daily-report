package main

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"time"

	"daily-report/internal/browser"
	"daily-report/internal/config"
	"daily-report/internal/monitor"
	"daily-report/internal/report"
	"daily-report/internal/scheduler"
	"daily-report/internal/storage"

	"github.com/getlantern/systray"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App 是 Wails 的 binding 层，前端通过调用 App 的方法与后端交互
type App struct {
	ctx        context.Context
	db         *storage.DB
	sampler    *monitor.Sampler
	sched      *scheduler.Scheduler
	reallyQuit bool // true 时 OnBeforeClose 允许关闭
}

// Status 返回当前运行状态
type Status struct {
	IsRunning    bool   `json:"isRunning"`
	Countdown    string `json:"countdown"`
	RecordedTime string `json:"recordedTime"`
	AppCount     int    `json:"appCount"`
}

// TodaySummary 今日活动汇总
type TodaySummary struct {
	Apps []AppUsage `json:"apps"`
}

// AppUsage 单个应用使用情况
type AppUsage struct {
	ProcessName string `json:"processName"`
	TotalSec    int    `json:"totalSec"`
	Duration    string `json:"duration"`
}

// ReportInfo 报告信息
type ReportInfo struct {
	FileName    string `json:"fileName"`
	ReportType  string `json:"reportType"`
	GeneratedAt string `json:"generatedAt"`
	FilePath    string `json:"filePath"`
}

// ProviderInfo LLM 提供商信息
type ProviderInfo struct {
	Name         string `json:"name"`
	Label        string `json:"label"`
	DefaultModel string `json:"defaultModel"`
	NeedsKey     bool   `json:"needsKey"`
}

// SaveConfigRequest 前端保存配置的请求
type SaveConfigRequest struct {
	LLM      *LLMConfigRequest      `json:"llm"`
	WorkTime *WorkTimeConfigRequest `json:"work_time"`
	Report   *ReportConfigRequest   `json:"report"`
}

type LLMConfigRequest struct {
	Provider string `json:"provider"`
	APIKey   string `json:"api_key"`
	Model    string `json:"model"`
}

type WorkTimeConfigRequest struct {
	Start    string `json:"start"`
	End      string `json:"end"`
	Weekdays []int  `json:"weekdays"`
}

type ReportConfigRequest struct {
	WeeklyDay int `json:"weekly_day"`
}

// startup 是 Wails 生命周期回调
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	log.Println("startup 回调开始")

	cfg := config.Get()

	// 初始化数据库
	db, err := storage.Init(cfg.DBPath())
	if err != nil {
		log.Printf("初始化数据库失败: %v", err)
		return
	}
	a.db = db
	log.Println("数据库初始化成功")

	// 启动采样器
	a.sampler = monitor.NewSampler(db)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("采样器 panic: %v", r)
			}
		}()
		a.sampler.Start()
	}()

	// 启动定时调度
	a.sched = scheduler.New()
	a.sched.OnDailyReport(func() {
		c := config.Get()
		gen := report.NewGenerator(db, c)
		if _, err := gen.GenerateDailyReport(time.Now()); err != nil {
			log.Printf("自动生成日报失败: %v", err)
		}
	})
	a.sched.OnWeeklyReport(func() {
		c := config.Get()
		gen := report.NewGenerator(db, c)
		if _, err := gen.GenerateWeeklyReport(time.Now()); err != nil {
			log.Printf("自动生成周报失败: %v", err)
		}
	})
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("调度器 panic: %v", r)
			}
		}()
		a.sched.Start()
	}()

	// 同步浏览器历史
	if cfg.Browser.Enabled {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("浏览器同步 panic: %v", r)
				}
			}()
			browser.SyncAll(db, cfg.Browser.Browsers)
		}()
	}

	// 启动系统托盘
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("系统托盘 panic: %v", r)
			}
		}()
		a.runTray()
	}()

	log.Println("startup 回调完成")
}

// runTray 运行系统托盘
func (a *App) runTray() {
	log.Println("系统托盘初始化中...")
	systray.Run(func() {
		log.Println("系统托盘 onReady 回调")
		systray.SetTitle("日报")
		systray.SetTooltip("日报助手 - 点击显示主窗口")

		mShow := systray.AddMenuItem("显示主窗口", "显示主窗口")
		systray.AddSeparator()
		mQuit := systray.AddMenuItem("退出", "退出日报助手")

		go func() {
			for {
				select {
				case <-mShow.ClickedCh:
					wailsRuntime.WindowShow(a.ctx)
				case <-mQuit.ClickedCh:
					a.reallyQuit = true
					wailsRuntime.Quit(a.ctx)
					return
				}
			}
		}()
	}, func() {
		log.Println("系统托盘 onExit 回调")
	})
	log.Println("系统托盘已退出")
}

// shouldClose 关闭按钮 -> 隐藏到托盘；托盘退出 -> 真正关闭
// Wails OnBeforeClose: 返回 true 取消关闭，返回 false 允许关闭
func (a *App) shouldClose(_ context.Context) bool {
	if a.reallyQuit {
		log.Println("托盘退出，允许关闭窗口")
		return false // 允许关闭
	}
	log.Println("关闭按钮点击，隐藏到托盘")
	wailsRuntime.WindowHide(a.ctx)
	return true // 取消关闭，仅隐藏窗口
}

// shutdown 清理资源
func (a *App) shutdown(_ context.Context) {
	log.Println("shutdown 回调开始")
	systray.Quit()
	if a.sampler != nil {
		a.sampler.Stop()
	}
	if a.sched != nil {
		a.sched.Stop()
	}
	if a.db != nil {
		a.db.Close()
	}
	log.Println("shutdown 回调完成")
}

// GetConfig 获取当前配置
func (a *App) GetConfig() *config.Config {
	return config.Get()
}

// SaveConfig 保存配置
func (a *App) SaveConfig(req *SaveConfigRequest) error {
	cfg := config.Get()
	if cfg == nil {
		return fmt.Errorf("配置未加载")
	}

	if req.LLM != nil {
		cfg.LLM.Provider = req.LLM.Provider
		cfg.LLM.APIKey = req.LLM.APIKey
		cfg.LLM.Model = req.LLM.Model
	}
	if req.WorkTime != nil {
		cfg.WorkTime.Start = req.WorkTime.Start
		cfg.WorkTime.End = req.WorkTime.End
		cfg.WorkTime.Weekdays = req.WorkTime.Weekdays
	}
	if req.Report != nil {
		cfg.Report.WeeklyDay = req.Report.WeeklyDay
	}

	return config.Save(cfg)
}

// GetStatus 获取运行状态
func (a *App) GetStatus() (*Status, error) {
	if a.db == nil {
		return &Status{}, nil
	}

	cfg := config.Get()
	now := time.Now()
	isRunning := cfg.IsWorkingTime(now)

	countdown := "非工作时间"
	if isRunning {
		endH, endM, _ := cfg.GetWorkEnd()
		endTime := time.Date(now.Year(), now.Month(), now.Day(), endH, endM, 0, 0, now.Location())
		remaining := endTime.Sub(now)
		if remaining > 0 {
			h := int(remaining.Hours())
			m := int(remaining.Minutes()) % 60
			countdown = fmt.Sprintf("距下班 %dh %dm", h, m)
		}
	}

	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	summaries, _ := a.db.GetAppUsageSummary(start, now)
	summaries = mergeCurrentActivityIntoSummaries(summaries, a.currentActivity())

	var totalSec int
	for _, s := range summaries {
		totalSec += s.TotalSec
	}

	return &Status{
		IsRunning:    isRunning,
		Countdown:    countdown,
		RecordedTime: formatDuration(totalSec),
		AppCount:     len(summaries),
	}, nil
}

// GetTodaySummary 获取今日活动汇总
func (a *App) GetTodaySummary() (*TodaySummary, error) {
	if a.db == nil {
		return &TodaySummary{}, nil
	}

	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	summaries, err := a.db.GetAppUsageSummary(start, now)
	if err != nil {
		return nil, err
	}
	summaries = mergeCurrentActivityIntoSummaries(summaries, a.currentActivity())

	var apps []AppUsage
	for _, s := range summaries {
		apps = append(apps, AppUsage{
			ProcessName: s.ProcessName,
			TotalSec:    s.TotalSec,
			Duration:    formatDuration(s.TotalSec),
		})
	}

	return &TodaySummary{Apps: apps}, nil
}

// GenerateDailyReport 生成日报
func (a *App) GenerateDailyReport() (string, error) {
	cfg := config.Get()
	gen := report.NewGenerator(a.db, cfg)
	return gen.GenerateDailyReport(time.Now())
}

// GenerateWeeklyReport 生成周报
func (a *App) GenerateWeeklyReport() (string, error) {
	cfg := config.Get()
	gen := report.NewGenerator(a.db, cfg)
	return gen.GenerateWeeklyReport(time.Now())
}

// GetReportHistory 获取报告历史
func (a *App) GetReportHistory() ([]ReportInfo, error) {
	if a.db == nil {
		return nil, nil
	}

	rows, err := a.db.Query(`
		SELECT report_type, period_start, period_end, file_path, generated_at
		FROM generated_reports
		ORDER BY generated_at DESC
		LIMIT 50`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []ReportInfo
	for rows.Next() {
		var reportType, periodStart, periodEnd, filePath, generatedAt string
		if err := rows.Scan(&reportType, &periodStart, &periodEnd, &filePath, &generatedAt); err != nil {
			continue
		}
		reports = append(reports, ReportInfo{
			FileName:    filepath.Base(filePath),
			ReportType:  reportType,
			GeneratedAt: generatedAt,
			FilePath:    filePath,
		})
	}
	return reports, nil
}

// OpenFile 用系统默认程序打开文件
func (a *App) OpenFile(path string) error {
	return exec.Command("rundll32", "url.dll,FileProtocolHandler", path).Run()
}

// OpenReportsDir 打开报告目录
func (a *App) OpenReportsDir() error {
	cfg := config.Get()
	dir := cfg.GetReportOutputDir()
	return exec.Command("explorer", dir).Run()
}

// TestLLMConnection 测试 LLM 连接
func (a *App) TestLLMConnection() (string, error) {
	cfg := config.Get()
	_, err := report.CallLLM(cfg, "请回复'连接成功'。", "测试连接")
	if err != nil {
		return "", err
	}
	return "连接成功", nil
}

// GetProviders 获取 LLM 提供商列表
func (a *App) GetProviders() []ProviderInfo {
	presets := config.SupportedProviders()
	var result []ProviderInfo
	for _, p := range presets {
		label := p.Name
		switch p.Name {
		case "deepseek":
			label = "DeepSeek（推荐）"
		case "openai":
			label = "OpenAI"
		case "qwen":
			label = "通义千问"
		case "moonshot":
			label = "Moonshot (Kimi)"
		case "zhipu":
			label = "智谱清言"
		case "ollama":
			label = "Ollama（本地）"
		case "lmstudio":
			label = "LM Studio（本地）"
		}
		result = append(result, ProviderInfo{
			Name:         p.Name,
			Label:        label,
			DefaultModel: p.DefaultModel,
			NeedsKey:     p.NeedsAPIKey,
		})
	}
	return result
}

func formatDuration(seconds int) string {
	h := seconds / 3600
	m := (seconds % 3600) / 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	if m > 0 {
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%ds", seconds)
}
