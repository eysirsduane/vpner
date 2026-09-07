package model

// Version 版本更新表
type Version struct {
	BaseModel
	Name     string `json:"name" gorm:"column:name;type:varchar(128);comment:版本名称"`
	Platform string `json:"platform" gorm:"column:platform;type:varchar(32);index:idx_platform_status_build;comment:设备平台(iphone=苹果,android=安卓)"`
	Version  string `json:"version" gorm:"column:version;type:varchar(64);comment:展示版本号"`
	Build    int    `json:"build" gorm:"column:build;type:int;index:idx_platform_status_build;comment:数字版本号，用于版本比较"`
	Force    bool   `json:"force" gorm:"column:force;type:boolean;comment:是否强制更新(false=否,true=是)"`
	Url      string `json:"url" gorm:"column:url;type:varchar(255);comment:下载地址"`
	Size     string `json:"size" gorm:"column:size;type:varchar(64);comment:安装包大小"`
	Content  string `json:"content" gorm:"column:content;type:text;comment:更新内容"`
	Status   int    `json:"status" gorm:"column:status;type:int;index:idx_platform_status_build;comment:状态(1=启用,0=禁用)"`
	Remark   string `json:"remark" gorm:"column:remark;type:text;comment:备注"`
}

func (Version) TableName() string { return "version" }

func GetLatestEnabledVersion(platform string) (Version, error) {
	var version Version
	err := DB.Where("platform = ?", platform).
		Where("status = ?", 1).
		Order("build desc").
		First(&version).Error
	return version, err
}
