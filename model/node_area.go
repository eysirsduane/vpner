package model

import (
	"time"

	"just-vpn/pkg/redis"

	"gorm.io/gorm"
)

// NodeArea VPN节点区域入口表
type NodeArea struct {
	BaseModel
	Code             string `json:"code" gorm:"column:code;type:varchar(64);comment:国家或区域代码"`
	Name             string `json:"name" gorm:"column:name;type:varchar(128);comment:国家或区域名称"`
	Sort             int    `json:"sort" gorm:"column:sort;type:int;comment:排序值，数字越小越靠前"`
	TodayActiveCount int    `json:"today_active_count" gorm:"column:today_active_count;type:int;comment:今日活跃用户数量"`
	MinConnTime      int    `json:"min_conn_time" gorm:"column:min_conn_time;type:int;comment:最小连接时间，单位毫秒"`
	MaxConnTime      int    `json:"max_conn_time" gorm:"column:max_conn_time;type:int;comment:最大连接时间，单位毫秒"`
	Status           int    `json:"status" gorm:"column:status;type:int;comment:状态(0=关闭,1=开启)"`
	ImgUrl           string `json:"img_url" gorm:"column:img_url;type:varchar(255);comment:国家或区域图片地址"`
}

func (NodeArea) TableName() string { return "node_area" }

const nodeAreaCacheKey = "node_area:available"
const nodeAreaCacheTTL = 20 * time.Second

func GetAvailableNodeAreas() ([]NodeArea, error) {
	var areas []NodeArea
	if ok, err := redis.Get(nodeAreaCacheKey, &areas); err == nil && ok {
		return areas, nil
	}

	areas, err := GetAvailableNodeAreasFromDB()
	if err != nil {
		return nil, err
	}
	_ = redis.Set(nodeAreaCacheKey, areas, nodeAreaCacheTTL)
	return areas, nil
}

func GetAvailableNodeAreasFromDB() ([]NodeArea, error) {
	var areas []NodeArea
	err := DB.Where("status = ?", 1).Order("sort ASC").Find(&areas).Error
	return areas, err
}

func AddNodeAreaTodayActiveCount(code string) error {
	return DB.Model(&NodeArea{}).Where("code = ?", code).UpdateColumn("today_active_count", gorm.Expr("today_active_count + ?", 1)).Error
}
