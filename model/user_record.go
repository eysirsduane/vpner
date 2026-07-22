package model

// UserRecord 用户记录表
type UserRecord struct {
	BaseModel
	UserId int    `json:"user_id" gorm:"column:user_id;type:int;index:idx_user_id;comment:用户ID"`
	Remark string `json:"remark" gorm:"column:remark;type:text;comment:备注"`
}

func (UserRecord) TableName() string { return "user_record" }

func CreateUserRecord(record UserRecord) error {
	return DB.Create(&record).Error
}
