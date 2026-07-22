package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// UserAdvert 用户广告位展示记录表
type UserAdvert struct {
	BaseModel
	UserId       int        `json:"user_id" gorm:"column:user_id;type:int;uniqueIndex:idx_user_advert;index:idx_user_id;comment:用户ID"`
	AdvertId     int        `json:"advert_id" gorm:"column:advert_id;type:int;uniqueIndex:idx_user_advert;index:idx_advert_id;comment:广告ID"`
	ShowTimes    int        `json:"show_times" gorm:"column:show_times;type:int;comment:已展示次数"`
	LastShowTime *time.Time `json:"last_show_time" gorm:"column:last_show_time;type:datetime;comment:最后展示时间"`
}

func (UserAdvert) TableName() string { return "user_advert" }

func GetUserAdvert(userId int, advertId int) (UserAdvert, error) {
	return getUserAdvertRecordFromDB(userId, advertId)
}

func getUserAdvertRecordFromDB(userId int, advertId int) (UserAdvert, error) {
	var record UserAdvert
	err := DB.Where("user_id = ?", userId).
		Where("advert_id = ?", advertId).
		First(&record).Error
	return record, err
}

func addUserAdvertShowToDB(userId int, advertId int, now time.Time) (UserAdvert, error) {
	record, err := getUserAdvertRecordFromDB(userId, advertId)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return UserAdvert{}, err
		}
		record = UserAdvert{
			BaseModel: BaseModel{
				CreateTime: now,
			},
			UserId:       userId,
			AdvertId:     advertId,
			ShowTimes:    1,
			LastShowTime: &now,
		}
		return record, DB.Create(&record).Error
	}
	record.ShowTimes++
	record.LastShowTime = &now
	err = DB.Model(&UserAdvert{}).
		Where("id = ?", record.Id).
		Updates(map[string]interface{}{
			"show_times":     record.ShowTimes,
			"last_show_time": now,
		}).Error
	return record, err
}
