package model

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	SystemNoticeStatusEnabled  = 1
	SystemNoticeStatusDisabled = 0
)

// SystemNotice 系统通知表
type SystemNotice struct {
	BaseModel
	Title     string     `json:"title" gorm:"column:title;type:varchar(128);comment:通知标题"`
	Content   string     `json:"content" gorm:"column:content;type:text;comment:通知内容"`
	Platform  string     `json:"platform" gorm:"column:platform;type:varchar(32);index:idx_status_platform_time;comment:平台(all=全部,iphone=苹果,android=安卓)"`
	Priority  int        `json:"priority" gorm:"column:priority;type:int;index:idx_priority;comment:优先级，数字越大越靠前"`
	Status    int        `json:"status" gorm:"column:status;type:int;index:idx_status_platform_time;comment:状态(1=启用,0=禁用)"`
	StartTime *time.Time `json:"start_time" gorm:"column:start_time;type:datetime;index:idx_status_platform_time;comment:开始展示时间"`
	EndTime   *time.Time `json:"end_time" gorm:"column:end_time;type:datetime;index:idx_status_platform_time;comment:结束展示时间"`
}

func (SystemNotice) TableName() string { return "system_notice" }

func GetAllVisibleSystemNotices(platform string) ([]SystemNotice, error) {
	var notices []SystemNotice
	err := visibleSystemNoticeQuery(platform).Order("priority DESC, id DESC").Find(&notices).Error
	return notices, err
}

func GetUnreadSystemNotices(userId int, platform string) ([]SystemNotice, error) {
	var notices []SystemNotice
	err := visibleSystemNoticeQuery(platform).
		Where("NOT EXISTS (?)", DB.Model(&SystemNoticeRead{}).
			Select("1").
			Where("system_notice_read.notice_id = system_notice.id").
			Where("system_notice_read.user_id = ?", userId)).
		Order("priority DESC, id DESC").
		Find(&notices).Error
	return notices, err
}

func CountUnreadSystemNotices(userId int, platform string) (int64, error) {
	var count int64
	err := visibleSystemNoticeQuery(platform).
		Where("NOT EXISTS (?)", DB.Model(&SystemNoticeRead{}).
			Select("1").
			Where("system_notice_read.notice_id = system_notice.id").
			Where("system_notice_read.user_id = ?", userId)).
		Count(&count).Error
	return count, err
}

func GetReadSystemNoticeIDMap(userId int, noticeIds []int) (map[int]bool, error) {
	readMap := make(map[int]bool, len(noticeIds))
	if len(noticeIds) == 0 {
		return readMap, nil
	}

	var reads []SystemNoticeRead
	err := DB.Where("user_id = ?", userId).
		Where("notice_id IN ?", noticeIds).
		Find(&reads).Error
	if err != nil {
		return nil, err
	}
	for _, read := range reads {
		readMap[read.NoticeId] = true
	}
	return readMap, nil
}

func MarkSystemNoticesRead(userId int, noticeIds []int, platform string) error {
	if len(noticeIds) == 0 {
		return nil
	}

	var visibleIds []int
	if err := visibleSystemNoticeQuery(platform).
		Where("id IN ?", noticeIds).
		Pluck("id", &visibleIds).Error; err != nil {
		return err
	}
	if len(visibleIds) == 0 {
		return nil
	}

	now := time.Now()
	reads := make([]SystemNoticeRead, 0, len(visibleIds))
	for _, noticeId := range visibleIds {
		reads = append(reads, SystemNoticeRead{
			UserId:   userId,
			NoticeId: noticeId,
			ReadTime: now,
		})
	}
	return DB.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(reads, 100).Error
}

func visibleSystemNoticeQuery(platform string) *gorm.DB {
	now := time.Now()
	return DB.Model(&SystemNotice{}).
		Where("system_notice.status = ?", SystemNoticeStatusEnabled).
		Where("system_notice.platform = ? OR system_notice.platform = ?", "all", platform).
		Where("(system_notice.start_time IS NULL OR system_notice.start_time <= ?)", now).
		Where("(system_notice.end_time IS NULL OR system_notice.end_time >= ?)", now)
}
