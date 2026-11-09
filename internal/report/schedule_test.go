package report

import (
	"path/filepath"
	"testing"
	"time"

	"daily-report/internal/storage"
)

func newTestDB(t *testing.T) *storage.DB {
	t.Helper()
	db, err := storage.Init(filepath.Join(t.TempDir(), "data.db"))
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestWeekMonday(t *testing.T) {
	tests := []struct {
		name string
		in   time.Time
		want time.Time
	}{
		{"周一", time.Date(2026, 6, 1, 10, 0, 0, 0, time.Local), time.Date(2026, 6, 1, 10, 0, 0, 0, time.Local)},
		{"周三", time.Date(2026, 6, 3, 10, 0, 0, 0, time.Local), time.Date(2026, 6, 1, 10, 0, 0, 0, time.Local)},
		{"周五", time.Date(2026, 6, 5, 10, 0, 0, 0, time.Local), time.Date(2026, 6, 1, 10, 0, 0, 0, time.Local)},
		{"周六", time.Date(2026, 6, 6, 10, 0, 0, 0, time.Local), time.Date(2026, 6, 1, 10, 0, 0, 0, time.Local)},
		// 2026-06-07 是周日，应归到 06-01 那一周
		{"周日", time.Date(2026, 6, 7, 10, 0, 0, 0, time.Local), time.Date(2026, 6, 1, 10, 0, 0, 0, time.Local)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WeekMonday(tt.in)
			if !got.Equal(tt.want) {
				t.Errorf("WeekMonday(%s) = %s, want %s",
					tt.in.Format("2006-01-02 Mon"), got.Format("2006-01-02 Mon"), tt.want.Format("2006-01-02 Mon"))
			}
		})
	}
}

func TestShouldAutoGenerate_Daily(t *testing.T) {
	db := newTestDB(t)
	day := time.Date(2026, 6, 3, 18, 0, 0, 0, time.Local)

	if !ShouldAutoGenerate(db, "daily", day) {
		t.Fatal("空库应允许生成")
	}

	if err := db.InsertReport("daily", "2026-06-03", "2026-06-03", "/tmp/a.docx"); err != nil {
		t.Fatal(err)
	}

	if ShouldAutoGenerate(db, "daily", day) {
		t.Fatal("当天已生成过，自动任务应跳过（重启后不能重复生成）")
	}

	// 换一天仍然允许
	if !ShouldAutoGenerate(db, "daily", day.AddDate(0, 0, 1)) {
		t.Fatal("第二天应允许生成")
	}
}

func TestShouldAutoGenerate_WeeklyUsesMonday(t *testing.T) {
	db := newTestDB(t)

	// 周报记录的 period_start 是周一
	if err := db.InsertReport("weekly", "2026-06-01", "2026-06-05", "/tmp/w.docx"); err != nil {
		t.Fatal(err)
	}

	// 周五触发时，用 WeekMonday 归一化到周一，才能命中已存在的记录
	friday := time.Date(2026, 6, 5, 18, 10, 0, 0, time.Local)
	if ShouldAutoGenerate(db, "weekly", WeekMonday(friday)) {
		t.Fatal("本周周报已生成，应跳过")
	}

	// 下一周应允许
	nextFriday := friday.AddDate(0, 0, 7)
	if !ShouldAutoGenerate(db, "weekly", WeekMonday(nextFriday)) {
		t.Fatal("下一周应允许生成")
	}
}

func TestShouldAutoGenerate_TypeIsolation(t *testing.T) {
	db := newTestDB(t)
	day := time.Date(2026, 6, 3, 18, 0, 0, 0, time.Local)

	if err := db.InsertReport("weekly", "2026-06-03", "2026-06-03", "/tmp/w.docx"); err != nil {
		t.Fatal(err)
	}

	if !ShouldAutoGenerate(db, "daily", day) {
		t.Fatal("weekly 记录不应阻止 daily 生成")
	}
}
