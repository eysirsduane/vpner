package controller

import (
	"time"

	"just-vpn/middleware"
	"just-vpn/model"

	"github.com/gin-gonic/gin"
)

// AdvertResponse 广告位信息
type AdvertResponse struct {
	Id           int    `json:"id" example:"1"`                                     // 广告ID
	Position     string `json:"position" example:"banner"`                          // 广告位标识(splash=启动页广告,home_popup=首页弹窗广告,banner=banner广告)
	Title        string `json:"title" example:"会员优惠"`                               // 广告标题
	Content      string `json:"content" example:"限时开通会员享优惠"`                        // 广告内容
	ImageUrl     string `json:"image_url" example:"https://example.com/advert.png"` // 广告图片地址
	LinkUrl      string `json:"link_url" example:"https://example.com"`             // 广告点击后使用浏览器打开的链接地址
	ShowTimes    int    `json:"show_times" example:"1"`                             // 当前用户已展示次数
	MaxShowTimes int    `json:"max_show_times" example:"3"`                         // 每个用户最大展示次数，0表示不限次数
	Sorter       int    `json:"sorter" example:"100"`                               // 排序值，越大越优先
}

// AdvertHandler 获取广告位列表
// @Summary 获取广告位列表
// @Description 获取当前用户当前平台和版本可展示的广告位列表，并记录展示次数
// @Tags 广告
// @Produce json
// @Security BearerAuth
// @Success 200 {object} Response{result=[]AdvertResponse}
// @Router /advert [get]
func AdvertHandler(c *gin.Context) {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	clientInfo := middleware.CurrentClientInfo(c)
	now := time.Now().In(time.Local).Truncate(time.Second)

	adverts, err := model.GetAvailableAdverts(now)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}

	result := make([]AdvertResponse, 0, len(adverts))
	for _, advert := range adverts {
		if !model.MatchCSVRule(advert.Platforms, clientInfo.Platform) {
			continue
		}
		if !model.MatchCSVRule(advert.Versions, clientInfo.Version) {
			continue
		}
		if _, ok, err := model.UserAdvertCanShow(user.Id, advert); err != nil {
			JsonReturn(c, CodeError, err.Error(), nil)
			return
		} else if !ok {
			continue
		}
		record, err := model.AddUserAdvertShow(user.Id, advert.Id, now)
		if err != nil {
			JsonReturn(c, CodeError, err.Error(), nil)
			return
		}
		result = append(result, AdvertResponse{
			Id:           advert.Id,
			Position:     advert.Position,
			Title:        localizedTextValue(advert.Title, clientInfo.Language),
			Content:      localizedTextValue(advert.Content, clientInfo.Language),
			ImageUrl:     advert.ImageUrl,
			LinkUrl:      advert.LinkUrl,
			ShowTimes:    record.ShowTimes,
			MaxShowTimes: advert.MaxShowTimes,
			Sorter:       advert.Sorter,
		})
	}

	JsonReturn(c, CodeSuccess, "success", result)
}
