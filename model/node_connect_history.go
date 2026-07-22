package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// NodeConnectHistory 节点连接历史表
type NodeConnectHistory struct {
	BaseModel
	UserId         int        `json:"user_id" gorm:"column:user_id;type:int;index:idx_user_connected;comment:用户ID"`
	NodeId         int        `json:"node_id" gorm:"column:node_id;type:int;index:idx_node_history_online;comment:节点ID"`
	Address        string     `json:"address" gorm:"column:address;type:varchar(255);comment:节点地址"`
	Ip             string     `json:"ip" gorm:"column:ip;type:varchar(64);comment:用户IP"`
	Code           string     `json:"code" gorm:"column:code;type:varchar(64);comment:国家代码"`
	CountryName    string     `json:"country_name" gorm:"column:country_name;type:varchar(128);comment:国家名称"`
	Platform       string     `json:"platform" gorm:"column:platform;type:varchar(32);comment:用户设备类型"`
	UserPayStatus  int        `json:"user_pay_status" gorm:"column:user_pay_status;type:int;comment:用户支付状态(1=免费,2=付费)"`
	ConnectedAt    time.Time  `json:"connected_at" gorm:"column:connected_at;type:datetime;index:idx_user_connected;comment:连接时间"`
	DisconnectedAt *time.Time `json:"disconnected_at" gorm:"column:disconnected_at;type:datetime;index:idx_node_history_online;comment:断开时间，空表示未断开"`
}

func (NodeConnectHistory) TableName() string { return "node_connect_history" }

const nodeConnectHistoryStaleAfter = 24 * time.Hour

func EnsureNodeConnectHistoryIndexes() error {
	if DB.Migrator().HasIndex(&NodeConnectHistory{}, "idx_node_history_online") {
		return nil
	}
	return DB.Exec("CREATE INDEX idx_node_history_online ON node_connect_history (node_id, disconnected_at)").Error
}

func CloseOpenNodeConnectHistory(userId int) error {
	now := time.Now()
	return DB.Model(&NodeConnectHistory{}).
		Where("user_id = ?", userId).
		Where("disconnected_at IS NULL").
		Updates(map[string]interface{}{
			"disconnected_at": &now,
		}).Error
}

func ShouldSkipDuplicateNodeConnectHistoryInsert(userId, nodeId int) (bool, error) {
	var history NodeConnectHistory
	err := DB.Where("user_id = ?", userId).
		Where("node_id = ?", nodeId).
		Where("disconnected_at IS NULL").
		Order("id DESC").
		First(&history).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	if history.ConnectedAt.IsZero() {
		return false, nil
	}
	return time.Since(history.ConnectedAt) <= nodeConnectHistoryStaleAfter, nil
}

func CreateNodeConnectHistory(history NodeConnectHistory) error {
	return DB.Create(&history).Error
}
