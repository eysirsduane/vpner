package task

import (
	"log"
	"time"

	"just-vpn/model"
)

func StartShowRecordFlushTask() {
	go func() {
		ticker := time.NewTicker(model.ShowRecordFlushInterval())
		defer ticker.Stop()
		for range ticker.C {
			if err := model.FlushShowRecordCaches(); err != nil {
				log.Printf("flush show record cache failed: %v", err)
			}
		}
	}()
}
