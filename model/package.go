package model

// Package 套餐表
type Package struct {
	BaseModel
	AppleId     string `json:"apple_id" gorm:"column:apple_id;type:varchar(128);comment:苹果产品ID"`
	Name        string `json:"name" gorm:"column:name;type:varchar(128);comment:名称"`
	SubName     string `json:"sub_name" gorm:"column:sub_name;type:varchar(128);comment:套餐副标题"`
	Selected    int    `json:"selected" gorm:"column:selected;type:int;comment:是否默认选中(1=选中,0=未选中)"`
	Corner      string `json:"corner" gorm:"column:corner;type:varchar(64);comment:角标文案"`
	Value       string `json:"value" gorm:"column:value;type:varchar(128);comment:值"`
	Price       int    `json:"price" gorm:"column:price;type:int;comment:订单金额，单位分"`
	OriginPrice string `json:"origin_price" gorm:"column:origin_price;type:varchar(64);comment:原价"`
	PakTips     string `json:"pak_tips" gorm:"column:pak_tips;type:varchar(255);comment:套餐提示"`
	Remark      string `json:"remark" gorm:"column:remark;type:text;comment:备注"`
	Day         int    `json:"day" gorm:"column:day;type:int;comment:套餐天数"`
	Status      int    `json:"status" gorm:"column:status;type:int;comment:状态(1=启用,0=禁用)"`
	Sorter      int    `json:"sorter" gorm:"column:sorter;type:int;comment:排序值"`
}

func (Package) TableName() string { return "package" }

func GetEnabledPackageByID(id int) (Package, error) {
	var pack Package
	err := DB.Where("id = ?", id).
		Where("status = ?", 1).
		First(&pack).Error
	return pack, err
}

func GetEnabledPackageByAppleId(appleId string) (Package, error) {
	var pack Package
	err := DB.Where("apple_id = ?", appleId).
		Where("status = ?", 1).
		First(&pack).Error
	return pack, err
}

func GetEnabledPackages() ([]Package, error) {
	var packages []Package
	err := DB.Where("status = ?", 1).
		Order("sorter ASC, id ASC").
		Find(&packages).Error
	return packages, err
}
