package model

import "gorm.io/gorm"

// LuckyWheel 幸运转盘表
type LuckyWheel struct {
	BaseModel
	Level  int    `json:"level" gorm:"column:level;type:int;comment:等级"`
	Title  string `json:"title" gorm:"column:title;type:varchar(128);comment:标题"`
	Desc   string `json:"desc" gorm:"column:desc;type:varchar(128);comment:描述"`
	Remark string `json:"remark" gorm:"column:remark;type:varchar(255);comment:备注"`
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
	var prize LuckyWheel
	err := db.Where("delete_time IS NULL").Order("RAND()").Take(&prize).Error
	return prize, err
}

// InitLuckyWheels 按等级补充初始数据及空描述，保留已有记录的其他修改。
func InitLuckyWheels() error {
	prizes := []LuckyWheel{
		{Level: 1, Title: "10分钟", Desc: "免费会员", Remark: "高速会员专属权益"},
		{Level: 2, Title: "20分钟", Desc: "免费会员", Remark: "高速会员专属权益"},
		{Level: 3, Title: "30分钟", Desc: "免费会员", Remark: "高速会员专属权益"},
		{Level: 4, Title: "1天", Desc: "免费会员", Remark: "高速会员专属权益"},
		{Level: 5, Title: "7天", Desc: "免费会员", Remark: "高速会员专属权益"},
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
		}
		return nil
	})
}
