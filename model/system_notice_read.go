package model

import "time"

// SystemNoticeRead 系统通知已读记录表
type SystemNoticeRead struct {
	BaseModel
	UserId   int       `json:"user_id" gorm:"column:user_id;type:int;uniqueIndex:idx_user_notice;comment:用户ID"`
	NoticeId int       `json:"notice_id" gorm:"column:notice_id;type:int;uniqueIndex:idx_user_notice;comment:系统通知ID"`
	ReadTime time.Time `json:"read_time" gorm:"column:read_time;type:datetime;comment:读取时间"`
}

func (SystemNoticeRead) TableName() string { return "system_notice_read" }
