package model

import "time"

const (
	AdvertPositionSplash    = "splash"
	AdvertPositionHomePopup = "home_popup"
	AdvertPositionBanner    = "banner"
)

// Advert 广告位配置表
type Advert struct {
	BaseModel
	Position     string     `json:"position" gorm:"column:position;type:varchar(64);index:idx_advert_position;comment:广告位标识(splash=启动页广告,home_popup=首页弹窗广告,banner=banner广告)"`
	Title        string     `json:"title" gorm:"column:title;type:varchar(128);comment:广告标题"`
	Content      string     `json:"content" gorm:"column:content;type:text;comment:广告内容"`
	ImageUrl     string     `json:"image_url" gorm:"column:image_url;type:varchar(255);comment:广告图片地址"`
	LinkUrl      string     `json:"link_url" gorm:"column:link_url;type:varchar(255);comment:广告点击后使用浏览器打开的链接地址"`
	MaxShowTimes int        `json:"max_show_times" gorm:"column:max_show_times;type:int;comment:每个用户最大展示次数，0表示不限次数"`
	StartTime    *time.Time `json:"start_time" gorm:"column:start_time;type:datetime;index:idx_advert_time;comment:展示开始时间"`
	EndTime      *time.Time `json:"end_time" gorm:"column:end_time;type:datetime;index:idx_advert_time;comment:展示结束时间"`
	Platforms    string     `json:"platforms" gorm:"column:platforms;type:varchar(255);comment:展示平台，多个用英文逗号分隔，空值或all表示全部"`
	Versions     string     `json:"versions" gorm:"column:versions;type:varchar(255);comment:展示版本，多个用英文逗号分隔，空值或all表示全部"`
	Sorter       int        `json:"sorter" gorm:"column:sorter;type:int;index:idx_advert_sorter;comment:排序值，越大越优先"`
}

func (Advert) TableName() string { return "advert" }

func GetAvailableAdverts(now time.Time) ([]Advert, error) {
	var adverts []Advert
	err := DB.Where("delete_time IS NULL").
		Where("position IN ?", []string{
			AdvertPositionSplash,
			AdvertPositionHomePopup,
			AdvertPositionBanner,
		}).
		Where("(start_time IS NULL OR start_time <= ?)", now).
		Where("(end_time IS NULL OR end_time >= ?)", now).
		Order("sorter DESC, id DESC").
		Find(&adverts).Error
	return adverts, err
}
