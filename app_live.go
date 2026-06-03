package main

import (
	"sort"

	"daily-report/internal/storage"
)

func (a *App) currentActivity() *storage.WindowActivity {
	if a == nil || a.sampler == nil {
		return nil
	}
	return a.sampler.CurrentActivity()
}

func mergeCurrentActivityIntoSummaries(summaries []storage.AppUsageSummary, current *storage.WindowActivity) []storage.AppUsageSummary {
	if current == nil || current.DurationSec < 1 {
		return summaries
	}

	merged := make([]storage.AppUsageSummary, len(summaries))
	copy(merged, summaries)

	for i := range merged {
		if merged[i].ProcessName != current.ProcessName {
			continue
		}
		merged[i].TotalSec += current.DurationSec
		if current.WindowTitle != "" && !containsTitle(merged[i].Titles, current.WindowTitle) {
			merged[i].Titles = append(merged[i].Titles, current.WindowTitle)
		}
		sortUsageSummaries(merged)
		return merged
	}

	merged = append(merged, storage.AppUsageSummary{
		ProcessName: current.ProcessName,
		TotalSec:    current.DurationSec,
		Titles:      []string{current.WindowTitle},
	})
	sortUsageSummaries(merged)
	return merged
}

func sortUsageSummaries(summaries []storage.AppUsageSummary) {
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].TotalSec > summaries[j].TotalSec
	})
}

func containsTitle(titles []string, title string) bool {
	for _, existing := range titles {
		if existing == title {
			return true
		}
	}
	return false
}
