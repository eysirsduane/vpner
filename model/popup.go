package model

import (
	"strings"
	"time"
)

const (
	PopupStatusClosed  = 0
	PopupStatusEnabled = 1
)

// Popup 统一弹窗配置表
type Popup struct {
	BaseModel
	Title        string     `json:"title" gorm:"column:title;type:varchar(128);comment:弹窗标题"`
	Content      string     `json:"content" gorm:"column:content;type:text;comment:弹窗内容"`
	ImageUrl     string     `json:"image_url" gorm:"column:image_url;type:varchar(255);comment:弹窗图片地址"`
	JumpType     string     `json:"jump_type" gorm:"column:jump_type;type:varchar(32);comment:跳转方式(none=不跳转,internal=内部跳转,external=外部浏览器)"`
	JumpTarget   string     `json:"jump_target" gorm:"column:jump_target;type:varchar(255);comment:跳转目标，内部跳转填业务code，外部跳转填URL"`
	Status       int        `json:"status" gorm:"column:status;type:int;index:idx_popup_status;comment:状态(0=关闭,1=开启)"`
	MaxShowTimes int        `json:"max_show_times" gorm:"column:max_show_times;type:int;comment:每个用户最大展示次数，0表示不限次数"`
	StartTime    *time.Time `json:"start_time" gorm:"column:start_time;type:datetime;index:idx_popup_time;comment:展示开始时间"`
	EndTime      *time.Time `json:"end_time" gorm:"column:end_time;type:datetime;index:idx_popup_time;comment:展示结束时间"`
	Platforms    string     `json:"platforms" gorm:"column:platforms;type:varchar(255);comment:展示平台，多个用英文逗号分隔，空值或all表示全部"`
	Versions     string     `json:"versions" gorm:"column:versions;type:varchar(255);comment:展示版本，多个用英文逗号分隔，空值或all表示全部"`
	Sorter       int        `json:"sorter" gorm:"column:sorter;type:int;index:idx_popup_sorter;comment:排序值，越大越优先"`
}

func (Popup) TableName() string { return "popup" }

func GetEnabledPopups(now time.Time) ([]Popup, error) {
	var popups []Popup
	err := DB.Where("status = ?", PopupStatusEnabled).
		Where("(start_time IS NULL OR start_time <= ?)", now).
		Where("(end_time IS NULL OR end_time >= ?)", now).
		Order("sorter DESC, id DESC").
		Find(&popups).Error
	return popups, err
}

func MatchCSVRule(rule string, value string) bool {
	rule = strings.TrimSpace(rule)
	if rule == "" || strings.EqualFold(rule, "all") {
		return true
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for _, item := range strings.Split(rule, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if strings.EqualFold(item, "all") || item == value {
			return true
		}
	}
	return false
}
