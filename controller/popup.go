package controller

import (
	"encoding/json"
	"strings"
	"time"

	"just-vpn/middleware"
	"just-vpn/model"

	"github.com/gin-gonic/gin"
)

type PopupResponse struct {
	Id           int      `json:"id" example:"1"`                                    // 弹窗ID
	Title        string   `json:"title" example:"会员活动"`                              // 弹窗标题
	Content      string   `json:"content" example:"限时开通会员享优惠"`                       // 弹窗内容
	ImageUrl     []string `json:"image_url" example:"https://example.com/popup.png"` // 弹窗图片地址数组
	JumpType     string   `json:"jump_type" example:"internal"`                      // 跳转方式(none=不跳转,internal=内部跳转,external=外部浏览器)
	JumpTarget   []string `json:"jump_target" example:"purchase"`                    // 跳转目标数组，内部跳转填业务code，外部跳转填URL
	CanClose     int      `json:"can_close" example:"1"`                             // 是否可关闭(0=不可关闭,1=可关闭)
	ShowTimes    int      `json:"show_times" example:"1"`                            // 当前用户已展示次数
	MaxShowTimes int      `json:"max_show_times" example:"3"`                        // 每个用户最大展示次数，0表示不限次数
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
	popup, record, ok, err := nextPopupForUser(user.Id, user.CreateTime, clientInfo.Platform, clientInfo.Version, c.ClientIP(), now)
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
		Title:        localizedTextValue(popup.Title, clientInfo.Language),
		Content:      localizedTextValue(popup.Content, clientInfo.Language),
		ImageUrl:     popupImageResponseValue(popup.ImageUrl),
		JumpType:     popup.JumpType,
		JumpTarget:   popupImageResponseValue(popup.JumpTarget),
		CanClose:     popup.CanClose,
		ShowTimes:    record.ShowTimes,
		MaxShowTimes: popup.MaxShowTimes,
	})
}

// popupImageResponseValue 兼容后台保存的 JSON 数组、逗号分隔或单个图片地址
func popupImageResponseValue(value string) []string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return []string{}
	}

	var urls []string
	if json.Unmarshal([]byte(trimmed), &urls) == nil {
		return filterPopupImageURLs(urls)
	}
	return filterPopupImageURLs(strings.Split(trimmed, ","))
}

func filterPopupImageURLs(values []string) []string {
	urls := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			urls = append(urls, value)
		}
	}
	return urls
}

func nextPopupForUser(userId int, registerTime time.Time, platform string, version string, requestIP string, now time.Time) (model.Popup, model.UserPopup, bool, error) {
	popups, err := model.GetEnabledPopups(now)
	if err != nil {
		return model.Popup{}, model.UserPopup{}, false, err
	}
	ipRegion := ""
	ipRegionLoaded := false
	for _, popup := range popups {
		if !model.MatchCSVRule(popup.Platforms, platform) {
			continue
		}
		if !model.MatchCSVRule(popup.Versions, version) {
			continue
		}
		if !popup.CanShowToRegisteredUser(registerTime, now) {
			continue
		}
		if popup.OnlyMainland == 1 {
			if !ipRegionLoaded {
				ipRegion = loginIPRegion(requestIP)
				ipRegionLoaded = true
			}
			if !popup.CanShowToIPRegion(ipRegion) {
				continue
			}
		}
		if _, ok, err := model.UserPopupCanShow(userId, popup); err != nil {
			return model.Popup{}, model.UserPopup{}, false, err
		} else if !ok {
			continue
		}
		acquired, err := model.AcquirePopupDailyShow(popup, now)
		if err != nil {
			return model.Popup{}, model.UserPopup{}, false, err
		}
		if !acquired {
			continue
		}
		record, err := model.AddUserPopupShow(userId, popup.Id, now)
		if err != nil {
			_ = model.ReleasePopupDailyShow(popup, now)
			return model.Popup{}, model.UserPopup{}, false, err
		}
		return popup, record, true, nil
	}
	return model.Popup{}, model.UserPopup{}, false, nil
}
