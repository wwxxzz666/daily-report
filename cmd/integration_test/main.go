package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"daily-report/internal/config"
	"daily-report/internal/report"
	"daily-report/internal/storage"

	"net/http"
	"net/http/httptest"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Println("====== 日报助手完整链路集成测试 ======")

	passed := 0
	failed := 0

	// -------------------------------------------------------
	// 步骤 1: 准备环境（临时目录、数据库、Mock LLM）
	// -------------------------------------------------------
	log.Println("[步骤1] 准备测试环境...")

	tmpDir, err := os.MkdirTemp("", "daily-report-integration-*")
	if err != nil {
		log.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	log.Printf("  临时目录: %s", tmpDir)

	dbPath := filepath.Join(tmpDir, "data.db")
	db, err := storage.Init(dbPath)
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	defer db.Close()
	log.Println("  数据库初始化完成")

	// Mock LLM 服务器：返回格式化的日报 markdown
	dailyMarkdown := `# 工作日报（2026-05-27）

## 今日完成

- 完成了前端页面的集成测试
- 使用 VS Code 进行代码开发和调试
- 通过浏览器查阅技术文档和 GitHub 项目
- 修复了单元测试中的问题

## 进行中

- 周报功能的端到端验证

## 明日计划

- 继续完善测试覆盖率
- 优化报告生成性能

## 备注

- 今日主要使用 VS Code 和 Chrome 浏览器`

	weeklyMarkdown := `# 工作周报（2026 第22周）

## 本周完成

1. 完成了 Wails 框架集成
2. 实现了系统托盘最小化功能
3. 修复了 TLS 证书验证问题
4. 完善了单元测试和集成测试

## 下周计划

- 性能优化
- 文档完善

## 总结

本周主要完成了项目的核心功能开发和测试验证。`

	llmCallCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		llmCallCount++

		// 验证请求格式
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)

		messages := req["messages"].([]interface{})
		userMsg := messages[len(messages)-1].(map[string]interface{})["content"].(string)

		var content string
		if strings.Contains(userMsg, "本周活动数据") {
			content = weeklyMarkdown
		} else {
			content = dailyMarkdown
		}

		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]interface{}{
						"content": content,
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()
	log.Printf("  Mock LLM 服务器启动: %s", server.URL)

	// 配置
	cfg := &config.Config{
		WorkTime: config.WorkTimeConfig{
			Start:    "09:00",
			End:      "18:00",
			Weekdays: []int{1, 2, 3, 4, 5},
		},
		Browser: config.BrowserConfig{Enabled: false},
		LLM: config.LLMConfig{
			Endpoint:    server.URL + "/v1/chat/completions",
			APIKey:      "integration-test-key",
			Model:       "test-model",
			Temperature: 0.7,
			MaxTokens:   4096,
			Timeout:     "30s",
		},
		Report: config.ReportConfig{
			OutputDir:  filepath.Join(tmpDir, "reports"),
			WeeklyDay:  5,
			WeeklyTime: "18:10",
		},
	}
	config.SetInstance(cfg)

	// -------------------------------------------------------
	// 步骤 2: 模拟一天的工作活动数据
	// -------------------------------------------------------
	log.Println("[步骤2] 插入模拟活动数据...")

	now := time.Now()
	// 构造今天工作时间内的数据
	baseTime := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, now.Location())

	activities := []struct {
		proc   string
		title  string
		offset time.Duration
		dur    time.Duration
		secs   int
	}{
		{"code.exe", "main.go - Visual Studio Code", 0, 2 * time.Hour, 7200},
		{"code.exe", "app.go - Visual Studio Code", 2 * time.Hour, 1 * time.Hour, 3600},
		{"chrome.exe", "GitHub - Pull Requests - Google Chrome", 3 * time.Hour, 30 * time.Minute, 1800},
		{"chrome.exe", "Stack Overflow - Questions - Google Chrome", 210 * time.Minute, 30 * time.Minute, 1800},
		{"code.exe", "report_test.go - Visual Studio Code", 4*time.Hour + 30*time.Minute, 1 * time.Hour, 3600},
		{"explorer.exe", "项目文件夹", 5*time.Hour + 30*time.Minute, 5 * time.Minute, 300},
		{"wechat.exe", "微信", 6 * time.Hour, 15 * time.Minute, 900},
		{"code.exe", "generator.go - Visual Studio Code", 6*time.Hour + 30*time.Minute, 1*time.Hour + 30*time.Minute, 5400},
		{"chrome.exe", "DeepSeek API Documentation - Google Chrome", 8 * time.Hour, 30 * time.Minute, 1800},
	}

	for i, a := range activities {
		startedAt := baseTime.Add(a.offset)
		endedAt := startedAt.Add(a.dur)
		act := &storage.WindowActivity{
			ProcessName: a.proc,
			WindowTitle: a.title,
			StartedAt:   startedAt,
			EndedAt:     endedAt,
			DurationSec: a.secs,
		}
		if err := db.InsertWindowActivity(act); err != nil {
			log.Fatalf("  插入活动 %d 失败: %v", i, err)
		}
	}
	log.Printf("  已插入 %d 条活动记录", len(activities))

	// 同时插入模拟浏览器访问记录
	visits := []storage.BrowserVisit{
		{Browser: "chrome", URL: "https://github.com/user/repo/pull/42", Title: "Fix: 修复日报生成链路", VisitedAt: baseTime.Add(3 * time.Hour), SourceVisitID: 1},
		{Browser: "chrome", URL: "https://stackoverflow.com/questions/12345", Title: "How to mock HTTP server in Go", VisitedAt: baseTime.Add(3*time.Hour + 15*time.Minute), SourceVisitID: 2},
		{Browser: "chrome", URL: "https://pkg.go.dev/net/http/httptest", Title: "httptest package - GoDoc", VisitedAt: baseTime.Add(3*time.Hour + 30*time.Minute), SourceVisitID: 3},
		{Browser: "chrome", URL: "https://api.deepseek.com/docs", Title: "DeepSeek API Documentation", VisitedAt: baseTime.Add(8 * time.Hour), SourceVisitID: 4},
	}
	inserted, err := db.InsertBrowserVisits(visits)
	if err != nil {
		log.Printf("  插入浏览器记录失败（非致命）: %v", err)
	} else {
		log.Printf("  已插入 %d 条浏览器记录", inserted)
	}

	// -------------------------------------------------------
	// 步骤 3: 验证数据库查询
	// -------------------------------------------------------
	log.Println("[步骤3] 验证数据库查询...")

	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, now.Location())
	dayEnd := time.Date(now.Year(), now.Month(), now.Day(), 18, 0, 0, 0, now.Location())

	queriedActivities, err := db.GetWindowActivities(dayStart, dayEnd)
	if err != nil {
		log.Printf("  FAIL: GetWindowActivities 错误: %v", err)
		failed++
	} else if len(queriedActivities) != len(activities) {
		log.Printf("  FAIL: 查询到 %d 条活动，期望 %d 条", len(queriedActivities), len(activities))
		failed++
	} else {
		log.Printf("  PASS: 查询到 %d 条活动记录", len(queriedActivities))
		passed++
	}

	summaries, err := db.GetAppUsageSummary(dayStart, dayEnd)
	if err != nil {
		log.Printf("  FAIL: GetAppUsageSummary 错误: %v", err)
		failed++
	} else {
		log.Printf("  PASS: 查询到 %d 个应用汇总", len(summaries))
		passed++
		for _, s := range summaries {
			log.Printf("    - %s: %d秒 (%d条标题)", s.ProcessName, s.TotalSec, len(s.Titles))
		}
	}

	browserVisits, err := db.GetBrowserVisits(dayStart, dayEnd)
	if err != nil {
		log.Printf("  FAIL: GetBrowserVisits 错误: %v", err)
		failed++
	} else if len(browserVisits) != len(visits) {
		log.Printf("  FAIL: 查询到 %d 条浏览器记录，期望 %d 条", len(browserVisits), len(visits))
		failed++
	} else {
		log.Printf("  PASS: 查询到 %d 条浏览器记录", len(browserVisits))
		passed++
	}

	// -------------------------------------------------------
	// 步骤 4: 生成日报
	// -------------------------------------------------------
	log.Println("[步骤4] 生成日报...")

	gen := report.NewGenerator(db, cfg)
	llmCallCount = 0

	reportPath, err := gen.GenerateDailyReport(now)
	if err != nil {
		log.Printf("  FAIL: 生成日报失败: %v", err)
		failed++
	} else {
		log.Printf("  PASS: 日报已生成: %s", reportPath)
		passed++

		// 验证文件存在且大小合理
		info, err := os.Stat(reportPath)
		if err != nil {
			log.Printf("  FAIL: 无法读取日报文件: %v", err)
			failed++
		} else if info.Size() == 0 {
			log.Println("  FAIL: 日报文件为空")
			failed++
		} else {
			log.Printf("  PASS: 日报文件大小: %d bytes", info.Size())
			passed++
		}

		// 验证文件名格式
		expectedName := fmt.Sprintf("日报_%s.docx", now.Format("2006-01-02"))
		actualName := filepath.Base(reportPath)
		if actualName != expectedName {
			log.Printf("  FAIL: 文件名 '%s' 不符合预期 '%s'", actualName, expectedName)
			failed++
		} else {
			log.Printf("  PASS: 文件名格式正确: %s", actualName)
			passed++
		}
	}

	// 验证 LLM 被调用了
	if llmCallCount != 1 {
		log.Printf("  FAIL: LLM 调用次数 %d，期望 1", llmCallCount)
		failed++
	} else {
		log.Printf("  PASS: LLM 正确调用了 1 次")
		passed++
	}

	// 验证报告记录已写入数据库
	rows, _ := db.Query("SELECT report_type, file_path FROM generated_reports WHERE report_type = 'daily'")
	dailyReports := 0
	for rows.Next() {
		dailyReports++
	}
	rows.Close()
	if dailyReports != 1 {
		log.Printf("  FAIL: generated_reports 中有 %d 条日报记录，期望 1", dailyReports)
		failed++
	} else {
		log.Println("  PASS: 报告记录已写入数据库")
		passed++
	}

	// -------------------------------------------------------
	// 步骤 5: 插入一周数据并生成周报
	// -------------------------------------------------------
	log.Println("[步骤5] 准备一周数据并生成周报...")

	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	monday := now.AddDate(0, 0, 1-weekday)

	// 为周一到周四各添加数据（周五即今天已有数据）
	for d := 0; d < 4; d++ {
		day := monday.AddDate(0, 0, d)
		dayStart := time.Date(day.Year(), day.Month(), day.Day(), 9, 30, 0, 0, day.Location())
		db.InsertWindowActivity(&storage.WindowActivity{
			ProcessName: "code.exe",
			WindowTitle: fmt.Sprintf("feature-%d - Visual Studio Code", d+1),
			StartedAt:   dayStart,
			EndedAt:     dayStart.Add(3 * time.Hour),
			DurationSec: 10800,
		})
		db.InsertWindowActivity(&storage.WindowActivity{
			ProcessName: "chrome.exe",
			WindowTitle: "技术文档 - Google Chrome",
			StartedAt:   dayStart.Add(4 * time.Hour),
			EndedAt:     dayStart.Add(5 * time.Hour),
			DurationSec: 3600,
		})
	}
	log.Println("  已添加周一至周四的历史数据")

	llmCallCount = 0
	weeklyPath, err := gen.GenerateWeeklyReport(now)
	if err != nil {
		log.Printf("  FAIL: 生成周报失败: %v", err)
		failed++
	} else {
		log.Printf("  PASS: 周报已生成: %s", weeklyPath)
		passed++

		info, _ := os.Stat(weeklyPath)
		if info != nil {
			log.Printf("  PASS: 周报文件大小: %d bytes", info.Size())
			passed++
		}

		// 验证周报文件名包含年份和周数
		weeklyName := filepath.Base(weeklyPath)
		if !strings.HasPrefix(weeklyName, "周报_") || !strings.Contains(weeklyName, ".docx") {
			log.Printf("  FAIL: 周报文件名格式错误: %s", weeklyName)
			failed++
		} else {
			log.Printf("  PASS: 周报文件名: %s", weeklyName)
			passed++
		}
	}

	if llmCallCount != 1 {
		log.Printf("  FAIL: 周报 LLM 调用次数 %d，期望 1", llmCallCount)
		failed++
	} else {
		log.Printf("  PASS: 周报 LLM 正确调用了 1 次")
		passed++
	}

	// -------------------------------------------------------
	// 步骤 6: 验证输出目录结构
	// -------------------------------------------------------
	log.Println("[步骤6] 验证输出目录...")

	entries, err := os.ReadDir(cfg.Report.OutputDir)
	if err != nil {
		log.Printf("  FAIL: 读取输出目录失败: %v", err)
		failed++
	} else {
		log.Printf("  PASS: 输出目录中有 %d 个文件", len(entries))
		passed++
		for _, e := range entries {
			log.Printf("    - %s", e.Name())
		}
	}

	// -------------------------------------------------------
	// 步骤 7: 验证边界情况
	// -------------------------------------------------------
	log.Println("[步骤7] 验证边界情况...")

	// 无数据日期应返回错误
	emptyDate := time.Date(2020, 1, 1, 10, 0, 0, 0, now.Location())
	_, err = gen.GenerateDailyReport(emptyDate)
	if err == nil {
		log.Println("  FAIL: 无数据日期应返回错误")
		failed++
	} else if !strings.Contains(err.Error(), "没有找到") {
		log.Printf("  FAIL: 错误信息不包含 '没有找到': %v", err)
		failed++
	} else {
		log.Printf("  PASS: 无数据日期正确返回错误: %v", err)
		passed++
	}

	// -------------------------------------------------------
	// 结果汇总
	// -------------------------------------------------------
	log.Println("======================================")
	log.Printf("测试结果: %d PASSED, %d FAILED", passed, failed)
	log.Println("======================================")

	if failed > 0 {
		os.Exit(1)
	}
	log.Println("所有集成测试通过！")
}
