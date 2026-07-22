package model

import (
	"fmt"
	"log"
	"sync"
	"time"

	"just-vpn/pkg/redis"
)

const (
	nodeOnlineKeyPrefix        = "vpn:online:"
	nodeOnlineTTL              = 90 * time.Second
	nodeOnlineCleanupInterval  = 5 * time.Minute
	nodeOnlineCleanupThreshold = 2 * time.Minute
	nodeOnlineCleanupBatchSize = 5000
)

type NodeOnlineStatus struct {
	UserId          int       `json:"user_id"`
	NodeId          int       `json:"node_id"`
	Code            string    `json:"code"`
	ConnectedAt     time.Time `json:"connected_at"`
	LastHeartbeatAt time.Time `json:"last_heartbeat_at"`
}

var nodeOnlineCleanerOnce sync.Once

func NodeOnlineKey(userId int) string {
	return fmt.Sprintf("%s%d", nodeOnlineKeyPrefix, userId)
}

func SaveNodeOnlineStatus(userId int, node Node) error {
	now := time.Now()
	status := NodeOnlineStatus{
		UserId:          userId,
		NodeId:          node.Id,
		Code:            node.Code,
		ConnectedAt:     now,
		LastHeartbeatAt: now,
	}
	return redis.Set(NodeOnlineKey(userId), status, nodeOnlineTTL)
}

func RefreshNodeOnlineStatus(userId int) (bool, error) {
	var status NodeOnlineStatus
	ok, err := redis.Get(NodeOnlineKey(userId), &status)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	status.LastHeartbeatAt = time.Now()
	if err := redis.Set(NodeOnlineKey(userId), status, nodeOnlineTTL); err != nil {
		return false, err
	}
	return true, nil
}

func DeleteNodeOnlineStatus(userId int) error {
	return redis.Del(NodeOnlineKey(userId))
}

func StartNodeOnlineCleaner() {
	nodeOnlineCleanerOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(nodeOnlineCleanupInterval)
			defer ticker.Stop()
			for range ticker.C {
				if err := CleanupExpiredNodeOnlineLogs(); err != nil {
					log.Printf("cleanup expired node online logs failed: %v", err)
				}
			}
		}()
	})
}

func CleanupExpiredNodeOnlineLogs() error {
	logs, err := GetStaleNodeConnectLogs(time.Now().Add(-nodeOnlineCleanupThreshold), nodeOnlineCleanupBatchSize)
	if err != nil {
		return err
	}
	for _, connectLog := range logs {
		online, err := redis.Exists(NodeOnlineKey(connectLog.UserId))
		if err != nil {
			return err
		}
		if online {
			if err := TouchNodeConnectLog(connectLog.Id); err != nil {
				return err
			}
			continue
		}
		if _, err := DisconnectNodeConnectLog(connectLog.UserId); err != nil {
			return err
		}
		if err := CloseOpenNodeConnectHistory(connectLog.UserId); err != nil {
			return err
		}
	}
	return nil
}
