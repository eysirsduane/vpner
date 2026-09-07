package task

import (
	"log"
	"strconv"
	"strings"
	"time"

	"just-vpn/model"
)

const (
	defaultNodePullInterval       = 5 * time.Minute
	nodePullIntervalCheckInterval = time.Second
)

// StartNodePullTask 启动节点订阅同步任务，启动后立即执行，之后按配置间隔执行
func StartNodePullTask() {
	go func() {
		runNodePull()
		lastPullTime := time.Now()
		ticker := time.NewTicker(nodePullIntervalCheckInterval)
		defer ticker.Stop()
		for now := range ticker.C {
			if now.Sub(lastPullTime) < configuredNodePullInterval() {
				continue
			}
			runNodePull()
			lastPullTime = time.Now()
		}
	}()
}

func configuredNodePullInterval() time.Duration {
	return parseNodePullInterval(model.ConfigValue(model.ConfigNodePullIntervalSeconds, "300"))
}

func parseNodePullInterval(value string) time.Duration {
	seconds, err := strconv.ParseUint(strings.TrimSpace(value), 10, 64)
	if err != nil || seconds == 0 {
		return defaultNodePullInterval
	}
	interval, err := time.ParseDuration(strconv.FormatUint(seconds, 10) + "s")
	if err != nil || interval <= 0 {
		return defaultNodePullInterval
	}
	return interval
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
