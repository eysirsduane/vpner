package controller

import (
	"time"

	"just-vpn/middleware"
	"just-vpn/model"
	"just-vpn/pkg/mapping"

	"github.com/gin-gonic/gin"
)

type DelayedPopupResponse struct {
	Id           int      `json:"id" example:"1"`                                  // 弹窗ID
	Title        string   `json:"title" example:"服务迁移提醒"`                          // 弹窗标题
	Content      string   `json:"content" example:"如当前软件无法使用，请转移到新软件"`             // 弹窗内容
	ImageUrl     []string `json:"image_url" example:"/api/v1/upload/pop.png"`      // 弹窗图片地址数组
	LinkUrl      string   `json:"link_url" example:"https://example.com/download"` // 弹窗跳转链接
	CanClose     int      `json:"can_close" example:"1"`                           // 是否可关闭(0=不可关闭,1=可关闭)
	DelayDays    int      `json:"delay_days" example:"3"`                          // 断网后延迟展示天数
	TransferCode string   `json:"transfer_code" example:"origin8f3k9q"`            // 当前用户转移码
}

// DelayedPopupHandler 获取延迟弹窗
// @Summary 获取延迟弹窗
// @Description 获取客户端断网后本地延迟展示的迁移弹窗，并返回当前用户转移码
// @Tags 系统
// @Produce json
// @Security BearerAuth
// @Success 200 {object} Response{result=DelayedPopupResponse}
// @Router /delayed_popup [get]
func DelayedPopupHandler(c *gin.Context) {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	if err := model.EnsureUserTransferCode(model.DB, &user, currentProductRouteCode()); err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}

	clientInfo := middleware.CurrentClientInfo(c)
	now := time.Now().In(time.Local).Truncate(time.Second)
	popup, ok, err := nextDelayedPopup(clientInfo.Platform, clientInfo.Version, now)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	if !ok {
		JsonReturn(c, CodeSuccess, "success", nil)
		return
	}

	JsonReturn(c, CodeSuccess, "success", DelayedPopupResponse{
		Id:           popup.Id,
		Title:        popup.Title,
		Content:      popup.Content,
		ImageUrl:     popupImageResponseValue(popup.ImageUrl),
		LinkUrl:      popup.LinkUrl,
		CanClose:     popup.CanClose,
		DelayDays:    popup.DelayDays,
		TransferCode: user.TransferCode,
	})
}

func nextDelayedPopup(platform string, version string, now time.Time) (model.DelayedPopup, bool, error) {
	popups, err := model.GetEnabledDelayedPopups(now)
	if err != nil {
		return model.DelayedPopup{}, false, err
	}
	for _, popup := range popups {
		if !model.MatchCSVRule(popup.Platforms, platform) {
			continue
		}
		if !model.MatchCSVRule(popup.Versions, version) {
			continue
		}
		if popup.DelayDays < 0 {
			popup.DelayDays = 0
		}
		return popup, true, nil
	}
	return model.DelayedPopup{}, false, nil
}

func currentProductRouteCode() string {
	return mapping.ProductCode()
}
