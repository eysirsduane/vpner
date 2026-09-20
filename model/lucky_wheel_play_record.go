package model

import (
	"errors"
	"time"

	"just-vpn/pkg/setting"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrLuckyWheelAlreadyPlayed = errors.New("lucky wheel already played today")

// LuckyWhellPlayRecord 用户幸运大转盘参与记录表
type LuckyWhellPlayRecord struct {
	BaseModel
	UserId int    `json:"user_id" gorm:"column:user_id;type:int;index:idx_user_id;comment:用户ID"`
	Title  string `json:"title" gorm:"column:title;type:varchar(128);comment:奖品标题"`
	Desc   string `json:"desc" gorm:"column:desc;type:varchar(128);comment:奖品描述"`
	Remark string `json:"remark" gorm:"column:remark;type:varchar(255);comment:奖品备注"`
}

func (LuckyWhellPlayRecord) TableName() string { return "lucky_wheel_play_record" }

// PlayLuckyWheel 同一用户每天仅可参与一次，抽奖及记录写入在同一事务中完成。
func PlayLuckyWheel(userId int) (LuckyWheel, error) {
	var prize LuckyWheel
	err := DB.Transaction(func(tx *gorm.DB) error {
		// 锁定始终存在的用户行，即使当天尚无参与记录，也能串行处理同一用户的请求。
		var user User
		if err := tx.Select("id").Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", userId).Take(&user).Error; err != nil {
			return err
		}

		// 获取锁后按中国时区的当前日期计算当天 [00:00, 次日00:00) 的范围。
		now := tx.NowFunc().In(setting.ChinaLocation).Truncate(time.Second)
		dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, setting.ChinaLocation)
		dayEnd := dayStart.AddDate(0, 0, 1)
		var count int64
		// 已软删除的中奖记录同样占用次数，避免删除记录后重复参与。
		if err := tx.Model(&LuckyWhellPlayRecord{}).
			Where("user_id = ? AND create_time >= ? AND create_time < ?", userId, dayStart, dayEnd).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrLuckyWheelAlreadyPlayed
		}

		var err error
		prize, err = getRandomLuckyWheel(tx)
		if err != nil {
			return err
		}
		return tx.Create(&LuckyWhellPlayRecord{
			BaseModel: BaseModel{CreateTime: now},
			UserId:    userId,
			Title:     prize.Title,
			Desc:      prize.Desc,
			Remark:    prize.Remark,
		}).Error
	})
	if err != nil {
		return LuckyWheel{}, err
	}
	return prize, nil
}
