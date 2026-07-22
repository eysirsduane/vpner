package model

// Error 错误日志表
type Error struct {
	BaseModel
	Desc          string `json:"desc" gorm:"column:desc;type:text;comment:错误信息"`
	MsgType       string `json:"msg_type" gorm:"column:msg_type;type:varchar(64);comment:消息类型"`
	Uid           int    `json:"uid" gorm:"column:uid;type:int;comment:用户ID"`
	DeviceType    string `json:"device_type" gorm:"column:device_type;type:varchar(64);comment:设备类型"`
	DeviceVersion string `json:"device_version" gorm:"column:device_version;type:varchar(64);comment:设备版本"`
}

func (Error) TableName() string { return "error" }
