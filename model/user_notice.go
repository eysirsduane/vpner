package model

import "time"

const (
	UserNoticeStatusUnread  = 1
	UserNoticeStatusRead    = 2
	UserNoticeStatusDeleted = 3
)

// UserNotice 用户通知表
type UserNotice struct {
	BaseModel
	UserId   int        `json:"user_id" gorm:"column:user_id;type:int;index:idx_user_status;comment:用户ID"`
	Title    string     `json:"title" gorm:"column:title;type:varchar(128);comment:通知标题"`
	Content  string     `json:"content" gorm:"column:content;type:text;comment:通知内容"`
	Priority int        `json:"priority" gorm:"column:priority;type:int;index:idx_priority;comment:优先级，数字越大越靠前"`
	Status   int        `json:"status" gorm:"column:status;type:int;index:idx_user_status;comment:状态(1=未读,2=已读,3=删除)"`
	ReadTime *time.Time `json:"read_time" gorm:"column:read_time;type:datetime;comment:读取时间"`
}

func (UserNotice) TableName() string { return "user_notice" }

func CreateUserNotice(notice UserNotice) error {
	if notice.Status == 0 {
		notice.Status = UserNoticeStatusUnread
	}
	return DB.Create(&notice).Error
}

func GetVisibleUserNotices(userId int) ([]UserNotice, error) {
	var notices []UserNotice
	err := DB.Where("user_id = ?", userId).
		Where("status <> ?", UserNoticeStatusDeleted).
		Order("priority DESC, id DESC").
		Find(&notices).Error
	return notices, err
}

func GetUnreadUserNotices(userId int) ([]UserNotice, error) {
	var notices []UserNotice
	err := DB.Where("user_id = ?", userId).
		Where("status = ?", UserNoticeStatusUnread).
		Order("priority DESC, id DESC").
		Find(&notices).Error
	return notices, err
}

func CountUnreadUserNotices(userId int) (int64, error) {
	var count int64
	err := DB.Model(&UserNotice{}).
		Where("user_id = ?", userId).
		Where("status = ?", UserNoticeStatusUnread).
		Count(&count).Error
	return count, err
}

func MarkUserNoticesRead(userId int, noticeIds []int) error {
	if len(noticeIds) == 0 {
		return nil
	}
	now := time.Now()
	return DB.Model(&UserNotice{}).
		Where("id IN ?", noticeIds).
		Where("user_id = ?", userId).
		Where("status = ?", UserNoticeStatusUnread).
		Updates(map[string]interface{}{
			"status":    UserNoticeStatusRead,
			"read_time": &now,
		}).Error
}
