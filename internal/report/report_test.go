package report

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"daily-report/internal/config"
	"daily-report/internal/storage"
)

// ========== 1. CallLLM: Mock HTTP Server 测试 ==========

func TestCallLLM_Success(t *testing.T) {
	// 创建 mock API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求头
		if r.Header.Get("Authorization") != "Bearer test-key-123" {
			t.Errorf("Expected Authorization header 'Bearer test-key-123', got '%s'", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type 'application/json'")
		}

		// 验证请求体
		var req ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}
		if req.Model != "test-model" {
			t.Errorf("Expected model 'test-model', got '%s'", req.Model)
		}

		// 返回成功响应
		resp := ChatResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Content string `json:"content"`
				}{Content: "这是一份测试日报内容"}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &config.Config{
		LLM: config.LLMConfig{
			Endpoint:    server.URL + "/v1/chat/completions",
			APIKey:      "test-key-123",
			Model:       "test-model",
			Temperature: 0.7,
			MaxTokens:   4096,
			Timeout:     "10s",
		},
	}

	result, err := CallLLM(cfg, "system prompt", "user message")
	if err != nil {
		t.Fatalf("CallLLM failed: %v", err)
	}
	if result != "这是一份测试日报内容" {
		t.Errorf("Expected '这是一份测试日报内容', got '%s'", result)
	}
}

func TestCallLLM_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"invalid api key"}}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		LLM: config.LLMConfig{
			Endpoint:    server.URL + "/v1/chat/completions",
			APIKey:      "bad-key",
			Model:       "test",
			Temperature: 0.7,
			MaxTokens:   100,
			Timeout:     "5s",
		},
	}

	_, err := CallLLM(cfg, "system", "user")
	if err == nil {
		t.Fatal("Expected error for 401 response, got nil")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("Error should mention 401, got: %v", err)
	}
}

func TestCallLLM_EmptyEndpoint(t *testing.T) {
	cfg := &config.Config{
		LLM: config.LLMConfig{
			Endpoint: "",
			APIKey:   "key",
		},
	}
	_, err := CallLLM(cfg, "sys", "user")
	if err == nil {
		t.Fatal("Expected error for empty endpoint")
	}
	if !strings.Contains(err.Error(), "未配置") {
		t.Errorf("Error should mention config issue, got: %v", err)
	}
}

// ========== 2. 报告数据格式化测试 ==========

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		seconds  int
		expected string
	}{
		{0, "0m"},
		{30, "0m"},
		{60, "1m"},
		{90, "1m"},
		{3600, "1h 0m"},
		{3661, "1h 1m"},
		{7200, "2h 0m"},
		{28800, "8h 0m"},
	}
	for _, tt := range tests {
		result := formatDuration(tt.seconds)
		if result != tt.expected {
			t.Errorf("formatDuration(%d) = '%s', want '%s'", tt.seconds, result, tt.expected)
		}
	}
}

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://www.google.com/search?q=test", "google.com"},
		{"http://github.com/user/repo", "github.com"},
		{"https://mail.example.com/inbox", "mail.example.com"},
		{"https://api.deepseek.com/v1/chat", "api.deepseek.com"},
		{"www.simple.com/page", "simple.com"},
	}
	for _, tt := range tests {
		result := extractDomain(tt.url)
		if result != tt.expected {
			t.Errorf("extractDomain('%s') = '%s', want '%s'", tt.url, result, tt.expected)
		}
	}
}

func TestSummarizeTitles(t *testing.T) {
	cfg := &config.Config{}
	gen := NewGenerator(nil, cfg)

	tests := []struct {
		name     string
		titles   []string
		max      int
		expected string
	}{
		{
			name:     "normal titles",
			titles:   []string{"编写代码", "调试程序", "代码审查"},
			max:      3,
			expected: "编写代码, 调试程序, 代码审查",
		},
		{
			name:     "truncate to max",
			titles:   []string{"A", "B", "C", "D"},
			max:      2,
			expected: "A, B",
		},
		{
			name:     "skip empty and no-title",
			titles:   []string{"有效标题", "", "(无标题)", "另一个标题"},
			max:      3,
			expected: "有效标题, 另一个标题",
		},
		{
			name:     "empty input",
			titles:   []string{},
			max:      3,
			expected: "",
		},
		{
			name:     "deduplicate",
			titles:   []string{"相同标题", "相同标题", "不同标题"},
			max:      3,
			expected: "相同标题, 不同标题",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.summarizeTitles(tt.titles, tt.max)
			if result != tt.expected {
				t.Errorf("summarizeTitles() = '%s', want '%s'", result, tt.expected)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	if result := truncate("short", 10); result != "short" {
		t.Errorf("truncate short string should return as-is, got '%s'", result)
	}
	long := strings.Repeat("a", 100)
	result := truncate(long, 50)
	if len(result) != 50 {
		t.Errorf("truncate should return %d chars, got %d", 50, len(result))
	}
	if !strings.HasSuffix(result, "...") {
		t.Errorf("truncated string should end with '...'")
	}
}

func TestGetWeekdayName(t *testing.T) {
	tests := []struct {
		day      time.Weekday
		expected string
	}{
		{time.Monday, "周一"},
		{time.Friday, "周五"},
		{time.Sunday, "周日"},
		{time.Saturday, "周六"},
	}
	for _, tt := range tests {
		result := getWeekdayName(tt.day)
		if result != tt.expected {
			t.Errorf("getWeekdayName(%v) = '%s', want '%s'", tt.day, result, tt.expected)
		}
	}
}

// ========== 3. Docx 生成测试 ==========

func TestSaveDocx(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_report.docx")

	content := `# 测试日报

## 工作总结

### 今日完成

- 完成了单元测试
- 修复了日志编码问题

| 应用 | 时长 |
|------|------|
| VS Code | 2h |

普通段落文本，包含**加粗**内容。`

	err := SaveDocx(content, filePath)
	if err != nil {
		t.Fatalf("SaveDocx failed: %v", err)
	}

	// 验证文件已创建
	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("File not created: %v", err)
	}
	if info.Size() == 0 {
		t.Error("Generated docx file is empty")
	}
}

// ========== 4. 完整日报生成链路测试（Mock LLM） ==========

func TestDailyReportEndToEnd(t *testing.T) {
	// 创建 mock LLM server
	llmContent := "# 日报\n\n## 今日工作\n\n完成了项目开发。"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ChatResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Content string `json:"content"`
				}{Content: llmContent}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// 创建临时数据库
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := storage.Init(dbPath)
	if err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	// 插入模拟活动数据（用精确日期确保落在工作时间范围内）
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, now.Location())
	activities := []*storage.WindowActivity{
		{
			ProcessName: "code.exe",
			WindowTitle: "main.go - Visual Studio Code",
			StartedAt:   today,
			EndedAt:     today.Add(1 * time.Hour),
			DurationSec: 3600,
		},
		{
			ProcessName: "chrome.exe",
			WindowTitle: "GitHub - Google Chrome",
			StartedAt:   today.Add(1 * time.Hour),
			EndedAt:     today.Add(2 * time.Hour),
			DurationSec: 1800,
		},
		{
			ProcessName: "code.exe",
			WindowTitle: "app.go - Visual Studio Code",
			StartedAt:   today.Add(2 * time.Hour),
			EndedAt:     today.Add(3 * time.Hour),
			DurationSec: 1800,
		},
	}
	for _, act := range activities {
		if err := db.InsertWindowActivity(act); err != nil {
			t.Fatalf("Failed to insert activity: %v", err)
		}
	}

	// 创建配置
	cfg := &config.Config{
		WorkTime: config.WorkTimeConfig{
			Start: "09:00",
			End:   "18:00",
		},
		Browser: config.BrowserConfig{Enabled: false},
		LLM: config.LLMConfig{
			Endpoint:    server.URL + "/v1/chat/completions",
			APIKey:      "test-key",
			Model:       "test-model",
			Temperature: 0.7,
			MaxTokens:   4096,
			Timeout:     "10s",
		},
		Report: config.ReportConfig{
			OutputDir:  filepath.Join(tmpDir, "reports"),
			WeeklyDay:  5,
			WeeklyTime: "18:10",
		},
	}

	// 生成日报
	gen := NewGenerator(db, cfg)
	reportPath, err := gen.GenerateDailyReport(now)
	if err != nil {
		t.Fatalf("GenerateDailyReport failed: %v", err)
	}

	// 验证报告文件
	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		t.Fatalf("Report file not created at: %s", reportPath)
	}
	info, _ := os.Stat(reportPath)
	if info.Size() == 0 {
		t.Error("Report file is empty")
	}
	t.Logf("Report generated: %s (%d bytes)", reportPath, info.Size())
}

// ========== 5. 周报生成测试（Mock LLM） ==========

func TestWeeklyReportEndToEnd(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ChatResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Content string `json:"content"`
				}{Content: "# 周报\n\n## 本周总结\n\n完成多项开发任务。"}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := storage.Init(dbPath)
	if err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	// 模拟一周的数据
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	monday := now.AddDate(0, 0, 1-weekday)

	for d := 0; d < 5; d++ {
		day := monday.AddDate(0, 0, d)
		// 用精确日期构造时间，确保落在工作时间内
		dayStart := time.Date(day.Year(), day.Month(), day.Day(), 9, 0, 0, 0, day.Location())
		dayMid := time.Date(day.Year(), day.Month(), day.Day(), 12, 0, 0, 0, day.Location())
		dayAfternoon := time.Date(day.Year(), day.Month(), day.Day(), 14, 0, 0, 0, day.Location())
		dayLate := time.Date(day.Year(), day.Month(), day.Day(), 16, 0, 0, 0, day.Location())
		db.InsertWindowActivity(&storage.WindowActivity{
			ProcessName: "code.exe",
			WindowTitle: "project - VS Code",
			StartedAt:   dayStart,
			EndedAt:     dayMid,
			DurationSec: 10800,
		})
		db.InsertWindowActivity(&storage.WindowActivity{
			ProcessName: "chrome.exe",
			WindowTitle: "Stack Overflow",
			StartedAt:   dayAfternoon,
			EndedAt:     dayLate,
			DurationSec: 7200,
		})
	}

	cfg := &config.Config{
		WorkTime: config.WorkTimeConfig{
			Start: "09:00",
			End:   "18:00",
		},
		Browser: config.BrowserConfig{Enabled: false},
		LLM: config.LLMConfig{
			Endpoint:    server.URL + "/v1/chat/completions",
			APIKey:      "test-key",
			Model:       "test-model",
			Temperature: 0.7,
			MaxTokens:   4096,
			Timeout:     "10s",
		},
		Report: config.ReportConfig{
			OutputDir:  filepath.Join(tmpDir, "reports"),
			WeeklyDay:  5,
			WeeklyTime: "18:10",
		},
	}

	gen := NewGenerator(db, cfg)
	reportPath, err := gen.GenerateWeeklyReport(now)
	if err != nil {
		t.Fatalf("GenerateWeeklyReport failed: %v", err)
	}

	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		t.Fatalf("Weekly report file not created at: %s", reportPath)
	}
	info, _ := os.Stat(reportPath)
	t.Logf("Weekly report generated: %s (%d bytes)", reportPath, info.Size())
}

// ========== 6. 日报无数据报错测试 ==========

func TestDailyReport_NoData(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := storage.Init(dbPath)
	if err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	cfg := &config.Config{
		WorkTime: config.WorkTimeConfig{Start: "09:00", End: "18:00"},
		Browser:  config.BrowserConfig{Enabled: false},
		LLM:      config.LLMConfig{},
		Report:   config.ReportConfig{OutputDir: filepath.Join(tmpDir, "reports")},
	}

	gen := NewGenerator(db, cfg)
	_, err = gen.GenerateDailyReport(time.Now())
	if err == nil {
		t.Fatal("Expected error when no activity data exists")
	}
	if !strings.Contains(err.Error(), "没有找到") {
		t.Errorf("Error should mention no data found, got: %v", err)
	}
}

// ========== 7. Docx 辅助函数测试 ==========

func TestIsTableSeparator(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"|------|------|", true},
		{"|---|---|", true},
		{"| :--- | :---: |", true},
		{"|data|data|", false},
		{"normal text", false},
	}
	for _, tt := range tests {
		result := isTableSeparator(tt.input)
		if result != tt.expected {
			t.Errorf("isTableSeparator('%s') = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestParseTableRow(t *testing.T) {
	result := parseTableRow("| app | time | title |")
	if len(result) != 3 {
		t.Fatalf("Expected 3 cells, got %d", len(result))
	}
	if strings.TrimSpace(result[0]) != "app" {
		t.Errorf("First cell should be 'app', got '%s'", result[0])
	}
}

func TestIsNumberedList(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"1. item one", true},
		{"2. item two", true},
		{"10. item ten", true},
		{"not a list", false},
		{"1.no space", false},
		{"", false},
	}
	for _, tt := range tests {
		result := isNumberedList(tt.input)
		if result != tt.expected {
			t.Errorf("isNumberedList('%s') = %v, want %v", tt.input, result, tt.expected)
		}
	}
}
