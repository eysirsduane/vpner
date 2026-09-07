package task

import (
	"log"
	"time"

	"just-vpn/model"
)

func StartDailyStatTask() {
	go func() {
		time.Sleep(10 * time.Second)
		runDailyStat()

		for {
			now := time.Now().In(time.Local)
			nextRun := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 5, 0, 0, now.Location())
			time.Sleep(time.Until(nextRun))
			runDailyStat()
		}
	}()
}

func runDailyStat() {
	statDate := time.Now().In(time.Local).AddDate(0, 0, -1)
	if err := model.SyncDailyStatForDate(statDate); err != nil {
		log.Printf("sync daily stat failed: date=%s err=%v", statDate.Format("2006-01-02"), err)
		return
	}
	if err := model.SyncDailyRetentionByActiveDate(statDate); err != nil {
		log.Printf("sync daily retention failed: active_date=%s err=%v", statDate.Format("2006-01-02"), err)
		return
	}
	log.Printf("sync daily stat done: date=%s", statDate.Format("2006-01-02"))
}
