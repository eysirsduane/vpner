package model

import "time"

const (
	DelayedPopupStatusClosed  = 0
	DelayedPopupStatusEnabled = 1
)

// DelayedPopup 延迟弹窗配置表
type DelayedPopup struct {
	BaseModel
	Title     string     `json:"title" gorm:"column:title;type:varchar(128);comment:弹窗标题"`
	Content   string     `json:"content" gorm:"column:content;type:text;comment:弹窗内容"`
	ImageUrl  string     `json:"image_url" gorm:"column:image_url;type:text;comment:弹窗图片地址，支持JSON字符串数组或英文逗号分隔"`
	LinkUrl   string     `json:"link_url" gorm:"column:link_url;type:varchar(255);comment:弹窗跳转链接"`
	CanClose  int        `json:"can_close" gorm:"column:can_close;type:tinyint;not null;default:1;comment:是否可关闭(0=不可关闭,1=可关闭)"`
	DelayDays int        `json:"delay_days" gorm:"column:delay_days;type:int;comment:断网后延迟展示天数"`
	Status    int        `json:"status" gorm:"column:status;type:int;index:idx_delayed_popup_status;comment:状态(0=关闭,1=开启)"`
	StartTime *time.Time `json:"start_time" gorm:"column:start_time;type:datetime;index:idx_delayed_popup_time;comment:生效开始时间"`
	EndTime   *time.Time `json:"end_time" gorm:"column:end_time;type:datetime;index:idx_delayed_popup_time;comment:生效结束时间"`
	Platforms string     `json:"platforms" gorm:"column:platforms;type:varchar(255);comment:展示平台，多个用英文逗号分隔，空值或all表示全部"`
	Versions  string     `json:"versions" gorm:"column:versions;type:varchar(255);comment:展示版本，多个用英文逗号分隔，空值或all表示全部"`
	Sorter    int        `json:"sorter" gorm:"column:sorter;type:int;index:idx_delayed_popup_sorter;comment:排序值，越大越优先"`
}

func (DelayedPopup) TableName() string { return "delayed_popup" }

func GetEnabledDelayedPopups(now time.Time) ([]DelayedPopup, error) {
	var popups []DelayedPopup
	err := DB.Where("status = ?", DelayedPopupStatusEnabled).
		Where("(start_time IS NULL OR start_time <= ?)", now).
		Where("(end_time IS NULL OR end_time >= ?)", now).
		Order("sorter DESC, id DESC").
		Find(&popups).Error
	return popups, err
}
