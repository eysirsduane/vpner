package model

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"just-vpn/pkg/redis"

	"gorm.io/gorm"
)

const (
	NodeStatusDisabled = 0
	NodeStatusEnabled  = 2
)

// Node VPN节点表
type Node struct {
	BaseModel
	Code             string     `json:"code" gorm:"column:code;type:varchar(64);comment:所属国家代码"`
	CodeName         string     `json:"code_name" gorm:"column:code_name;type:varchar(128);comment:所属国家名称"`
	IncludeAuto      int        `json:"include_auto" gorm:"column:include_auto;type:tinyint;not null;default:0;index;comment:是否同时参与自动线路 1是 0否"`
	Name             string     `json:"name" gorm:"column:name;type:varchar(128);comment:节点名称"`
	NodeType         string     `json:"node_type" gorm:"column:node_type;type:varchar(32);comment:节点类型"`
	LinkUrl          string     `json:"link_url" gorm:"column:link_url;type:text;comment:链接地址或JSON节点数据"`
	Address          string     `json:"address" gorm:"column:address;type:varchar(255);comment:节点地址"`
	Status           int        `json:"status" gorm:"column:status;type:int;comment:状态(0=关闭,2=正常)"`
	TimeoutCount     int        `json:"timeout_count" gorm:"column:timeout_count;type:int;comment:超时次数"`
	TodayActiveCount int        `json:"today_active_count" gorm:"column:today_active_count;type:int;comment:今日活跃用户数量"`
	WallTime         *time.Time `json:"wall_time" gorm:"column:wall_time;type:datetime;comment:被墙时间"`
	DisableTime      *time.Time `json:"disable_time" gorm:"column:disable_time;type:datetime;comment:禁用时间"`
}

func (Node) TableName() string { return "node" }

const nodeListCacheTTL = 20 * time.Second

var availableNodeCacheVersion atomic.Uint64

type NodeOnlineCount struct {
	NodeId int `json:"node_id"`
	Count  int `json:"count"`
}

func GetAvailableNodesByCode(code string) ([]Node, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	cacheKey := availableNodeCacheKey(code)
	var nodes []Node
	if ok, err := redis.Get(cacheKey, &nodes); err == nil && ok {
		return nodes, nil
	}

	nodes, err := GetAvailableNodesByCodeFromDB(code)
	if err != nil {
		return nil, err
	}
	_ = redis.Set(cacheKey, nodes, nodeListCacheTTL)
	return nodes, nil
}

func availableNodeCacheKey(code string) string {
	return fmt.Sprintf("node:available:%d:%s", availableNodeCacheVersion.Load(), code)
}

// InvalidateAvailableNodeCache 使后续节点请求立即读取同步后的节点状态
func InvalidateAvailableNodeCache() {
	availableNodeCacheVersion.Add(1)
}

func GetAvailableNodesByCodeFromDB(code string) ([]Node, error) {
	var nodes []Node
	query := DB.Where("status = ?", NodeStatusEnabled)
	if code == nodeSubscriptionCode {
		query = query.Where("(code = ? OR include_auto = ?)", nodeSubscriptionCode, 1)
	} else {
		query = query.Where("code = ?", code)
	}
	err := query.Order("id ASC").Find(&nodes).Error
	return nodes, err
}

func GetNodeOnlineCountMap(nodes []Node) map[int]int {
	countMap := make(map[int]int, len(nodes))
	if len(nodes) == 0 {
		return countMap
	}

	nodeIds := make([]int, 0, len(nodes))
	for _, node := range nodes {
		nodeIds = append(nodeIds, node.Id)
		countMap[node.Id] = 0
	}

	var counts []NodeOnlineCount
	err := DB.Model(&NodeConnectHistory{}).
		Select("node_id, COUNT(*) as count").
		Where("node_id IN ?", nodeIds).
		Where("disconnected_at IS NULL").
		Group("node_id").
		Find(&counts).Error
	if err != nil {
		return countMap
	}

	for _, item := range counts {
		countMap[item.NodeId] = item.Count
	}
	return countMap
}

func SelectNodeByOnlineCount(nodes []Node) (Node, bool) {
	if len(nodes) == 0 {
		return Node{}, false
	}

	countMap := GetNodeOnlineCountMap(nodes)
	selected := nodes[0]
	minCount := countMap[selected.Id]
	for _, node := range nodes[1:] {
		count := countMap[node.Id]
		if count < minCount {
			selected = node
			minCount = count
		}
	}
	return selected, true
}

func GetAvailableNode(code string) (Node, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	nodes, err := GetAvailableNodesByCode(code)
	if err != nil {
		return Node{}, err
	}
	if len(nodes) == 0 && code != "AUTO" {
		nodes, err = GetAvailableNodesByCode("AUTO")
		if err != nil {
			return Node{}, err
		}
	}
	node, ok := SelectNodeByOnlineCount(nodes)
	if !ok {
		return Node{}, gorm.ErrRecordNotFound
	}
	return nodeForRequestedCode(node, code), nil
}

func nodeForRequestedCode(node Node, requestedCode string) Node {
	if strings.EqualFold(strings.TrimSpace(requestedCode), nodeSubscriptionCode) && !strings.EqualFold(node.Code, nodeSubscriptionCode) {
		node.Code = nodeSubscriptionCode
		node.CodeName = nodeSubscriptionName
		node.Name = nodeSubscriptionName + "节点"
	}
	return node
}

func AddNodeTodayActiveCount(id int) error {
	return DB.Model(&Node{}).Where("id = ?", id).UpdateColumn("today_active_count", gorm.Expr("today_active_count + ?", 1)).Error
}
