package browser

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"daily-report/internal/privacy"
	"daily-report/internal/storage"

	_ "modernc.org/sqlite"
)

// WebKit epoch: 1601-01-01 00:00:00 UTC
var webKitEpoch = time.Date(1601, 1, 1, 0, 0, 0, 0, time.UTC)

// webKitToTime 将 Chrome/Edge 的 WebKit 时间戳转为 time.Time
// WebKit 时间戳是从 1601-01-01 起的微秒数
func webKitToTime(webkitTimestamp int64) time.Time {
	// 微秒 -> 秒 + 剩余微秒
	sec := webkitTimestamp / 1_000_000
	usec := webkitTimestamp % 1_000_000
	// 用 time.Unix 避免 Duration 溢出（Duration 最大 ~290 年）
	// Unix epoch (1970) 比 WebKit epoch (1601) 晚 11644473600 秒
	const unixToWebKit int64 = 11644473600
	return time.Unix(sec-unixToWebKit, usec*1000)
}

// SyncHistory 同步指定浏览器的历史记录
func SyncHistory(db *storage.DB, browser string) error {
	paths, err := BrowserPath(browser)
	if err != nil {
		return fmt.Errorf("获取 %s 路径失败: %w", browser, err)
	}

	for _, historyPath := range paths {
		profileName := filepath.Base(filepath.Dir(historyPath))
		if err := syncHistoryFile(db, browser, historyPath, profileName); err != nil {
			log.Printf("同步 %s (%s) 历史失败: %v", browser, profileName, err)
		}
	}

	return nil
}

func syncHistoryFile(db *storage.DB, browser, historyPath, profile string) error {
	// 浏览器运行时会锁住 History 文件，需要先拷贝
	// M2: 使用安全临时文件（随机名 + 0600 权限）
	tmpFile, err := os.CreateTemp("", fmt.Sprintf("dailyreport_%s_%s_*.db", browser, profile))
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}
	tmpPath := tmpFile.Name()
	// 立即关闭，后面用 WriteFile 写入
	tmpFile.Close()
	// 确保无论成功失败都清理临时文件
	defer os.Remove(tmpPath)

	srcData, err := os.ReadFile(historyPath)
	if err != nil {
		return fmt.Errorf("读取历史文件失败: %w", err)
	}
	if err := os.WriteFile(tmpPath, srcData, 0600); err != nil {
		return fmt.Errorf("拷贝历史文件失败: %w", err)
	}

	// 打开拷贝的数据库
	conn, err := sql.Open("sqlite", tmpPath+"?mode=ro")
	if err != nil {
		return fmt.Errorf("打开历史数据库失败: %w", err)
	}
	defer conn.Close()

	// 获取上次同步位置
	lastID, err := db.GetSyncState(browser + "_" + profile)
	if err != nil {
		return fmt.Errorf("获取同步状态失败: %w", err)
	}

	// 查询新记录
	rows, err := conn.Query(`
		SELECT v.id, u.url, u.title, v.visit_time
		FROM visits v
		JOIN urls u ON v.url = u.id
		WHERE v.id > ?
		ORDER BY v.id ASC
		LIMIT 5000`, lastID)
	if err != nil {
		return fmt.Errorf("查询历史记录失败: %w", err)
	}
	defer rows.Close()

	var visits []storage.BrowserVisit
	maxID := lastID
	for rows.Next() {
		var visitID int
		var url, title string
		var visitTime int64

		if err := rows.Scan(&visitID, &url, &title, &visitTime); err != nil {
			continue
		}

		visitedAt := webKitToTime(visitTime).Local()

		// 隐私保护：清除 URL 中的查询参数和 fragment
		sanitizedURL := privacy.SanitizeURL(url)

		visits = append(visits, storage.BrowserVisit{
			Browser:       browser,
			URL:           sanitizedURL,
			Title:         title,
			VisitedAt:     visitedAt,
			SourceVisitID: visitID,
			SourceProfile: profile,
		})

		if visitID > maxID {
			maxID = visitID
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("遍历历史记录失败: %w", err)
	}

	// 写入数据库
	inserted, err := db.InsertBrowserVisits(visits)
	if err != nil {
		return fmt.Errorf("写入浏览器访问记录失败: %w", err)
	}

	// 更新同步状态
	if maxID > lastID {
		if err := db.UpdateSyncState(browser+"_"+profile, maxID); err != nil {
			return fmt.Errorf("更新同步状态失败: %w", err)
		}
	}

	log.Printf("同步 %s (%s): %d 条记录, %d 条新增", browser, profile, len(visits), inserted)
	return nil
}

// SyncAll 同步所有配置的浏览器
func SyncAll(db *storage.DB, browsers []string) {
	for _, b := range browsers {
		if err := SyncHistory(db, b); err != nil {
			log.Printf("同步浏览器 %s 失败: %v", b, err)
		}
	}
}
