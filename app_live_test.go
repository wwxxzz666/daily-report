package main

import (
	"testing"

	"daily-report/internal/storage"
)

func TestMergeCurrentActivityIntoSummariesAddsPendingDuration(t *testing.T) {
	summaries := []storage.AppUsageSummary{
		{
			ProcessName: "code",
			TotalSec:    120,
			Titles:      []string{"main.go"},
		},
		{
			ProcessName: "chrome",
			TotalSec:    60,
			Titles:      []string{"Docs"},
		},
	}

	current := &storage.WindowActivity{
		ProcessName: "code",
		WindowTitle: "app.go",
		DurationSec: 30,
	}

	merged := mergeCurrentActivityIntoSummaries(summaries, current)
	if len(merged) != 2 {
		t.Fatalf("expected 2 summaries, got %d", len(merged))
	}
	if merged[0].ProcessName != "code" {
		t.Fatalf("expected code to remain first, got %s", merged[0].ProcessName)
	}
	if merged[0].TotalSec != 150 {
		t.Fatalf("expected merged total 150, got %d", merged[0].TotalSec)
	}
	if !containsTitle(merged[0].Titles, "app.go") {
		t.Fatalf("expected merged titles to include current window title")
	}
}

func TestMergeCurrentActivityIntoSummariesCreatesLiveEntry(t *testing.T) {
	current := &storage.WindowActivity{
		ProcessName: "wechat",
		WindowTitle: "Chat",
		DurationSec: 45,
	}

	merged := mergeCurrentActivityIntoSummaries(nil, current)
	if len(merged) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(merged))
	}
	if merged[0].ProcessName != "wechat" || merged[0].TotalSec != 45 {
		t.Fatalf("unexpected merged summary: %+v", merged[0])
	}
}
