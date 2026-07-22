package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// UserPopup 用户弹窗展示记录表
type UserPopup struct {
	BaseModel
	UserId       int        `json:"user_id" gorm:"column:user_id;type:int;uniqueIndex:idx_user_popup;index:idx_user_id;comment:用户ID"`
	PopupId      int        `json:"popup_id" gorm:"column:popup_id;type:int;uniqueIndex:idx_user_popup;index:idx_popup_id;comment:弹窗ID"`
	ShowTimes    int        `json:"show_times" gorm:"column:show_times;type:int;comment:已展示次数"`
	LastShowTime *time.Time `json:"last_show_time" gorm:"column:last_show_time;type:datetime;comment:最后展示时间"`
}

func (UserPopup) TableName() string { return "user_popup" }

func GetUserPopup(userId int, popupId int) (UserPopup, error) {
	return getUserPopupRecordFromDB(userId, popupId)
}

func getUserPopupRecordFromDB(userId int, popupId int) (UserPopup, error) {
	var record UserPopup
	err := DB.Where("user_id = ?", userId).
		Where("popup_id = ?", popupId).
		First(&record).Error
	return record, err
}

func addUserPopupShowToDB(userId int, popupId int, now time.Time) (UserPopup, error) {
	record, err := getUserPopupRecordFromDB(userId, popupId)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return UserPopup{}, err
		}
		record = UserPopup{
			BaseModel: BaseModel{
				CreateTime: now,
			},
			UserId:       userId,
			PopupId:      popupId,
			ShowTimes:    1,
			LastShowTime: &now,
		}
		return record, DB.Create(&record).Error
	}
	record.ShowTimes++
	record.LastShowTime = &now
	err = DB.Model(&UserPopup{}).
		Where("id = ?", record.Id).
		Updates(map[string]interface{}{
			"show_times":     record.ShowTimes,
			"last_show_time": now,
		}).Error
	return record, err
}
