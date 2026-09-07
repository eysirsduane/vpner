package model

import "time"

// BaseModel 通用模型字段
type BaseModel struct {
	Id         int        `json:"id" gorm:"column:id;type:int;primaryKey;autoIncrement;comment:主键ID"`
	CreateTime time.Time  `json:"create_time" gorm:"column:create_time;type:datetime;autoCreateTime;comment:创建时间"`
	UpdateTime *time.Time `json:"update_time" gorm:"column:update_time;type:datetime;autoUpdateTime;comment:更新时间"`
	DeleteTime *time.Time `json:"delete_time" gorm:"column:delete_time;type:datetime;index:idx_delete_time;comment:删除时间"`
}
