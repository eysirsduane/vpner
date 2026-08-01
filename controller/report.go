package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"just-vpn/middleware"
	"just-vpn/model"
	"just-vpn/pkg/setting"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type errorReportParams struct {
	Msg             string
	MobileVersion   string
	MobileModelName string
	MsgType         string
}

type ErrorReportRequest struct {
	Msg             string `json:"msg" binding:"required" example:"vpn connect timeout"`
	MobileVersion   string `json:"mobile_version" binding:"required" example:"iOS 17.5"`
	MobileModelName string `json:"mobile_model_name" binding:"required" example:"iPhone 15"`
	MsgType         string `json:"msg_type" example:"vpn"`
}

type InternalReportStatsRequest struct {
	Key       string `json:"key" example:"just.303"`
	StartTime string `json:"start_time" example:"2026-07-15 00:00:00"`
	EndTime   string `json:"end_time" example:"2026-07-15 12:00:00"`
}

type InternalReportStatsResult struct {
	NewUsers         int64   `json:"new_users"`
	ConnectBaseUsers int64   `json:"connect_base_users"`
	ConnectedUsers   int64   `json:"connected_users"`
	CreatedOrders    int64   `json:"created_orders"`
	PaidOrders       int64   `json:"paid_orders"`
	TotalIncome      float64 `json:"total_income"`
	AppleIncome      float64 `json:"apple_income"`
	ThirdPartyIncome float64 `json:"third_party_income"`
}

type internalReportResponse struct {
	Code   int                       `json:"code"`
	Msg    string                    `json:"msg"`
	Result InternalReportStatsResult `json:"result"`
}

type InternalMemberLookupRequest struct {
	Id  int    `json:"id"`
	Key string `json:"key"`
}

type InternalMemberLookupResult struct {
	UserID           int                         `json:"user_id"`
	Username         string                      `json:"username"`
	Status           int                         `json:"status"`
	Type             int                         `json:"type"`
	Platform         string                      `json:"platform"`
	Version          string                      `json:"version"`
	VipTimeUnix      int64                       `json:"vip_time_unix"`
	VipTimeText      string                      `json:"vip_time_text"`
	VipExpired       bool                        `json:"vip_expired"`
	VipRemainingText string                      `json:"vip_remaining_text"`
	PayTimes         int                         `json:"pay_times"`
	PayAll           float64                     `json:"pay_all"`
	CreateTimeText   string                      `json:"create_time_text"`
	SuccessfulOrders []InternalMemberOrderResult `json:"successful_orders"`
}

type InternalMemberOrderResult struct {
	ID         int     `json:"id"`
	OrderNo    string  `json:"order_no"`
	PakName    string  `json:"pak_name"`
	Name       string  `json:"name"`
	PayType    string  `json:"pay_type"`
	Money      float64 `json:"money"`
	Price      float64 `json:"price"`
	PayTime    string  `json:"pay_time"`
	CreateTime string  `json:"create_time"`
	Origin     string  `json:"origin"`
	Platform   string  `json:"platform"`
}

type internalMemberLookupResponse struct {
	Code   int                        `json:"code"`
	Msg    string                     `json:"msg"`
	Result InternalMemberLookupResult `json:"result"`
}

func getErrorReportParams(body map[string]interface{}) errorReportParams {
	const originalPath = "/api/v1/error"
	return errorReportParams{
		Msg:             getMappedBodyString(body, originalPath, "msg", ""),
		MobileVersion:   getMappedBodyString(body, originalPath, "mobile_version", ""),
		MobileModelName: getMappedBodyString(body, originalPath, "mobile_model_name", ""),
		MsgType:         getMappedBodyString(body, originalPath, "msg_type", "app"),
	}
}

// ErrorReportHandler 错误信息上报
// @Summary 错误信息上报
// @Description 客户端上报 App 或 VPN 运行错误，msg_type 允许 app 或 vpn，默认 app
// @Tags 上报
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ErrorReportRequest true "错误信息上报请求"
// @Success 200 {object} Response
// @Router /error [post]
func ErrorReportHandler(c *gin.Context) {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}

	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		JsonReturn(c, CodeError, "invalid json body", nil)
		return
	}

	params := getErrorReportParams(body)
	params.MsgType = strings.TrimSpace(params.MsgType)
	if params.MsgType == "" {
		params.MsgType = "app"
	}
	if params.Msg == "" || params.MobileVersion == "" || params.MobileModelName == "" {
		JsonReturn(c, CodeError, "msg, mobile_version and mobile_model_name are required", nil)
		return
	}
	if params.MsgType != "app" && params.MsgType != "vpn" {
		JsonReturn(c, CodeError, "msg_type must be app or vpn", nil)
		return
	}

	if err := model.DB.Create(&model.Error{
		Desc:          params.Msg,
		MsgType:       params.MsgType,
		Uid:           user.Id,
		DeviceType:    params.MobileModelName,
		DeviceVersion: params.MobileVersion,
	}).Error; err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}

	JsonReturn(c, CodeSuccess, "success", nil)
}

func InternalReportStatsHandler(c *gin.Context) {
	var req InternalReportStatsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		internalReportJSON(c, http.StatusBadRequest, CodeError, "invalid json body", InternalReportStatsResult{})
		return
	}
	if strings.TrimSpace(req.Key) == "" {
		req.Key = c.GetHeader("X-Report-Key")
	}
	if req.Key == "" || setting.AppConfig.InternalKey == "" || req.Key != setting.AppConfig.InternalKey {
		internalReportJSON(c, http.StatusForbidden, CodeError, "invalid key", InternalReportStatsResult{})
		return
	}

	start, err := parseInternalReportTime(req.StartTime)
	if err != nil {
		internalReportJSON(c, http.StatusBadRequest, CodeError, "invalid start_time", InternalReportStatsResult{})
		return
	}
	end, err := parseInternalReportTime(req.EndTime)
	if err != nil {
		internalReportJSON(c, http.StatusBadRequest, CodeError, "invalid end_time", InternalReportStatsResult{})
		return
	}
	if !end.After(start) {
		internalReportJSON(c, http.StatusBadRequest, CodeError, "end_time must be after start_time", InternalReportStatsResult{})
		return
	}

	result, err := queryInternalReportStats(start, end)
	if err != nil {
		internalReportJSON(c, http.StatusInternalServerError, CodeError, err.Error(), InternalReportStatsResult{})
		return
	}
	internalReportJSON(c, http.StatusOK, CodeSuccess, "success", result)
}

func InternalMemberLookupHandler(c *gin.Context) {
	var req InternalMemberLookupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		internalMemberLookupJSON(c, http.StatusBadRequest, CodeError, "invalid json body", InternalMemberLookupResult{})
		return
	}
	if strings.TrimSpace(req.Key) == "" {
		req.Key = c.GetHeader("X-Lookup-Key")
	}
	if req.Key == "" || setting.AppConfig.InternalKey == "" || req.Key != setting.AppConfig.InternalKey {
		internalMemberLookupJSON(c, http.StatusForbidden, CodeError, "invalid key", InternalMemberLookupResult{})
		return
	}
	if req.Id <= 0 {
		internalMemberLookupJSON(c, http.StatusBadRequest, CodeError, "invalid id", InternalMemberLookupResult{})
		return
	}

	result, err := queryInternalMember(req.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			internalMemberLookupJSON(c, http.StatusNotFound, CodeError, "user not found", InternalMemberLookupResult{})
			return
		}
		internalMemberLookupJSON(c, http.StatusInternalServerError, CodeError, err.Error(), InternalMemberLookupResult{})
		return
	}
	internalMemberLookupJSON(c, http.StatusOK, CodeSuccess, "success", result)
}

func internalReportJSON(c *gin.Context, httpStatus, code int, msg string, result InternalReportStatsResult) {
	c.JSON(httpStatus, internalReportResponse{
		Code:   code,
		Msg:    msg,
		Result: result,
	})
}

func internalMemberLookupJSON(c *gin.Context, httpStatus, code int, msg string, result InternalMemberLookupResult) {
	c.JSON(httpStatus, internalMemberLookupResponse{
		Code:   code,
		Msg:    msg,
		Result: result,
	})
}

func parseInternalReportTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", value, time.Local); err == nil {
		return t, nil
	}
	return time.ParseInLocation(time.RFC3339, value, time.Local)
}

func queryInternalReportStats(start, end time.Time) (InternalReportStatsResult, error) {
	var result InternalReportStatsResult
	newUserIds, hasNewUserCache, err := dailyNewUserIDsForReport(start, end)
	if err != nil {
		return result, err
	}
	if hasNewUserCache {
		result.NewUsers = int64(len(newUserIds))
		if err := scanConnectStatsByUserIDs(newUserIds, start, end, &result); err != nil {
			return result, err
		}
	} else {
		if err := queryInternalReportNewUsersByDB(start, end, &result); err != nil {
			return result, err
		}
		if err := queryInternalReportConnectStatsByDB(start, end, &result); err != nil {
			return result, err
		}
	}

	if err := queryInternalReportPaymentStats(start, end, &result); err != nil {
		return result, err
	}
	if err := queryInternalReportIncome(start, end, &result); err != nil {
		return result, err
	}
	return result, nil
}

func queryInternalMember(userID int) (InternalMemberLookupResult, error) {
	var user model.User
	if err := model.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		return InternalMemberLookupResult{}, err
	}

	now := time.Now()
	result := InternalMemberLookupResult{
		UserID:           user.Id,
		Username:         user.Username,
		Status:           user.Status,
		Type:             user.Type,
		Platform:         user.Platform,
		Version:          user.Version,
		PayTimes:         user.PayTimes,
		PayAll:           user.PayAll,
		CreateTimeText:   user.CreateTime.In(time.Local).Format("2006-01-02 15:04:05"),
		VipTimeText:      "无",
		VipRemainingText: "无",
		VipExpired:       true,
	}
	if user.VipTime != nil {
		vipTime := user.VipTime.In(time.Local)
		result.VipTimeUnix = vipTime.Unix()
		result.VipTimeText = vipTime.Format("2006-01-02 15:04:05")
		result.VipExpired = !vipTime.After(now)
		result.VipRemainingText = internalVipRemainingText(vipTime, now)
	}

	orders, err := queryInternalMemberOrders(user.Id)
	if err != nil {
		return InternalMemberLookupResult{}, err
	}
	result.SuccessfulOrders = orders
	return result, nil
}

func queryInternalMemberOrders(userID int) ([]InternalMemberOrderResult, error) {
	var orders []model.Order
	if err := model.DB.Where("uid = ?", userID).
		Where("pay_status = ?", model.OrderPayStatusPaid).
		Order("pay_time desc, id desc").
		Limit(200).
		Find(&orders).Error; err != nil {
		return nil, err
	}

	results := make([]InternalMemberOrderResult, 0, len(orders))
	for _, order := range orders {
		item := InternalMemberOrderResult{
			ID:       order.Id,
			OrderNo:  order.OrderNo,
			PakName:  order.PakName,
			Name:     order.Name,
			PayType:  order.PayType,
			Money:    centsToYuan(int64(order.Money)),
			Price:    centsToYuan(int64(order.Price)),
			Origin:   order.Origin,
			Platform: order.RegPlatform,
		}
		if order.PayTime != nil {
			item.PayTime = order.PayTime.In(time.Local).Format("2006-01-02 15:04:05")
		}
		item.CreateTime = order.CreateTime.In(time.Local).Format("2006-01-02 15:04:05")
		results = append(results, item)
	}
	return results, nil
}

func internalVipRemainingText(vipTime, now time.Time) string {
	diff := vipTime.Sub(now)
	if diff >= 0 {
		days := int64(diff.Hours()+23) / 24
		if days < 1 {
			days = 1
		}
		return "剩余：" + strconv.FormatInt(days, 10) + "天"
	}
	days := int64((-diff).Hours()+23) / 24
	if days < 1 {
		days = 1
	}
	return "已过期" + strconv.FormatInt(days, 10) + "天"
}

func dailyNewUserIDsForReport(start, end time.Time) ([]int, bool, error) {
	if !isSingleDailyReportRange(start, end) {
		return nil, false, nil
	}
	return model.DailyNewUserIDsFromCache(start)
}

func isSingleDailyReportRange(start, end time.Time) bool {
	start = start.In(time.Local)
	end = end.In(time.Local)
	startDay := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.Local)
	nextDay := startDay.AddDate(0, 0, 1)
	return start.Equal(startDay) && end.After(start) && !end.After(nextDay)
}

func queryInternalReportNewUsersByDB(start, end time.Time, result *InternalReportStatsResult) error {
	return model.DB.Raw(`
SELECT COUNT(*)
FROM `+"`user`"+`
WHERE (
    create_time >= ?
    AND create_time < ?
    AND last_login_time >= ?
    AND last_login_time < ?
  )
  OR (
    migration_source IS NOT NULL
    AND last_login_time >= ?
    AND last_login_time < ?
  )
`, start, end, start, end, start, end).Scan(&result.NewUsers).Error
}

func queryInternalReportConnectStatsByDB(start, end time.Time, result *InternalReportStatsResult) error {
	return model.DB.Raw(`
SELECT
  COUNT(DISTINCT u.id) AS connect_base_users,
  COUNT(DISTINCT h.user_id) AS connected_users
FROM `+"`user`"+` u
LEFT JOIN node_connect_history h
  ON h.user_id = u.id
  AND h.connected_at >= ?
  AND h.connected_at < ?
WHERE (
    (
      u.create_time >= ?
      AND u.create_time < ?
      AND u.last_login_time >= ?
      AND u.last_login_time < ?
    )
    OR (
      u.migration_source IS NOT NULL
      AND u.last_login_time >= ?
      AND u.last_login_time < ?
    )
  )
  AND u.is_real = 1
`, start, end, start, end, start, end, start, end).Row().Scan(&result.ConnectBaseUsers, &result.ConnectedUsers)
}

func scanConnectStatsByUserIDs(userIds []int, start, end time.Time, result *InternalReportStatsResult) error {
	const batchSize = 1000
	if len(userIds) == 0 {
		return nil
	}
	for i := 0; i < len(userIds); i += batchSize {
		next := i + batchSize
		if next > len(userIds) {
			next = len(userIds)
		}
		var connectBaseUsers, connectedUsers int64
		if err := model.DB.Raw(`
SELECT
  COUNT(DISTINCT u.id) AS connect_base_users,
  COUNT(DISTINCT h.user_id) AS connected_users
FROM `+"`user`"+` u
LEFT JOIN node_connect_history h
  ON h.user_id = u.id
  AND h.connected_at >= ?
  AND h.connected_at < ?
WHERE u.id IN ?
  AND u.is_real = 1
`, start, end, userIds[i:next]).Row().Scan(&connectBaseUsers, &connectedUsers); err != nil {
			return err
		}
		result.ConnectBaseUsers += connectBaseUsers
		result.ConnectedUsers += connectedUsers
	}
	return nil
}

func queryInternalReportPaymentStats(start, end time.Time, result *InternalReportStatsResult) error {
	var createdOrders, paidOrders int64
	if err := model.DB.Raw(`
SELECT
  COUNT(*) AS created_orders,
  COALESCE(SUM(CASE WHEN pay_status = 3 THEN 1 ELSE 0 END), 0) AS paid_orders
FROM `+"`order`"+`
WHERE create_time >= ?
  AND create_time < ?
`, start, end).Row().Scan(&createdOrders, &paidOrders); err != nil {
		return err
	}

	result.CreatedOrders = createdOrders
	result.PaidOrders = paidOrders
	return nil
}

func queryInternalReportIncome(start, end time.Time, result *InternalReportStatsResult) error {
	var totalCents, appleCents, thirdPartyCents int64
	if err := model.DB.Raw(`
SELECT
  COALESCE(SUM(money), 0) AS total_income,
  COALESCE(SUM(CASE WHEN pay_type = 'apple_iap' THEN money ELSE 0 END), 0) AS apple_income,
  COALESCE(SUM(CASE WHEN pay_type <> 'apple_iap' THEN money ELSE 0 END), 0) AS third_party_income
FROM `+"`order`"+`
WHERE pay_status = 3
  AND pay_time >= ?
  AND pay_time < ?
`, start, end).Row().Scan(&totalCents, &appleCents, &thirdPartyCents); err != nil {
		return err
	}

	result.TotalIncome = centsToYuan(totalCents)
	result.AppleIncome = centsToYuan(appleCents)
	result.ThirdPartyIncome = centsToYuan(thirdPartyCents)
	return nil
}

func centsToYuan(cents int64) float64 {
	return float64(cents) / 100
}
