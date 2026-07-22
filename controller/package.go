package controller

import (
	"just-vpn/model"

	"github.com/gin-gonic/gin"
)

// PackageResponse 套餐信息
type PackageResponse struct {
	Id          int    `json:"id" example:"1"`               // 套餐ID
	AppleId     string `json:"apple_id" example:"vip_month"` // 苹果内购产品ID
	Name        string `json:"name" example:"月度会员"`          // 套餐名称
	SubName     string `json:"sub_name" example:"连续30天高速线路"` // 套餐副标题
	Selected    int    `json:"selected" example:"1"`         // 是否默认选中(1=选中,0=未选中)
	Corner      string `json:"corner" example:"推荐"`          // 角标文案
	Price       int    `json:"price" example:"1990"`         // 真实价格，单位分
	PriceText   string `json:"price_text" example:"¥19.9"`   // 展示价格文案
	OriginPrice string `json:"origin_price" example:"¥29.9"` // 原价文案
	Value       string `json:"value" example:"畅享全部会员线路"`     // 底部描述内容
	Remark      string `json:"remark" example:"适合短期使用"`      // 套餐描述
	Day         int    `json:"day" example:"30"`             // 套餐天数
}

// PackagesHandler 获取套餐列表
// @Summary 获取套餐列表
// @Description 获取当前启用的会员套餐列表
// @Tags 套餐
// @Produce json
// @Security BearerAuth
// @Success 200 {object} Response{result=[]PackageResponse}
// @Router /packages [get]
func PackagesHandler(c *gin.Context) {
	packages, err := model.GetEnabledPackages()
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}

	result := make([]PackageResponse, 0, len(packages))
	for _, item := range packages {
		result = append(result, PackageResponse{
			Id:          item.Id,
			AppleId:     item.AppleId,
			Name:        item.Name,
			SubName:     item.SubName,
			Selected:    item.Selected,
			Corner:      item.Corner,
			Price:       item.Price,
			PriceText:   item.PakTips,
			OriginPrice: item.OriginPrice,
			Value:       item.Value,
			Remark:      item.Remark,
			Day:         item.Day,
		})
	}

	JsonReturn(c, CodeSuccess, "success", result)
}
