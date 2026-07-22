package controller

import (
	"time"

	"just-vpn/middleware"
	"just-vpn/model"

	"github.com/gin-gonic/gin"
)

type PopupResponse struct {
	Id           int    `json:"id" example:"1"`                                    // 弹窗ID
	Title        string `json:"title" example:"会员活动"`                              // 弹窗标题
	Content      string `json:"content" example:"限时开通会员享优惠"`                       // 弹窗内容
	ImageUrl     string `json:"image_url" example:"https://example.com/popup.png"` // 弹窗图片地址
	JumpType     string `json:"jump_type" example:"internal"`                      // 跳转方式(none=不跳转,internal=内部跳转,external=外部浏览器)
	JumpTarget   string `json:"jump_target" example:"purchase"`                    // 跳转目标，内部跳转填业务code，外部跳转填URL
	ShowTimes    int    `json:"show_times" example:"1"`                            // 当前用户已展示次数
	MaxShowTimes int    `json:"max_show_times" example:"3"`                        // 每个用户最大展示次数，0表示不限次数
}

// PopupHandler 获取统一弹窗
// @Summary 获取统一弹窗
// @Description 获取当前用户当前平台和版本应该展示的弹窗，并记录展示次数
// @Tags 系统
// @Produce json
// @Security BearerAuth
// @Success 200 {object} Response{result=PopupResponse}
// @Router /popup [get]
func PopupHandler(c *gin.Context) {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	clientInfo := middleware.CurrentClientInfo(c)
	now := time.Now().In(time.Local).Truncate(time.Second)
	popup, record, ok, err := nextPopupForUser(user.Id, clientInfo.Platform, clientInfo.Version, now)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	if !ok {
		JsonReturn(c, CodeSuccess, "success", nil)
		return
	}
	JsonReturn(c, CodeSuccess, "success", PopupResponse{
		Id:           popup.Id,
		Title:        popup.Title,
		Content:      popup.Content,
		ImageUrl:     popup.ImageUrl,
		JumpType:     popup.JumpType,
		JumpTarget:   popup.JumpTarget,
		ShowTimes:    record.ShowTimes,
		MaxShowTimes: popup.MaxShowTimes,
	})
}

func nextPopupForUser(userId int, platform string, version string, now time.Time) (model.Popup, model.UserPopup, bool, error) {
	popups, err := model.GetEnabledPopups(now)
	if err != nil {
		return model.Popup{}, model.UserPopup{}, false, err
	}
	for _, popup := range popups {
		if !model.MatchCSVRule(popup.Platforms, platform) {
			continue
		}
		if !model.MatchCSVRule(popup.Versions, version) {
			continue
		}
		if _, ok, err := model.UserPopupCanShow(userId, popup); err != nil {
			return model.Popup{}, model.UserPopup{}, false, err
		} else if !ok {
			continue
		}
		record, err := model.AddUserPopupShow(userId, popup.Id, now)
		if err != nil {
			return model.Popup{}, model.UserPopup{}, false, err
		}
		return popup, record, true, nil
	}
	return model.Popup{}, model.UserPopup{}, false, nil
}
