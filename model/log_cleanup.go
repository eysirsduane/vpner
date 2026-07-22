package model

import (
	"fmt"
	"time"
)

const logCleanupBatchSize = 5000

// LogCleanupResult 日志清理结果
type LogCleanupResult struct {
	Table      string
	KeepDays   int
	Deleted    int64
	CutoffTime time.Time
	LastErr    error
}

type logCleanupTable struct {
	Table    string
	KeepDays int
	Model    interface{}
}

var logCleanupTables = []logCleanupTable{
	{Table: "turn_pay_exposure_log", KeepDays: 7, Model: &TurnPayExposureLog{}},
	{Table: "node_connect_history", KeepDays: 7, Model: &NodeConnectHistory{}},
	{Table: "error", KeepDays: 7, Model: &Error{}},
	{Table: "apple_notification", KeepDays: 30, Model: &AppleNotification{}},
	{Table: "third_pay_callback", KeepDays: 30, Model: &ThirdPayCallback{}},
}

func EnsureLogCleanupIndexes() error {
	for _, table := range logCleanupTables {
		if DB.Migrator().HasIndex(table.Model, "idx_create_time") {
			continue
		}
		if err := DB.Exec(fmt.Sprintf("CREATE INDEX idx_create_time ON `%s` (`create_time`)", table.Table)).Error; err != nil {
			return err
		}
	}
	return nil
}

func CleanupExpiredLogs(now time.Time) []LogCleanupResult {
	results := make([]LogCleanupResult, 0, len(logCleanupTables))
	for _, table := range logCleanupTables {
		cutoff := now.AddDate(0, 0, -table.KeepDays)
		deleted, err := cleanupExpiredLogTable(table.Table, cutoff, logCleanupBatchSize)
		results = append(results, LogCleanupResult{
			Table:      table.Table,
			KeepDays:   table.KeepDays,
			Deleted:    deleted,
			CutoffTime: cutoff,
			LastErr:    err,
		})
	}
	return results
}

func cleanupExpiredLogTable(table string, cutoff time.Time, batchSize int) (int64, error) {
	var total int64
	for {
		result := DB.Exec(fmt.Sprintf("DELETE FROM `%s` WHERE `create_time` < ? ORDER BY `create_time` ASC, `id` ASC LIMIT ?", table), cutoff, batchSize)
		if result.Error != nil {
			return total, result.Error
		}
		total += result.RowsAffected
		if result.RowsAffected < int64(batchSize) {
			return total, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
}
