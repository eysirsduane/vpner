package model

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"just-vpn/pkg/redis"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	UserStatusBanned = 0
	UserStatusNormal = 1

	UserBanStatusActive   = 1
	UserBanStatusReleased = 2
)

const (
	userBanKeyPrefix   = "ban:user:"
	deviceBanKeyPrefix = "ban:device:"
)

// UserBan 用户封禁记录表
type UserBan struct {
	BaseModel
	UserId    int        `json:"user_id" gorm:"column:user_id;type:int;index:idx_user_status;comment:用户ID"`
	DeviceNo  string     `json:"device_no" gorm:"column:device_no;type:varchar(128);index:idx_device_status;comment:设备号"`
	Reason    string     `json:"reason" gorm:"column:reason;type:varchar(255);comment:封禁原因"`
	Operator  string     `json:"operator" gorm:"column:operator;type:varchar(128);comment:操作人"`
	BanTime   time.Time  `json:"ban_time" gorm:"column:ban_time;type:datetime;index:idx_ban_time;comment:封禁时间"`
	UnbanTime *time.Time `json:"unban_time" gorm:"column:unban_time;type:datetime;index:idx_unban_time;comment:解封时间"`
	Status    int        `json:"status" gorm:"column:status;type:int;index:idx_user_status;index:idx_device_status;comment:状态(1=封禁中,2=已解封)"`
}

func (UserBan) TableName() string { return "user_ban" }

func UserBanKey(userId int) string {
	return fmt.Sprintf("%s%d", userBanKeyPrefix, userId)
}

func DeviceBanKey(deviceNo string) string {
	return deviceBanKeyPrefix + strings.TrimSpace(deviceNo)
}

func BanUser(userId int, deviceNo string, reason string, operator string) error {
	deviceNo = strings.TrimSpace(deviceNo)
	if userId <= 0 && deviceNo == "" {
		return errors.New("user_id or device_no is required")
	}
	now := time.Now()
	if err := DB.Transaction(func(tx *gorm.DB) error {
		if userId > 0 {
			if err := tx.Model(&User{}).Where("id = ?", userId).Updates(map[string]interface{}{
				"status": UserStatusBanned,
			}).Error; err != nil {
				return err
			}
		}

		var existing UserBan
		query := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("status = ?", UserBanStatusActive)
		switch {
		case userId > 0 && deviceNo != "":
			query = query.Where("user_id = ? OR device_no = ?", userId, deviceNo)
		case userId > 0:
			query = query.Where("user_id = ?", userId)
		default:
			query = query.Where("device_no = ?", deviceNo)
		}
		err := query.Order("id ASC").First(&existing).Error
		if err == nil {
			return tx.Model(&UserBan{}).Where("id = ?", existing.Id).Updates(map[string]interface{}{
				"user_id":    userId,
				"device_no":  deviceNo,
				"reason":     reason,
				"operator":   operator,
				"ban_time":   now,
				"unban_time": nil,
			}).Error
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		record := UserBan{
			BaseModel: BaseModel{
				CreateTime: now,
			},
			UserId:   userId,
			DeviceNo: deviceNo,
			Reason:   reason,
			Operator: operator,
			BanTime:  now,
			Status:   UserBanStatusActive,
		}
		if err := tx.Create(&record).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return SyncUserBanCache(userId, deviceNo)
}

func ReleaseUserBan(userId int, deviceNo string) error {
	deviceNo = strings.TrimSpace(deviceNo)
	if userId <= 0 && deviceNo == "" {
		return errors.New("user_id or device_no is required")
	}
	now := time.Now()
	releasedUserIds := map[int]struct{}{}
	releasedDeviceNos := map[string]struct{}{}
	if err := DB.Transaction(func(tx *gorm.DB) error {
		query := tx.Model(&UserBan{}).Where("status = ?", UserBanStatusActive)
		if userId > 0 {
			query = query.Where("user_id = ?", userId)
		}
		if deviceNo != "" {
			query = query.Where("device_no = ?", deviceNo)
		}
		var bans []UserBan
		if err := query.Find(&bans).Error; err != nil {
			return err
		}
		for _, ban := range bans {
			if ban.UserId > 0 {
				releasedUserIds[ban.UserId] = struct{}{}
			}
			if strings.TrimSpace(ban.DeviceNo) != "" {
				releasedDeviceNos[strings.TrimSpace(ban.DeviceNo)] = struct{}{}
			}
		}
		if userId > 0 {
			releasedUserIds[userId] = struct{}{}
		}
		if deviceNo != "" {
			releasedDeviceNos[deviceNo] = struct{}{}
		}
		updateQuery := tx.Model(&UserBan{}).Where("status = ?", UserBanStatusActive)
		if userId > 0 {
			updateQuery = updateQuery.Where("user_id = ?", userId)
		}
		if deviceNo != "" {
			updateQuery = updateQuery.Where("device_no = ?", deviceNo)
		}
		return updateQuery.Updates(map[string]interface{}{
			"status":     UserBanStatusReleased,
			"unban_time": now,
		}).Error
	}); err != nil {
		return err
	}
	for releasedUserId := range releasedUserIds {
		stillActive, err := hasActiveUserBan(releasedUserId)
		if err != nil {
			return err
		}
		if stillActive {
			if err := DB.Model(&User{}).Where("id = ?", releasedUserId).Updates(map[string]interface{}{
				"status": UserStatusBanned,
			}).Error; err != nil {
				return err
			}
			continue
		}
		if err := DB.Model(&User{}).Where("id = ?", releasedUserId).Updates(map[string]interface{}{
			"status": UserStatusNormal,
		}).Error; err != nil {
			return err
		}
		if err := redis.Del(UserBanKey(releasedUserId)); err != nil {
			return err
		}
	}
	for releasedDeviceNo := range releasedDeviceNos {
		stillActive, err := hasActiveDeviceBan(releasedDeviceNo)
		if err != nil {
			return err
		}
		if stillActive {
			continue
		}
		if err := redis.Del(DeviceBanKey(releasedDeviceNo)); err != nil {
			return err
		}
	}
	return nil
}

func hasActiveUserBan(userId int) (bool, error) {
	var count int64
	if err := DB.Model(&UserBan{}).
		Where("user_id = ?", userId).
		Where("status = ?", UserBanStatusActive).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func hasActiveDeviceBan(deviceNo string) (bool, error) {
	var count int64
	if err := DB.Model(&UserBan{}).
		Where("device_no = ?", strings.TrimSpace(deviceNo)).
		Where("status = ?", UserBanStatusActive).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func SyncUserBanCache(userId int, deviceNo string) error {
	deviceNo = strings.TrimSpace(deviceNo)
	if userId <= 0 && deviceNo == "" {
		return nil
	}
	if userId > 0 {
		if err := redis.Set(UserBanKey(userId), 1, 0); err != nil {
			return err
		}
	}
	if deviceNo != "" {
		if err := redis.Set(DeviceBanKey(deviceNo), 1, 0); err != nil {
			return err
		}
	}
	return nil
}

func ClearUserBanCache(userId int, deviceNo string) error {
	deviceNo = strings.TrimSpace(deviceNo)
	if userId > 0 {
		if err := redis.Del(UserBanKey(userId)); err != nil {
			return err
		}
	}
	if deviceNo != "" {
		if err := redis.Del(DeviceBanKey(deviceNo)); err != nil {
			return err
		}
	}
	return nil
}

func IsUserBannedInCache(userId int, deviceNo string) (bool, error) {
	deviceNo = strings.TrimSpace(deviceNo)
	if userId > 0 {
		ok, err := redis.Exists(UserBanKey(userId))
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}
	if deviceNo != "" {
		ok, err := redis.Exists(DeviceBanKey(deviceNo))
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}

func LoadActiveUserBansToCache() error {
	var bans []UserBan
	if err := DB.Where("status = ?", UserBanStatusActive).
		Find(&bans).Error; err != nil {
		return err
	}
	for _, ban := range bans {
		if err := SyncUserBanCache(ban.UserId, ban.DeviceNo); err != nil {
			return err
		}
	}
	return nil
}
