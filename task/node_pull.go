package task

import (
	"log"
	"time"

	"just-vpn/model"
)

const nodePullInterval = 5 * time.Minute

// StartNodePullTask 启动节点订阅同步任务，启动后立即执行，之后每五分钟执行一次
func StartNodePullTask() {
	go func() {
		runNodePull()
		ticker := time.NewTicker(nodePullInterval)
		defer ticker.Stop()
		for range ticker.C {
			runNodePull()
		}
	}()
}

func runNodePull() {
	result, err := model.SyncNodesFromConfiguredSource()
	if err != nil {
		log.Printf("sync node subscription failed: %v", err)
		return
	}
	if result.Skipped {
		return
	}
	log.Printf("sync node subscription done: pulled=%d added=%d activated=%d retained=%t", result.Pulled, result.Added, result.Activated, result.Retained)
}
