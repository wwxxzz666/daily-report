package scheduler

import (
	"fmt"
	"log"
	"sync"
	"time"

	"daily-report/internal/config"
)

type Scheduler struct {
	onDailyReport  func()
	onWeeklyReport func()
	stopCh         chan struct{}
	wg             sync.WaitGroup
}

func New() *Scheduler {
	return &Scheduler{
		stopCh: make(chan struct{}),
	}
}

func (s *Scheduler) OnDailyReport(fn func()) {
	s.onDailyReport = fn
}

func (s *Scheduler) OnWeeklyReport(fn func()) {
	s.onWeeklyReport = fn
}

func (s *Scheduler) Start() {
	s.wg.Add(1)
	defer s.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	log.Println("定时调度器已启动")

	// 记录今天是否已触发日报/周报
	var lastDailyDate string
	var lastWeeklyDate string

	for {
		select {
		case <-s.stopCh:
			log.Println("定时调度器已停止")
			return
		case now := <-ticker.C:
			cfg := config.Get()
			if cfg == nil {
				continue
			}

			today := now.Format("2006-01-02")

			// 检查日报触发
			if s.onDailyReport != nil && today != lastDailyDate {
				if shouldTriggerDaily(cfg, now) {
					log.Println("定时触发日报生成")
					lastDailyDate = today
					go s.onDailyReport()
				}
			}

			// 检查周报触发
			if s.onWeeklyReport != nil && today != lastWeeklyDate {
				if shouldTriggerWeekly(cfg, now) {
					log.Println("定时触发周报生成")
					lastWeeklyDate = today
					go s.onWeeklyReport()
				}
			}
		}
	}
}

func (s *Scheduler) Stop() {
	close(s.stopCh)
	s.wg.Wait()
}

func shouldTriggerDaily(cfg *config.Config, now time.Time) bool {
	endH, endM, err := cfg.GetWorkEnd()
	if err != nil {
		return false
	}
	return now.Hour() == endH && now.Minute() == endM && cfg.IsWorkday(now.Weekday())
}

func shouldTriggerWeekly(cfg *config.Config, now time.Time) bool {
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	if weekday != cfg.Report.WeeklyDay {
		return false
	}

	var wH, wM int
	_, err := fmt.Sscanf(cfg.Report.WeeklyTime, "%d:%d", &wH, &wM)
	if err != nil {
		return false
	}

	return now.Hour() == wH && now.Minute() == wM
}
