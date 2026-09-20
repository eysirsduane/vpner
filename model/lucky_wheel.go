package model

import (
	"math/rand/v2"
	"time"

	"just-vpn/pkg/redis"

	"gorm.io/gorm"
)

const (
	luckyWheelListCacheKey = "lucky_wheel:available:v2"
	luckyWheelListCacheTTL = 30 * time.Second
)

// LuckyWheel 幸运转盘表
type LuckyWheel struct {
	BaseModel
	Level   int    `json:"level" gorm:"column:level;type:int;comment:等级"`
	Seconds int    `json:"seconds" gorm:"column:seconds;type:int;comment:时长(秒)"`
	Title   string `json:"title" gorm:"column:title;type:varchar(128);comment:标题"`
	Desc    string `json:"desc" gorm:"column:desc;type:varchar(128);comment:描述"`
	Remark  string `json:"remark" gorm:"column:remark;type:varchar(255);comment:备注"`
}

func (LuckyWheel) TableName() string { return "lucky_wheel" }

// GetAvailableLuckyWheels 读取全部未删除的转盘记录。
func GetAvailableLuckyWheels() ([]LuckyWheel, error) {
	var luckyWheels []LuckyWheel
	err := DB.Where("delete_time IS NULL").Find(&luckyWheels).Error
	return luckyWheels, err
}

// GetRandomLuckyWheel 等概率随机读取一条未删除的转盘记录。
func GetRandomLuckyWheel() (LuckyWheel, error) {
	return getRandomLuckyWheel(DB)
}

func getRandomLuckyWheel(db *gorm.DB) (LuckyWheel, error) {
	var prizes []LuckyWheel
	if ok, err := redis.Get(luckyWheelListCacheKey, &prizes); err != nil || !ok {
		if err := db.Where("delete_time IS NULL").Find(&prizes).Error; err != nil {
			return LuckyWheel{}, err
		}
		_ = redis.Set(luckyWheelListCacheKey, prizes, luckyWheelListCacheTTL)
	}
	if len(prizes) == 0 {
		return LuckyWheel{}, gorm.ErrRecordNotFound
	}
	// 缓存完整奖池，每次调用都独立随机抽取，避免缓存单个中奖结果。
	return prizes[rand.IntN(len(prizes))], nil
}

// InitLuckyWheels 按等级补充初始数据、空描述及缺失时长，保留已有有效配置。
func InitLuckyWheels() error {
	prizes := []LuckyWheel{
		{Level: 1, Seconds: 600, Title: "10分钟", Desc: "免费会员", Remark: "高速会员专属权益"},
		{Level: 2, Seconds: 1200, Title: "20分钟", Desc: "免费会员", Remark: "高速会员专属权益"},
		{Level: 3, Seconds: 1800, Title: "30分钟", Desc: "免费会员", Remark: "高速会员专属权益"},
		{Level: 4, Seconds: 86400, Title: "1天", Desc: "免费会员", Remark: "高速会员专属权益"},
		{Level: 5, Seconds: 604800, Title: "7天", Desc: "免费会员", Remark: "高速会员专属权益"},
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		for _, prize := range prizes {
			var existing LuckyWheel
			if err := tx.Where("level = ?", prize.Level).Attrs(prize).FirstOrCreate(&existing).Error; err != nil {
				return err
			}
			if existing.Desc == "" {
				if err := tx.Model(&existing).Where("`desc` IS NULL OR `desc` = ?", "").Update("desc", prize.Desc).Error; err != nil {
					return err
				}
			}
			if existing.Seconds == 0 {
				if err := tx.Model(&existing).Where("seconds IS NULL OR seconds = ?", 0).Update("seconds", prize.Seconds).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}
