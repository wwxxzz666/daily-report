package monitor

import (
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"daily-report/internal/config"
	"daily-report/internal/privacy"
	"daily-report/internal/storage"
)

type Sampler struct {
	db     *storage.DB
	stopCh chan struct{}
	wg     sync.WaitGroup
	mu     sync.RWMutex

	// 当前活动的缓存，用于合并连续相同窗口
	lastActivity *storage.WindowActivity
}

func NewSampler(db *storage.DB) *Sampler {
	return &Sampler{
		db:     db,
		stopCh: make(chan struct{}),
	}
}

// getConfig 每次获取最新的配置，支持热重载
func (s *Sampler) getConfig() *config.Config {
	return config.Get()
}

func (s *Sampler) Start() {
	s.wg.Add(1)
	defer s.wg.Done()

	cfg := s.getConfig()
	interval := cfg.GetSampleInterval()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// 每 30 秒强制刷写一次缓存，确保 Dashboard 能看到实时数据
	flushTicker := time.NewTicker(30 * time.Second)
	defer flushTicker.Stop()

	log.Printf("采样器已启动，间隔: %v", interval)

	for {
		select {
		case <-s.stopCh:
			// 停止前刷写最后一条记录
			s.flush()
			log.Println("采样器已停止")
			return
		case <-ticker.C:
			s.sample()
		case <-flushTicker.C:
			s.flush()
		}
	}
}

func (s *Sampler) Stop() {
	close(s.stopCh)
	s.wg.Wait()
}

func (s *Sampler) CurrentActivity() *storage.WindowActivity {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.lastActivity == nil {
		return nil
	}

	current := *s.lastActivity
	now := time.Now()
	duration := int(now.Sub(current.StartedAt).Seconds())
	if duration < current.DurationSec {
		duration = current.DurationSec
	}
	if duration < 0 {
		duration = 0
	}

	current.EndedAt = now
	current.DurationSec = duration
	return &current
}

func (s *Sampler) sample() {
	cfg := s.getConfig()

	// 检查是否在工作时间内
	now := time.Now()
	if !cfg.IsWorkingTime(now) {
		// 不在工作时间，刷写并清空缓存
		s.flushAndClear()
		return
	}

	win, err := GetActiveWindow()
	if err != nil {
		return
	}

	// 检查是否应该监控此应用
	if !shouldMonitor(cfg, win.ProcessName) {
		s.flushAndClear()
		return
	}

	// 隐私保护：跳过密码管理器等敏感进程
	if privacy.ShouldIgnoreProcess(win.ProcessName) {
		s.flushAndClear()
		return
	}

	// 隐私保护：敏感词过滤，标题包含敏感词则跳过
	if privacy.IsSensitiveTitle(win.WindowTitle, cfg.SensitiveWords) {
		s.flushAndClear()
		return
	}

	// 隐私保护：标题脱敏
	win.WindowTitle = privacy.SanitizeTitle(win.WindowTitle)

	s.mu.Lock()
	defer s.mu.Unlock()

	// 如果和上次相同窗口，合并（更新结束时间）
	if s.lastActivity != nil &&
		s.lastActivity.ProcessName == win.ProcessName &&
		s.lastActivity.WindowTitle == win.WindowTitle {
		duration := int(now.Sub(s.lastActivity.StartedAt).Seconds())
		s.lastActivity.EndedAt = now
		s.lastActivity.DurationSec = duration
		return
	}

	// 新窗口，先刷写旧的
	s.flushAndClearLocked()

	// 创建新记录
	s.lastActivity = &storage.WindowActivity{
		ProcessName: win.ProcessName,
		WindowTitle: win.WindowTitle,
		StartedAt:   now,
		EndedAt:     now,
		DurationSec: 0,
	}
}

func (s *Sampler) flush() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.flushLocked()
}

func (s *Sampler) flushLocked() {
	if s.lastActivity == nil {
		return
	}

	// 如果持续时间太短（<1秒），忽略但不清缓存
	if s.lastActivity.DurationSec < 1 {
		s.lastActivity.DurationSec = int(time.Since(s.lastActivity.StartedAt).Seconds())
	}

	if s.lastActivity.DurationSec < 1 {
		s.lastActivity = nil
		return
	}

	s.lastActivity.EndedAt = s.lastActivity.StartedAt.Add(time.Duration(s.lastActivity.DurationSec) * time.Second)

	if err := s.db.InsertWindowActivity(s.lastActivity); err != nil {
		log.Printf("写入窗口活动记录失败: %v", err)
	}

	// 写入后保留缓存引用，重置起始时间以便继续合并同一窗口
	now := time.Now()
	s.lastActivity.StartedAt = now
	s.lastActivity.EndedAt = now
	s.lastActivity.DurationSec = 0
}

// flushAndClear 写入当前缓存并清空，用于活动中断场景（窗口切换、非工作时间等）
func (s *Sampler) flushAndClear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.flushAndClearLocked()
}

func (s *Sampler) flushAndClearLocked() {
	s.flushLocked()
	s.lastActivity = nil
}

func shouldMonitor(cfg *config.Config, processName string) bool {
	// 提取不带扩展名的进程名
	name := strings.ToLower(strings.TrimSuffix(processName, filepath.Ext(processName)))

	// 检查忽略列表（优先级更高）
	for _, ignored := range cfg.IgnoredApps {
		if strings.EqualFold(name, strings.ToLower(ignored)) {
			return false
		}
	}

	// 检查监控列表
	for _, app := range cfg.MonitoredApps {
		if app == "*" {
			return true
		}
		if strings.EqualFold(name, strings.ToLower(app)) {
			return true
		}
	}

	return false
}
