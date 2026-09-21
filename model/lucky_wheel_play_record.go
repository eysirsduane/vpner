package model

import (
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"just-vpn/pkg/redis"
	"just-vpn/pkg/setting"

	"gorm.io/gorm"
)

var ErrLuckyWheelAlreadyPlayed = errors.New("lucky wheel already played today")

const (
	LuckyWheelUnclaimed = 0
	LuckyWheelClaimed   = 1
)

// LuckyWheelPlayRecord 用户幸运大转盘参与记录表
type LuckyWheelPlayRecord struct {
	BaseModel
	UserId     int    `json:"user_id" gorm:"column:user_id;type:int;index:idx_user_id;comment:用户ID"`
	VipSeconds int    `json:"vip_secs" gorm:"column:vip_secs;type:int;comment:会员时长(秒)"`
	Status     int    `json:"status" gorm:"column:status;type:int;default:0;comment:状态(1=已领取,0=未领取)"`
	Title      string `json:"title" gorm:"column:title;type:varchar(128);comment:奖品标题"`
	Desc       string `json:"desc" gorm:"column:desc;type:varchar(128);comment:奖品描述"`
	Remark     string `json:"remark" gorm:"column:remark;type:varchar(255);comment:奖品备注"`
}

func (LuckyWheelPlayRecord) TableName() string { return "lucky_wheel_play_record" }

func luckyWheelPlayedKey(userID int, now time.Time) string {
	return fmt.Sprintf("lucky_wheel:played:%s:%d", now.In(setting.ChinaLocation).Format("20060102"), userID)
}

// LuckyWheelPlayedToday 与抽奖接口使用同一个 Redis 标记，不修改标记或过期时间。
func LuckyWheelPlayedToday(userID int) (bool, error) {
	return redis.Exists(luckyWheelPlayedKey(userID, DB.NowFunc()))
}

// PlayLuckyWheel 通过 Redis 原子占用当天参与次数，中奖记录通过单条 INSERT 保存。
func PlayLuckyWheel(userId int) (LuckyWheel, error) {
	// 日期参与键名，过期时间为北京时间次日零点。
	now := DB.NowFunc().In(setting.ChinaLocation)
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, setting.ChinaLocation)
	dayEnd := dayStart.AddDate(0, 0, 1)
	if redis.Redis == nil {
		return LuckyWheel{}, errors.New("redis is not initialized")
	}
	key := luckyWheelPlayedKey(userId, now)
	token := rand.Text()
	acquired, err := redis.Redis.SetNX(key, token, dayEnd.Sub(now)).Result()
	if err != nil {
		return LuckyWheel{}, err
	}
	if !acquired {
		return LuckyWheel{}, ErrLuckyWheelAlreadyPlayed
	}
	saved := false
	defer func() {
		if !saved {
			// 只释放本次请求的占位，避免误删其他请求写入的标记。
			_ = redis.Redis.Eval(`if redis.call("GET", KEYS[1]) == ARGV[1] then return redis.call("DEL", KEYS[1]) end return 0`, []string{key}, token).Err()
		}
	}()

	prize, err := getRandomLuckyWheel(DB)
	if err != nil {
		return LuckyWheel{}, err
	}
	if prize.Seconds <= 0 {
		return LuckyWheel{}, errors.New("lucky wheel reward duration invalid")
	}
	err = DB.Session(&gorm.Session{SkipDefaultTransaction: true}).Create(&LuckyWheelPlayRecord{
		BaseModel:  BaseModel{CreateTime: now.Truncate(time.Second)},
		UserId:     userId,
		VipSeconds: prize.Seconds,
		Status:     LuckyWheelUnclaimed,
		Title:      prize.Title,
		Desc:       prize.Desc,
		Remark:     prize.Remark,
	}).Error
	if err != nil {
		return LuckyWheel{}, err
	}
	saved = true
	return prize, nil
}
