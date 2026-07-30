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
	Title             string     `json:"title" gorm:"column:title;type:varchar(128);comment:弹窗标题"`
	Content           string     `json:"content" gorm:"column:content;type:text;comment:弹窗内容"`
	ImageUrl          string     `json:"image_url" gorm:"column:image_url;type:text;comment:弹窗图片地址，支持JSON字符串数组或英文逗号分隔"`
	JumpType          string     `json:"jump_type" gorm:"column:jump_type;type:varchar(32);comment:跳转方式(none=不跳转,internal=内部跳转,external=外部浏览器)"`
	JumpTarget        string     `json:"jump_target" gorm:"column:jump_target;type:varchar(255);comment:跳转目标，内部跳转填业务code，外部跳转填URL"`
	CanClose          int        `json:"can_close" gorm:"column:can_close;type:tinyint;not null;default:1;comment:是否可关闭(0=不可关闭,1=可关闭)"`
	Status            int        `json:"status" gorm:"column:status;type:int;index:idx_popup_status;comment:状态(0=关闭,1=开启)"`
	OnlyNewUser       int        `json:"only_new_user" gorm:"column:only_new_user;type:tinyint;not null;default:0;comment:是否仅新用户展示(0=所有用户,1=仅新用户)"`
	OnlyMainland      int        `json:"only_mainland" gorm:"column:only_mainland;type:tinyint;not null;default:1;comment:是否仅中国大陆IP用户展示(0=所有地区,1=仅中国大陆)"`
	NewUserHours      int        `json:"new_user_hours" gorm:"column:new_user_hours;type:int;not null;default:24;comment:新用户注册时长阈值，单位小时"`
	MaxShowTimes      int        `json:"max_show_times" gorm:"column:max_show_times;type:int;comment:每个用户最大展示次数，0表示不限次数"`
	MaxDailyShowTimes int        `json:"max_daily_show_times" gorm:"column:max_daily_show_times;type:int;not null;default:0;comment:每天全体用户最大展示次数，0表示不限次数"`
	StartTime         *time.Time `json:"start_time" gorm:"column:start_time;type:datetime;index:idx_popup_time;comment:展示开始时间"`
	EndTime           *time.Time `json:"end_time" gorm:"column:end_time;type:datetime;index:idx_popup_time;comment:展示结束时间"`
	Platforms         string     `json:"platforms" gorm:"column:platforms;type:varchar(255);comment:展示平台，多个用英文逗号分隔，空值或all表示全部"`
	Versions          string     `json:"versions" gorm:"column:versions;type:varchar(255);comment:展示版本，多个用英文逗号分隔，空值或all表示全部"`
	Sorter            int        `json:"sorter" gorm:"column:sorter;type:int;index:idx_popup_sorter;comment:排序值，越大越优先"`
}

func (Popup) TableName() string { return "popup" }

func (popup Popup) CanShowToRegisteredUser(registerTime time.Time, now time.Time) bool {
	if popup.OnlyNewUser != 1 {
		return true
	}
	if popup.NewUserHours <= 0 || registerTime.IsZero() || registerTime.After(now) {
		return false
	}
	return now.Sub(registerTime).Hours() <= float64(popup.NewUserHours)
}

func (popup Popup) CanShowToIPRegion(region string) bool {
	if popup.OnlyMainland != 1 {
		return true
	}
	parts := strings.Split(strings.TrimSpace(region), "|")
	if len(parts) < 5 || strings.TrimSpace(parts[1]) != "中国" {
		return false
	}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.Contains(part, "香港") || strings.Contains(part, "澳门") || strings.Contains(part, "台湾") {
			return false
		}
	}
	return true
}

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
