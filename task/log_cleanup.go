package task

import (
	"log"
	"time"

	"just-vpn/model"
)

func StartLogCleanupTask() {
	go func() {
		for {
			nextRun := nextMidnight(time.Now().In(time.Local))
			time.Sleep(time.Until(nextRun))
			runLogCleanup()
		}
	}()
}

func nextMidnight(now time.Time) time.Time {
	return time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
}

func runLogCleanup() {
	results := model.CleanupExpiredLogs(time.Now().In(time.Local))
	for _, result := range results {
		if result.LastErr != nil {
			log.Printf("cleanup log table %s failed: %v", result.Table, result.LastErr)
			continue
		}
		log.Printf("cleanup log table %s done, keep_days=%d, deleted=%d, cutoff=%s",
			result.Table,
			result.KeepDays,
			result.Deleted,
			result.CutoffTime.Format("2006-01-02 15:04:05"),
		)
	}
}
