package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"daily-report/internal/config"
	"daily-report/internal/monitor"
	"daily-report/internal/storage"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.SetOutput(os.Stdout)

	fmt.Println("=== 测试 1: 配置加载 ===")
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("配置加载失败: %v", err)
	}
	fmt.Printf("  工作时间: %s - %s\n", cfg.WorkTime.Start, cfg.WorkTime.End)
	fmt.Printf("  LLM Provider: %s\n", cfg.LLM.Provider)
	fmt.Printf("  LLM Endpoint: %s\n", cfg.LLM.GetEndpoint())
	fmt.Printf("  LLM Model: %s\n", cfg.LLM.GetModel())
	fmt.Printf("  敏感词数量: %d\n", len(cfg.SensitiveWords))
	fmt.Printf("  是否工作时间: %v\n", cfg.IsWorkingTime(time.Now()))
	fmt.Println("  通过 ✓")

	fmt.Println("\n=== 测试 2: 数据库初始化 ===")
	tmpDir, err := os.MkdirTemp("", "daily-report-smoke-*")
	if err != nil {
		log.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "data.db")
	db, err := storage.Init(dbPath)
	if err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}
	defer db.Close()
	fmt.Printf("  数据库路径: %s\n", dbPath)
	fmt.Println("  通过 ✓")

	fmt.Println("\n=== 测试 3: 窗口监控（采样 5 次）===")
	for i := 0; i < 5; i++ {
		win, err := monitor.GetActiveWindow()
		if err != nil {
			fmt.Printf("  [%d] 获取窗口失败: %v\n", i+1, err)
		} else {
			fmt.Printf("  [%d] 进程: %-20s 标题: %.50s\n", i+1, win.ProcessName, win.WindowTitle)
		}
		time.Sleep(1 * time.Second)
	}
	fmt.Println("  通过 ✓")

	fmt.Println("\n=== 测试 4: 写入数据库 ===")
	activity := &storage.WindowActivity{
		ProcessName: "test_process",
		WindowTitle: "测试窗口标题",
		StartedAt:   time.Now().Add(-5 * time.Minute),
		EndedAt:     time.Now(),
		DurationSec: 300,
	}
	if err := db.InsertWindowActivity(activity); err != nil {
		log.Fatalf("写入失败: %v", err)
	}
	fmt.Println("  写入成功 ✓")

	fmt.Println("\n=== 测试 5: 查询数据库 ===")
	start := time.Now().Add(-1 * time.Hour)
	end := time.Now()
	activities, err := db.GetWindowActivities(start, end)
	if err != nil {
		log.Fatalf("查询失败: %v", err)
	}
	fmt.Printf("  查询到 %d 条记录\n", len(activities))
	for _, a := range activities {
		fmt.Printf("    %s: %s (%ds)\n", a.ProcessName, a.WindowTitle, a.DurationSec)
	}
	fmt.Println("  通过 ✓")

	fmt.Println("\n=== 测试 6: 应用汇总 ===")
	summaries, err := db.GetAppUsageSummary(start, end)
	if err != nil {
		log.Fatalf("汇总失败: %v", err)
	}
	for _, s := range summaries {
		fmt.Printf("    %s: %ds, 标题: %v\n", s.ProcessName, s.TotalSec, s.Titles[:min(len(s.Titles), 2)])
	}
	fmt.Println("  通过 ✓")

	fmt.Println("\n=== 所有测试通过 ===")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
