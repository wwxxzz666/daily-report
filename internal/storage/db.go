package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

var schemaSQL = `
CREATE TABLE IF NOT EXISTS window_activities (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    process_name TEXT NOT NULL,
    window_title TEXT NOT NULL,
    started_at   DATETIME NOT NULL,
    ended_at     DATETIME NOT NULL,
    duration_sec INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_wa_started_at ON window_activities(started_at);
CREATE INDEX IF NOT EXISTS idx_wa_process_name ON window_activities(process_name);
-- 报表按 process_name 聚合 + 时间范围过滤，复合索引可以省掉一次排序
CREATE INDEX IF NOT EXISTS idx_wa_process_started ON window_activities(process_name, started_at);

CREATE TABLE IF NOT EXISTS browser_visits (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    browser         TEXT NOT NULL,
    url             TEXT NOT NULL,
    title           TEXT NOT NULL,
    visited_at      DATETIME NOT NULL,
    source_visit_id INTEGER,
    source_profile  TEXT,
    UNIQUE(browser, source_visit_id, source_profile)
);
CREATE INDEX IF NOT EXISTS idx_bv_visited_at ON browser_visits(visited_at);

CREATE TABLE IF NOT EXISTS generated_reports (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    report_type  TEXT NOT NULL,
    period_start DATE NOT NULL,
    period_end   DATE NOT NULL,
    file_path    TEXT NOT NULL,
    generated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sync_state (
    browser      TEXT PRIMARY KEY,
    profile      TEXT NOT NULL DEFAULT 'Default',
    last_sync_id INTEGER NOT NULL DEFAULT 0
);
`

type DB struct {
	*sql.DB
}

func Init(dbPath string) (*DB, error) {
	// 用 filepath.Dir 而不是按 "/data.db" 的固定长度切字符串：
	// 文件名一旦不是 data.db，旧实现会切出错误目录甚至越界 panic。
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("创建数据库目录失败: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	// SQLite 只支持单写者。database/sql 默认开多个连接，
	// WAL 模式下并发写会撞 SQLITE_BUSY（busy_timeout 只能缓解）。
	// 采样器每 5 秒写一次、Dashboard 同时在读，限制成单连接最稳。
	db.SetMaxOpenConns(1)

	// SQLite 优化设置
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA synchronous=NORMAL",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			db.Close()
			return nil, fmt.Errorf("设置 PRAGMA 失败: %w", err)
		}
	}

	if _, err := db.Exec(schemaSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("初始化表结构失败: %w", err)
	}

	// M4: 限制数据库文件权限为仅所有者可读写
	os.Chmod(dbPath, 0600)

	return &DB{db}, nil
}
