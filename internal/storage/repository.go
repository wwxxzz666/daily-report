package storage

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// WindowActivity 窗口活动记录
type WindowActivity struct {
	ID          int64
	ProcessName string
	WindowTitle string
	StartedAt   time.Time
	EndedAt     time.Time
	DurationSec int
}

// BrowserVisit 浏览器访问记录
type BrowserVisit struct {
	ID            int64
	Browser       string
	URL           string
	Title         string
	VisitedAt     time.Time
	SourceVisitID int
	SourceProfile string
}

// InsertWindowActivity 插入一条窗口活动记录
func (db *DB) InsertWindowActivity(act *WindowActivity) error {
	_, err := db.Exec(`
		INSERT INTO window_activities (process_name, window_title, started_at, ended_at, duration_sec)
		VALUES (?, ?, ?, ?, ?)`,
		act.ProcessName, act.WindowTitle, act.StartedAt, act.EndedAt, act.DurationSec,
	)
	return err
}

// UpdateWindowActivityEnd 更新活动记录的结束时间
func (db *DB) UpdateWindowActivityEnd(id int64, endedAt time.Time, durationSec int) error {
	_, err := db.Exec(`
		UPDATE window_activities SET ended_at = ?, duration_sec = ? WHERE id = ?`,
		endedAt, durationSec, id,
	)
	return err
}

// GetLastWindowActivity 获取最后一条窗口活动记录
func (db *DB) GetLastWindowActivity() (*WindowActivity, error) {
	row := db.QueryRow(`SELECT id, process_name, window_title, started_at, ended_at, duration_sec
		FROM window_activities ORDER BY id DESC LIMIT 1`)

	var act WindowActivity
	err := row.Scan(&act.ID, &act.ProcessName, &act.WindowTitle, &act.StartedAt, &act.EndedAt, &act.DurationSec)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &act, nil
}

// GetWindowActivities 查询指定时间段的窗口活动
func (db *DB) GetWindowActivities(start, end time.Time) ([]WindowActivity, error) {
	rows, err := db.Query(`
		SELECT id, process_name, window_title, started_at, ended_at, duration_sec
		FROM window_activities
		WHERE started_at >= ? AND started_at < ?
		ORDER BY started_at ASC`,
		start, end,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activities []WindowActivity
	for rows.Next() {
		var act WindowActivity
		if err := rows.Scan(&act.ID, &act.ProcessName, &act.WindowTitle, &act.StartedAt, &act.EndedAt, &act.DurationSec); err != nil {
			return nil, err
		}
		activities = append(activities, act)
	}
	return activities, rows.Err()
}

// AppUsageSummary 应用使用汇总
type AppUsageSummary struct {
	ProcessName string
	TotalSec    int
	Titles      []string
}

// GetAppUsageSummary 获取指定时间段的应用使用汇总
func (db *DB) GetAppUsageSummary(start, end time.Time) ([]AppUsageSummary, error) {
	rows, err := db.Query(`
		SELECT process_name,
			   SUM(duration_sec) as total_sec,
			   GROUP_CONCAT(window_title, '|') as titles
		FROM (
			SELECT process_name, window_title, SUM(duration_sec) as duration_sec
			FROM window_activities
			WHERE started_at >= ? AND started_at < ?
			GROUP BY process_name, window_title
		)
		GROUP BY process_name
		ORDER BY total_sec DESC`,
		start, end,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []AppUsageSummary
	for rows.Next() {
		var s AppUsageSummary
		var titlesStr string
		if err := rows.Scan(&s.ProcessName, &s.TotalSec, &titlesStr); err != nil {
			return nil, err
		}
		s.Titles = strings.Split(titlesStr, "|")
		summaries = append(summaries, s)
	}
	return summaries, rows.Err()
}

// InsertBrowserVisits 批量插入浏览器访问记录
func (db *DB) InsertBrowserVisits(visits []BrowserVisit) (int, error) {
	if len(visits) == 0 {
		return 0, nil
	}

	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR IGNORE INTO browser_visits (browser, url, title, visited_at, source_visit_id, source_profile)
		VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	inserted := 0
	for _, v := range visits {
		result, err := stmt.Exec(v.Browser, v.URL, v.Title, v.VisitedAt, v.SourceVisitID, v.SourceProfile)
		if err != nil {
			continue
		}
		if rows, _ := result.RowsAffected(); rows > 0 {
			inserted++
		}
	}

	return inserted, tx.Commit()
}

// GetBrowserVisits 查询指定时间段的浏览器访问记录
func (db *DB) GetBrowserVisits(start, end time.Time) ([]BrowserVisit, error) {
	rows, err := db.Query(`
		SELECT id, browser, url, title, visited_at, COALESCE(source_visit_id, 0), COALESCE(source_profile, '')
		FROM browser_visits
		WHERE visited_at >= ? AND visited_at < ?
		ORDER BY visited_at ASC`,
		start, end,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var visits []BrowserVisit
	for rows.Next() {
		var v BrowserVisit
		if err := rows.Scan(&v.ID, &v.Browser, &v.URL, &v.Title, &v.VisitedAt, &v.SourceVisitID, &v.SourceProfile); err != nil {
			return nil, err
		}
		visits = append(visits, v)
	}
	return visits, rows.Err()
}

// GetSyncState 获取浏览器同步状态
func (db *DB) GetSyncState(browser string) (int, error) {
	var lastID int
	err := db.QueryRow(`SELECT last_sync_id FROM sync_state WHERE browser = ?`, browser).Scan(&lastID)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return lastID, err
}

// UpdateSyncState 更新浏览器同步状态
func (db *DB) UpdateSyncState(browser string, lastID int) error {
	_, err := db.Exec(`
		INSERT INTO sync_state (browser, last_sync_id) VALUES (?, ?)
		ON CONFLICT(browser) DO UPDATE SET last_sync_id = ?`,
		browser, lastID, lastID,
	)
	return err
}

// InsertReport 记录已生成的报告
func (db *DB) InsertReport(reportType, periodStart, periodEnd, filePath string) error {
	_, err := db.Exec(`
		INSERT INTO generated_reports (report_type, period_start, period_end, file_path)
		VALUES (?, ?, ?, ?)`,
		reportType, periodStart, periodEnd, filePath,
	)
	return err
}

// CleanupOldActivities 清理指定天数前的活动记录
func (db *DB) CleanupOldActivities(days int) error {
	cutoff := time.Now().AddDate(0, 0, -days)
	_, err := db.Exec(`DELETE FROM window_activities WHERE ended_at < ?`, cutoff)
	if err != nil {
		return fmt.Errorf("清理窗口活动记录失败: %w", err)
	}
	_, err = db.Exec(`DELETE FROM browser_visits WHERE visited_at < ?`, cutoff)
	if err != nil {
		return fmt.Errorf("清理浏览器访问记录失败: %w", err)
	}
	return nil
}
