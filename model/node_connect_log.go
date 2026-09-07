package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// NodeConnectLog 节点当前连接日志表
type NodeConnectLog struct {
	BaseModel
	NodeId        int    `json:"node_id" gorm:"column:node_id;type:int;index:idx_node_status_update;comment:节点ID"`
	Address       string `json:"address" gorm:"column:address;type:varchar(255);comment:节点IP地址"`
	Ip            string `json:"ip" gorm:"column:ip;type:varchar(64);comment:用户IP"`
	Code          string `json:"code" gorm:"column:code;type:varchar(64);comment:国家代码"`
	CountryName   string `json:"country_name" gorm:"column:country_name;type:varchar(128);comment:国家名称"`
	UserId        int    `json:"user_id" gorm:"column:user_id;type:int;index:idx_user_id;comment:用户ID"`
	Platform      string `json:"platform" gorm:"column:platform;type:varchar(32);comment:用户设备类型"`
	UserPayStatus int    `json:"user_pay_status" gorm:"column:user_pay_status;type:int;comment:用户支付状态(1=免费,2=付费)"`
	Status        int    `json:"status" gorm:"column:status;type:int;index:idx_node_status_update;comment:连接状态(1=连接中,2=已断开)"`
}

func (NodeConnectLog) TableName() string { return "node_connect_log" }

func EnsureNodeConnectLogIndexes() error {
	if DB.Migrator().HasIndex(&NodeConnectLog{}, "idx_node_connect_status_update_time") {
		return nil
	}
	return DB.Exec("CREATE INDEX idx_node_connect_status_update_time ON node_connect_log (status, update_time)").Error
}

func GetNodeConnectLogByUserId(userId int) (NodeConnectLog, error) {
	var log NodeConnectLog
	err := DB.Where("user_id = ?", userId).Order("id DESC").First(&log).Error
	return log, err
}

func SaveNodeConnectedLog(log NodeConnectLog) error {
	current, err := GetNodeConnectLogByUserId(log.UserId)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return DB.Create(&log).Error
	}
	return DB.Model(&NodeConnectLog{}).Where("id = ?", current.Id).Updates(map[string]interface{}{
		"node_id":         log.NodeId,
		"address":         log.Address,
		"ip":              log.Ip,
		"code":            log.Code,
		"country_name":    log.CountryName,
		"platform":        log.Platform,
		"user_pay_status": log.UserPayStatus,
		"status":          1,
	}).Error
}

func TouchNodeConnectLog(id int) error {
	now := time.Now()
	return DB.Model(&NodeConnectLog{}).Where("id = ?", id).Update("update_time", &now).Error
}

func GetStaleNodeConnectLogs(before time.Time, limit int) ([]NodeConnectLog, error) {
	var logs []NodeConnectLog
	err := DB.Where("status = ?", 1).
		Where(DB.Where("update_time IS NULL").Or("update_time < ?", before)).
		Order("update_time ASC, id ASC").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}

func DisconnectNodeConnectLog(userId int) (bool, error) {
	now := time.Now()
	result := DB.Model(&NodeConnectLog{}).Where("user_id = ?", userId).Where("status = ?", 1).Updates(map[string]interface{}{
		"status":      2,
		"update_time": &now,
	})
	return result.RowsAffected > 0, result.Error
}
