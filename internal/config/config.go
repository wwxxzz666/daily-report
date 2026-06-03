package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	WorkTime        WorkTimeConfig `yaml:"work_time"`
	SampleInterval  string         `yaml:"sample_interval"`
	MonitoredApps   []string       `yaml:"monitored_apps"`
	IgnoredApps     []string       `yaml:"ignored_apps"`
	SensitiveWords  []string       `yaml:"sensitive_words"`
	Browser         BrowserConfig  `yaml:"browser"`
	LLM             LLMConfig      `yaml:"llm"`
	Report          ReportConfig   `yaml:"report"`
}

type WorkTimeConfig struct {
	Start    string `yaml:"start"`
	End      string `yaml:"end"`
	Weekdays []int  `yaml:"weekdays"`
}

type BrowserConfig struct {
	Enabled      bool     `yaml:"enabled"`
	Browsers     []string `yaml:"browsers"`
	HistoryHours int      `yaml:"history_hours"`
}

type LLMConfig struct {
	Provider     string  `yaml:"provider"`      // 预设提供商名称
	Endpoint     string  `yaml:"endpoint"`      // 自定义端点（provider 为空或 "custom" 时使用）
	Model        string  `yaml:"model"`         // 模型名称（空则使用预设默认值）
	APIKey       string  `yaml:"api_key"`
	Temperature  float64 `yaml:"temperature"`
	MaxTokens    int     `yaml:"max_tokens"`
	Timeout      string  `yaml:"timeout"`
}

// ProviderPreset 预设提供商配置
type ProviderPreset struct {
	Name        string
	Endpoint    string
	DefaultModel string
	NeedsAPIKey bool
	Description string
}

// SupportedProviders 返回所有支持的预设提供商
func SupportedProviders() []ProviderPreset {
	return []ProviderPreset{
		{
			Name:         "deepseek",
			Endpoint:     "https://api.deepseek.com/v1/chat/completions",
			DefaultModel: "deepseek-chat",
			NeedsAPIKey:  true,
			Description:  "DeepSeek（性价比高，推荐国内用户）",
		},
		{
			Name:         "openai",
			Endpoint:     "https://api.openai.com/v1/chat/completions",
			DefaultModel: "gpt-4o-mini",
			NeedsAPIKey:  true,
			Description:  "OpenAI（GPT-4o-mini）",
		},
		{
			Name:         "qwen",
			Endpoint:     "https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions",
			DefaultModel: "qwen-plus",
			NeedsAPIKey:  true,
			Description:  "通义千问（阿里云）",
		},
		{
			Name:         "moonshot",
			Endpoint:     "https://api.moonshot.cn/v1/chat/completions",
			DefaultModel: "moonshot-v1-8k",
			NeedsAPIKey:  true,
			Description:  "Moonshot（月之暗面 Kimi）",
		},
		{
			Name:         "zhipu",
			Endpoint:     "https://open.bigmodel.cn/api/paas/v4/chat/completions",
			DefaultModel: "glm-4-flash",
			NeedsAPIKey:  true,
			Description:  "智谱清言（GLM-4）",
		},
		{
			Name:         "ollama",
			Endpoint:     "https://localhost:11434/v1/chat/completions",
			DefaultModel: "qwen2.5:7b",
			NeedsAPIKey:  false,
			Description:  "Ollama 本地模型（数据不出本机，隐私最佳）",
		},
		{
			Name:         "lmstudio",
			Endpoint:     "https://localhost:1234/v1/chat/completions",
			DefaultModel: "default",
			NeedsAPIKey:  false,
			Description:  "LM Studio 本地模型",
		},
	}
}

// GetProviderPreset 根据名称获取预设配置
func GetProviderPreset(name string) *ProviderPreset {
	for _, p := range SupportedProviders() {
		if p.Name == name {
			return &p
		}
	}
	return nil
}

// GetEndpoint 获取最终使用的 API 端点
// 优先使用预设提供商的端点，否则使用自定义 endpoint
func (l *LLMConfig) GetEndpoint() string {
	if l.Provider != "" && l.Provider != "custom" {
		if preset := GetProviderPreset(l.Provider); preset != nil {
			return preset.Endpoint
		}
	}
	return l.Endpoint
}

// GetModel 获取最终使用的模型名称
func (l *LLMConfig) GetModel() string {
	if l.Provider != "" && l.Provider != "custom" {
		if l.Model != "" {
			return l.Model
		}
		if preset := GetProviderPreset(l.Provider); preset != nil {
			return preset.DefaultModel
		}
	}
	return l.Model
}

type ReportConfig struct {
	OutputDir  string `yaml:"output_dir"`
	WeeklyDay  int    `yaml:"weekly_day"`
	WeeklyTime string `yaml:"weekly_time"`
}

var (
	instance *Config
	mu       sync.RWMutex
)

func appDataDir() string {
	appData := os.Getenv("APPDATA")
	return filepath.Join(appData, "日报助手")
}

func defaultConfig() *Config {
	return &Config{
		WorkTime: WorkTimeConfig{
			Start:    "09:00",
			End:      "18:00",
			Weekdays: []int{1, 2, 3, 4, 5},
		},
		SampleInterval: "5s",
		MonitoredApps:  []string{"*"},
		IgnoredApps: []string{
			"explorer",
			"SearchHost",
			"ShellExperienceHost",
		},
		SensitiveWords: []string{
			"密码",
			"password",
			"薪资",
			"salary",
			"身份证",
			"银行卡",
			"token",
			"secret",
		},
		Browser: BrowserConfig{
			Enabled:      true,
			Browsers:     []string{"chrome", "edge"},
			HistoryHours: 8,
		},
		LLM: LLMConfig{
			Provider:     "deepseek",
			Endpoint:     "",
			Model:        "",
			APIKey:       "sk-xxx",
			Temperature:  0.7,
			MaxTokens:    4096,
			Timeout:      "60s",
		},
		Report: ReportConfig{
			OutputDir:  filepath.Join(appDataDir(), "reports"),
			WeeklyDay:  5,
			WeeklyTime: "18:10",
		},
	}
}

func Load() (*Config, error) {
	dir := appDataDir()
	configPath := filepath.Join(dir, "config.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := defaultConfig()
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, fmt.Errorf("创建配置目录失败: %w", err)
			}
			if err := Save(cfg); err != nil {
				return nil, fmt.Errorf("保存默认配置失败: %w", err)
			}
			SetInstance(cfg)
			return cfg, nil
		}
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	cfg := defaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	SetInstance(cfg)
	return cfg, nil
}

func Save(cfg *Config) error {
	dir := appDataDir()
	configPath := filepath.Join(dir, "config.yaml")

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	return os.WriteFile(configPath, data, 0600)
}

func Get() *Config {
	mu.RLock()
	defer mu.RUnlock()
	return instance
}

func SetInstance(cfg *Config) {
	mu.Lock()
	defer mu.Unlock()
	instance = cfg
}

func Reload() (*Config, error) {
	return Load()
}

func (c *Config) GetSampleInterval() time.Duration {
	d, err := time.ParseDuration(c.SampleInterval)
	if err != nil {
		return 5 * time.Second
	}
	return d
}

func (c *Config) GetWorkStart() (int, int, error) {
	return parseTime(c.WorkTime.Start)
}

func (c *Config) GetWorkEnd() (int, int, error) {
	return parseTime(c.WorkTime.End)
}

func (c *Config) GetLLMTimeout() time.Duration {
	d, err := time.ParseDuration(c.LLM.Timeout)
	if err != nil {
		return 60 * time.Second
	}
	return d
}

func (c *Config) IsWorkday(weekday time.Weekday) bool {
	// time.Weekday: Sunday=0, Monday=1, ..., Saturday=6
	// config: Monday=1, ..., Sunday=7
	day := int(weekday)
	if day == 0 {
		day = 7
	}
	for _, d := range c.WorkTime.Weekdays {
		if d == day {
			return true
		}
	}
	return false
}

func (c *Config) IsWorkingTime(now time.Time) bool {
	if !c.IsWorkday(now.Weekday()) {
		return false
	}

	startH, startM, err := c.GetWorkStart()
	if err != nil {
		return false
	}
	endH, endM, err := c.GetWorkEnd()
	if err != nil {
		return false
	}

	nowMinutes := now.Hour()*60 + now.Minute()
	startMinutes := startH*60 + startM
	endMinutes := endH*60 + endM

	return nowMinutes >= startMinutes && nowMinutes <= endMinutes
}

func (c *Config) ConfigPath() string {
	return filepath.Join(appDataDir(), "config.yaml")
}

func (c *Config) DBPath() string {
	return filepath.Join(appDataDir(), "data.db")
}

func (c *Config) LogPath() string {
	return filepath.Join(appDataDir(), "app.log")
}

func (c *Config) GetReportOutputDir() string {
	dir := c.Report.OutputDir
	// 展开 %APPDATA%
	if len(dir) > 10 && dir[:10] == "%APPDATA%" {
		dir = filepath.Join(os.Getenv("APPDATA"), dir[11:])
	}
	return dir
}

func parseTime(s string) (int, int, error) {
	var h, m int
	_, err := fmt.Sscanf(s, "%d:%d", &h, &m)
	if err != nil {
		return 0, 0, fmt.Errorf("无效的时间格式: %s", s)
	}
	return h, m, nil
}
