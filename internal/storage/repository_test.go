package storage

import (
	"path/filepath"
	"testing"
	"time"
)

func TestGetAppUsageSummaryAggregatesRepeatedTitles(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "data.db")
	db, err := Init(dbPath)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer db.Close()

	start := time.Date(2026, 6, 1, 9, 0, 0, 0, time.Local)
	end := start.Add(8 * time.Hour)

	activities := []*WindowActivity{
		{
			ProcessName: "code",
			WindowTitle: "main.go - VS Code",
			StartedAt:   start,
			EndedAt:     start.Add(5 * time.Minute),
			DurationSec: 300,
		},
		{
			ProcessName: "code",
			WindowTitle: "main.go - VS Code",
			StartedAt:   start.Add(5 * time.Minute),
			EndedAt:     start.Add(15 * time.Minute),
			DurationSec: 600,
		},
		{
			ProcessName: "code",
			WindowTitle: "app.go - VS Code",
			StartedAt:   start.Add(15 * time.Minute),
			EndedAt:     start.Add(17 * time.Minute),
			DurationSec: 120,
		},
		{
			ProcessName: "chrome",
			WindowTitle: "Go Docs",
			StartedAt:   start.Add(20 * time.Minute),
			EndedAt:     start.Add(21 * time.Minute),
			DurationSec: 60,
		},
	}

	for _, act := range activities {
		if err := db.InsertWindowActivity(act); err != nil {
			t.Fatalf("InsertWindowActivity failed: %v", err)
		}
	}

	summaries, err := db.GetAppUsageSummary(start, end)
	if err != nil {
		t.Fatalf("GetAppUsageSummary failed: %v", err)
	}

	if len(summaries) != 2 {
		t.Fatalf("expected 2 summaries, got %d", len(summaries))
	}

	if summaries[0].ProcessName != "code" {
		t.Fatalf("expected first summary for code, got %s", summaries[0].ProcessName)
	}

	if summaries[0].TotalSec != 1020 {
		t.Fatalf("expected code total 1020 seconds, got %d", summaries[0].TotalSec)
	}

	if len(summaries[0].Titles) != 2 {
		t.Fatalf("expected 2 unique titles for code, got %d", len(summaries[0].Titles))
	}

	if summaries[1].ProcessName != "chrome" || summaries[1].TotalSec != 60 {
		t.Fatalf("expected chrome total 60 seconds, got %+v", summaries[1])
	}
}
