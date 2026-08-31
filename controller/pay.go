package controller

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"just-vpn/middleware"
	"just-vpn/model"
	"just-vpn/pkg/pay/xxpay"
	"just-vpn/pkg/util"

	"github.com/gin-gonic/gin"
	"github.com/go-pay/gopay/apple"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	PayLaunchTypeAppleIAP = "apple_iap"
	PayLaunchTypeH5       = "h5"
	ThirdPayProviderXXPay = "xxpay"

	appleVerifyModeMock       = "mock"
	appleVerifyModeProduction = "production"
	appleVerifyModeSandbox    = "sandbox"
)

type PayLaunchRequest struct {
	PackageId int `json:"package_id" binding:"required" example:"1"` // 套餐ID
}

type PayLaunchResponse struct {
	Type            string `json:"type" example:"apple_iap"`                                         // 支付方式(apple_iap=苹果内购,h5=H5支付)
	Target          string `json:"target" example:"vip_month"`                                       // 支付目标，内购返回apple_id，H5返回支付页面地址
	AppAccountToken string `json:"app_account_token" example:"C56A4180-65AA-42EC-A945-5FD21DEC0538"` // 苹果内购订单标识，内购时前端作为appAccountToken传给StoreKit
}

type AppleVerifyRequest struct {
	TransactionId string `json:"transaction_id" binding:"required" example:"1000000000000000"` // 苹果交易号
}

type AppleVerifyResponse struct {
	OrderNo       string `json:"order_no" example:"20260707120000123456"`   // 订单号
	PayStatus     int    `json:"pay_status" example:"3"`                    // 支付状态(1=未支付,3=已支付)
	PayType       string `json:"pay_type" example:"apple_iap"`              // 支付方式
	TransactionId string `json:"transaction_id" example:"1000000000000000"` // 苹果交易号
	VipTime       string `json:"vip_time" example:"2026-08-06 12:00:00"`    // 会员到期时间
}

type AppleCallbackRequest struct {
	SignedPayload string `json:"signedPayload" binding:"required" example:"eyJhbGciOiJFUzI1NiIsIng1YyI6Wy..."`
}

type payLaunchDecision struct {
	payType string
	target  string
	reason  string
}

type payContext struct {
	c        *gin.Context
	user     model.User
	pack     model.Package
	now      time.Time
	regions  string
	version  string
	timeZone string
}

// PayLaunchHandler 发起支付
// @Summary 发起支付
// @Description 非中国大陆时区（含未传时区）直接返回苹果内购；中国大陆时区继续根据用户地区、客户端版本和每日动态内购限额判断返回苹果内购或H5支付，并创建未支付订单；内购返回套餐 apple_id 和订单级 app_account_token，前端支付时必须作为 StoreKit appAccountToken 携带
// @Tags 支付
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body PayLaunchRequest true "发起支付请求"
// @Success 200 {object} Response{result=PayLaunchResponse}
// @Router /pay/launch [post]
func PayLaunchHandler(c *gin.Context) {
	var req PayLaunchRequest
	if err := BindMappedJSON(c, &req); err != nil {
		if strings.Contains(err.Error(), "'PackageId' failed on the 'required' tag") {
			JsonReturn(c, CodeError, "package_id is required", nil)
			return
		}
		JsonReturn(c, CodeError, "invalid json body", nil)
		return
	}
	if req.PackageId <= 0 {
		JsonReturn(c, CodeError, "package_id is required", nil)
		return
	}

	user, err := middleware.CurrentUser(c)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	pack, err := model.GetEnabledPackageByID(req.PackageId)
	if err != nil {
		JsonReturn(c, CodeError, "package not found", nil)
		return
	}
	if strings.TrimSpace(pack.AppleId) == "" {
		JsonReturn(c, CodeError, "package apple_id is required", nil)
		return
	}
	requestIP := c.ClientIP()
	requestIPRegion := loginIPRegion(requestIP)
	clientInfo := middleware.CurrentClientInfo(c)

	decision, err := decidePayLaunch(payContext{
		c:        c,
		user:     user,
		pack:     pack,
		now:      time.Now().In(time.Local),
		regions:  requestIPRegion,
		version:  clientInfo.Version,
		timeZone: clientInfo.TimeZone,
	})
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	order := buildLaunchOrder(user, pack, decision, requestIP, requestIPRegion, clientInfo)
	if decision.payType == PayLaunchTypeAppleIAP {
		order.AppAccountToken = newAppleAccountToken()
	} else if decision.payType == PayLaunchTypeH5 {
		order.PayType = model.OrderPayTypeXXPay
		order.PayProductId = strings.TrimSpace(model.PayConfigValue(model.PayConfigXXPayAlipayID, ""))
	}
	if err := model.CreateOrder(&order); err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	if decision.payType == PayLaunchTypeH5 {
		payInfo, err := createXXPayOrder(c, user, pack, order)
		if err != nil {
			JsonReturn(c, CodeError, err.Error(), nil)
			return
		}
		if err := model.DB.Model(&model.Order{}).Where("id = ?", order.Id).Updates(map[string]interface{}{
			"pay_sn":  payInfo.PayOrderId,
			"pay_url": payInfo.PayURL,
		}).Error; err != nil {
			JsonReturn(c, CodeError, err.Error(), nil)
			return
		}
		decision.target = payInfo.PayURL
	}

	JsonReturn(c, CodeSuccess, "success", PayLaunchResponse{
		Type:            decision.payType,
		Target:          decision.target,
		AppAccountToken: order.AppAccountToken,
	})
}

// AppleVerifyHandler 苹果订单验证
// @Summary 苹果订单验证
// @Description 客户端完成苹果内购后提交苹果交易号，后端通过苹果交易信息中的 productId 和 appAccountToken 匹配发起支付时创建的订单，验证成功后更新订单并发放会员权益
// @Tags 支付
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body AppleVerifyRequest true "苹果订单验证请求"
// @Success 200 {object} Response{result=AppleVerifyResponse}
// @Router /pay/apple_verify [post]
func AppleVerifyHandler(c *gin.Context) {
	var req AppleVerifyRequest
	if err := BindMappedJSON(c, &req); err != nil {
		if strings.Contains(err.Error(), "'TransactionId' failed on the 'required' tag") {
			JsonReturn(c, CodeError, "transaction_id is required", nil)
			return
		}
		JsonReturn(c, CodeError, "invalid json body", nil)
		return
	}
	req.TransactionId = strings.TrimSpace(req.TransactionId)
	if req.TransactionId == "" {
		JsonReturn(c, CodeError, "transaction_id is required", nil)
		return
	}

	transaction, err := verifyAppleTransaction(req.TransactionId)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	if strings.TrimSpace(transaction.ProductId) == "" {
		JsonReturn(c, CodeError, "apple product id is required", nil)
		return
	}

	pack, err := model.GetEnabledPackageByAppleId(transaction.ProductId)
	if err != nil {
		JsonReturn(c, CodeError, "package not found", nil)
		return
	}
	user, err := middleware.CurrentUser(c)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}

	var orderNo string
	var vipTime *time.Time
	if err := model.DB.Transaction(func(tx *gorm.DB) error {
		return completeAppleOrder(tx, c, user.Id, pack, transaction, model.OrderPaySourceAppleVerify, &orderNo, &vipTime)
	}); err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}

	JsonReturn(c, CodeSuccess, "success", AppleVerifyResponse{
		OrderNo:       orderNo,
		PayStatus:     model.OrderPayStatusPaid,
		PayType:       model.OrderPayTypeAppleIAP,
		TransactionId: req.TransactionId,
		VipTime:       formatVipTime(vipTime),
	})
}

// AppleCallbackHandler 苹果支付服务端通知
// @Summary 苹果支付服务端通知
// @Description 接收 App Store Server Notifications V2，用于苹果内购补单、退款和撤销处理；接口固定返回 HTTP 200，业务结果写入 apple_notification 表
// @Tags 支付
// @Accept json
// @Produce json
// @Param request body AppleCallbackRequest true "苹果服务端通知请求"
// @Success 200 {string} string "OK"
// @Router /pay/apple_callback [post]
func AppleCallbackHandler(c *gin.Context) {
	var req AppleCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.SignedPayload) == "" {
		log.Printf("apple s2s invalid body err=%v", err)
		c.Status(http.StatusOK)
		return
	}

	payload, err := apple.DecodeSignedPayload(req.SignedPayload)
	if err != nil {
		log.Printf("apple s2s decode payload error=%v", err)
		c.Status(http.StatusOK)
		return
	}
	transaction, err := payload.DecodeTransactionInfo()
	if err != nil {
		log.Printf("apple s2s decode transaction error uuid=%s type=%s err=%v", payload.NotificationUUID, payload.NotificationType, err)
		_ = saveAppleNotificationLog(payload, nil, req.SignedPayload, model.AppleNotificationStatusIgnored, err.Error())
		c.Status(http.StatusOK)
		return
	}
	if err := validateAppleNotification(payload, transaction); err != nil {
		log.Printf("apple s2s validate error uuid=%s transaction_id=%s err=%v", payload.NotificationUUID, transaction.TransactionId, err)
		_ = saveAppleNotificationLog(payload, transaction, req.SignedPayload, model.AppleNotificationStatusIgnored, err.Error())
		c.Status(http.StatusOK)
		return
	}
	if appleNotificationAlreadyHandled(payload.NotificationUUID) {
		c.Status(http.StatusOK)
		return
	}

	status, msg := processAppleNotification(c, payload, transaction)
	if err := saveAppleNotificationLog(payload, transaction, req.SignedPayload, status, msg); err != nil {
		log.Printf("apple s2s save log error uuid=%s err=%v", payload.NotificationUUID, err)
	}
	c.Status(http.StatusOK)
}

// XXPayCallbackHandler XX支付回调
// @Summary XX支付回调
// @Description 接收 XX 支付异步回调，验签、验金额并发放会员权益；处理成功返回纯字符串 success
// @Tags 支付
// @Accept x-www-form-urlencoded
// @Produce plain
// @Success 200 {string} string "success"
// @Router /pay/xx_callback [post]
func XXPayCallbackHandler(c *gin.Context) {
	params := collectXXPayNotifyParams(c)
	if !isXXPayCallbackIPAllowed(c.ClientIP()) {
		log.Printf("xxpay callback ip rejected ip=%s", c.ClientIP())
		saveThirdPayCallbackLog(ThirdPayProviderXXPay, params, c.ClientIP(), 0, model.ThirdPayCallbackStatusIgnored, "callback ip rejected")
		c.String(http.StatusOK, "")
		return
	}
	key := strings.TrimSpace(model.PayConfigValue(model.PayConfigXXPayKey, ""))
	signValid := key != "" && xxpay.VerifySign(params, key)
	if !signValid {
		log.Printf("xxpay callback sign error params=%v", params)
		saveThirdPayCallbackLog(ThirdPayProviderXXPay, params, c.ClientIP(), 0, model.ThirdPayCallbackStatusFailed, "sign error")
		c.String(http.StatusOK, "")
		return
	}
	if !xxpay.IsPaidStatus(params["status"]) {
		log.Printf("xxpay callback unpaid status=%s order_no=%s", params["status"], params["mchOrderNo"])
		saveThirdPayCallbackLog(ThirdPayProviderXXPay, params, c.ClientIP(), 1, model.ThirdPayCallbackStatusIgnored, "unpaid status:"+params["status"])
		c.String(http.StatusOK, "")
		return
	}
	if err := completeXXPayOrder(params); err != nil {
		log.Printf("xxpay callback process error order_no=%s err=%v", params["mchOrderNo"], err)
		saveThirdPayCallbackLog(ThirdPayProviderXXPay, params, c.ClientIP(), 1, model.ThirdPayCallbackStatusFailed, err.Error())
		c.String(http.StatusOK, "")
		return
	}
	saveThirdPayCallbackLog(ThirdPayProviderXXPay, params, c.ClientIP(), 1, model.ThirdPayCallbackStatusDone, "success")
	c.String(http.StatusOK, "success")
}

func decidePayLaunch(ctx payContext) (payLaunchDecision, error) {
	if matched, reason := shouldForceAppleByTimeZone(ctx); matched {
		return appleIAPDecision(ctx.pack, reason), nil
	}
	if matched, reason := shouldForceAppleByOverseas(ctx); matched {
		return appleIAPDecision(ctx.pack, reason), nil
	}
	if matched, reason, err := shouldDirectH5ByPaidThirdPay(ctx); err != nil {
		log.Printf("pay launch paid third rule error uid=%d err=%v", ctx.user.Id, err)
		return appleIAPDecision(ctx.pack, reason), nil
	} else if matched {
		return h5PayDecision(reason), nil
	}
	if matched, reason := shouldForceAppleByRegion(ctx); matched {
		return appleIAPDecision(ctx.pack, reason), nil
	}
	if matched, reason := shouldForceAppleByVersion(ctx); matched {
		return appleIAPDecision(ctx.pack, reason), nil
	}
	if matched, reason, err := shouldForceAppleByTodayAmount(ctx); err != nil {
		log.Printf("pay launch amount rule error uid=%d err=%v", ctx.user.Id, err)
		return appleIAPDecision(ctx.pack, reason), nil
	} else if matched {
		return appleIAPDecision(ctx.pack, reason), nil
	}
	return h5PayDecision("allowed"), nil
}

// shouldForceAppleByTimeZone 判断客户端是否明确处于中国大陆时区。
// 未传时区时无法确认用户位于中国大陆，因此按非大陆时区处理。
func shouldForceAppleByTimeZone(ctx payContext) (bool, string) {
	if strings.TrimSpace(payTimeZoneConfigValue(model.PayConfigOverseasTimeZoneAppleOnly, "1")) != "1" {
		return false, "overseas_timezone_apple_only_disabled"
	}
	timeZone := strings.ToLower(strings.TrimSpace(ctx.timeZone))
	switch timeZone {
	case "asia/shanghai",
		"asia/chongqing",
		"asia/chungking",
		"asia/harbin",
		"asia/urumqi",
		"asia/kashgar",
		"prc":
		return false, "mainland_china_timezone"
	default:
		return true, "non_mainland_timezone"
	}
}

var payTimeZoneConfigValue = model.PayConfigValue

// shouldForceAppleByOverseas 判断非中国 IP 是否仅允许苹果内购
func shouldForceAppleByOverseas(ctx payContext) (bool, string) {
	if strings.TrimSpace(payOverseasIPConfigValue(model.PayConfigOverseasAppleOnly, "1")) != "1" {
		return false, "overseas_apple_only_disabled"
	}
	country := ipRegionCountry(ctx.regions)
	if country == "" || country == "0" || country == "中国" {
		return false, "overseas_region_not_matched"
	}
	return true, "overseas_region:" + country
}

var payOverseasIPConfigValue = model.PayConfigValue

func ipRegionCountry(region string) string {
	parts := strings.Split(strings.TrimSpace(region), "|")
	if len(parts) >= 5 {
		return strings.TrimSpace(parts[1])
	}
	if len(parts) > 0 {
		return strings.TrimSpace(parts[0])
	}
	return ""
}

// shouldDirectH5ByPaidThirdPay 判断已成功三方支付用户是否直接继续走三方支付
func shouldDirectH5ByPaidThirdPay(ctx payContext) (bool, string, error) {
	if strings.TrimSpace(model.PayConfigValue(model.PayConfigPaidThirdDirectH5, "0")) != "1" {
		return false, "paid_third_direct_h5_disabled", nil
	}
	paid, err := model.UserHasPaidThirdPayOrder(ctx.user.Id)
	if err != nil {
		return false, "paid_third_query_error", err
	}
	if paid {
		return true, "paid_third_direct_h5", nil
	}
	return false, "paid_third_not_found", nil
}

// shouldForceAppleByRegion 判断用户地区是否命中仅内购地区配置
func shouldForceAppleByRegion(ctx payContext) (bool, string) {
	regions := splitCommaConfig(model.PayConfigValue(model.PayConfigOnlyAppleRegions, ""))
	if len(regions) == 0 {
		return false, "only_apple_region_empty"
	}
	userRegions := strings.ToLower(ctx.regions)
	for _, region := range regions {
		if strings.Contains(userRegions, strings.ToLower(region)) {
			return true, "only_apple_region:" + region
		}
	}
	return false, "only_apple_region_not_matched"
}

// shouldForceAppleByVersion 判断请求头版本是否命中仅内购版本配置
func shouldForceAppleByVersion(ctx payContext) (bool, string) {
	versions := splitCommaConfig(model.PayConfigValue(model.PayConfigOnlyAppleVersions, ""))
	if len(versions) == 0 {
		return false, "only_apple_version_empty"
	}
	clientVersion := strings.TrimSpace(ctx.version)
	if clientVersion == "" {
		return false, "only_apple_version_missing"
	}
	for _, version := range versions {
		if clientVersion == version {
			return true, "only_apple_version:" + version
		}
	}
	return false, "only_apple_version_not_matched"
}

// shouldForceAppleByTodayAmount 判断今日苹果内购收款是否未达到当天动态限额
func shouldForceAppleByTodayAmount(ctx payContext) (bool, string, error) {
	threshold, err := payTodayH5AppleLimit(ctx.now)
	if err != nil {
		return true, "daily_h5_apple_limit_query_error", err
	}
	if threshold <= 0 {
		return false, "h5_threshold_disabled", nil
	}
	total, err := payTodayApplePaidAmount(ctx.now)
	if err != nil {
		return true, "today_apple_amount_query_error", err
	}
	if int64(total) < threshold {
		return true, fmt.Sprintf("today_apple_amount_not_reached:%d/%d", total, threshold), nil
	}
	return false, fmt.Sprintf("today_apple_amount_reached:%d/%d", total, threshold), nil
}

var payTodayH5AppleLimit = model.TodayH5AppleLimit
var payTodayApplePaidAmount = model.TodayApplePaidAmount

func appleIAPDecision(pack model.Package, reason string) payLaunchDecision {
	return payLaunchDecision{
		payType: PayLaunchTypeAppleIAP,
		target:  pack.AppleId,
		reason:  reason,
	}
}

func h5PayDecision(reason string) payLaunchDecision {
	return payLaunchDecision{
		payType: PayLaunchTypeH5,
		target:  model.PayConfigValue(model.PayConfigH5Target, "https://www.baidu.com"),
		reason:  reason,
	}
}

func splitCommaConfig(value string) []string {
	items := strings.Split(value, ",")
	result := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		result = append(result, item)
	}
	return result
}

func buildLaunchOrder(user model.User, pack model.Package, decision payLaunchDecision, requestIP string, requestIPRegion string, clientInfo middleware.ClientInfo) model.Order {
	orderFlag := 1
	if isValidVipTime(user.VipTime) {
		orderFlag = 2
	}
	return model.Order{
		PakId:          pack.Id,
		PakName:        pack.Name,
		PakTime:        pack.Day * 24 * 60 * 60,
		OrderNo:        util.GetOrderId(),
		Name:           pack.Name,
		PayStatus:      model.OrderPayStatusUnpaid,
		PayType:        decision.payType,
		PayReason:      decision.reason,
		ClientTimeZone: clientInfo.TimeZone,
		PayProductId:   pack.AppleId,
		Uid:            user.Id,
		RegPlatform:    user.Platform,
		RegTime:        user.CreateTime,
		Price:          pack.Price,
		Money:          pack.Price,
		Origin:         clientInfo.Platform,
		Ip:             requestIP,
		OrderFlag:      orderFlag,
		PaySource:      model.OrderPaySourceLaunch,
		SourceChannel:  user.SourceChannel,
		RegIpRegion:    requestIPRegion,
	}
}

func createXXPayOrder(c *gin.Context, user model.User, pack model.Package, order model.Order) (xxpay.PayInfo, error) {
	packageName := localizedTextValue(pack.Name, middleware.CurrentClientInfo(c).Language)
	req := xxpay.CreateOrderRequest{
		APIURL:     strings.TrimSpace(model.PayConfigValue(model.PayConfigXXPayAPIURL, "")),
		MchId:      strings.TrimSpace(model.PayConfigValue(model.PayConfigXXPayMchId, "")),
		AppId:      strings.TrimSpace(model.PayConfigValue(model.PayConfigXXPayAppId, "")),
		ProductId:  strings.TrimSpace(model.PayConfigValue(model.PayConfigXXPayAlipayID, "")),
		MchOrderNo: order.OrderNo,
		Amount:     order.Money,
		ClientIP:   c.ClientIP(),
		Device:     xxpayDevice(middleware.CurrentClientInfo(c).Platform),
		NotifyURL:  strings.TrimSpace(model.PayConfigValue(model.PayConfigXXPayNotifyURL, "")),
		ReturnURL:  strings.TrimSpace(model.PayConfigValue(model.PayConfigXXPayReturnURL, "")),
		Subject:    packageName,
		Body:       fmt.Sprintf("%s-%s", packageName, order.OrderNo),
		Key:        strings.TrimSpace(model.PayConfigValue(model.PayConfigXXPayKey, "")),
	}
	if req.ProductId == "" {
		return xxpay.PayInfo{}, fmt.Errorf("xxpay alipay_product_id is required")
	}
	payInfo, err := xxpay.CreateOrder(req)
	if err != nil {
		return xxpay.PayInfo{}, err
	}
	log.Printf("xxpay order created uid=%d order_no=%s pay_order_id=%s method=%s", user.Id, order.OrderNo, payInfo.PayOrderId, payInfo.PayMethod)
	return payInfo, nil
}

func xxpayDevice(platform string) string {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case "iphone", "ios", "ipad":
		return "iOS"
	case "android":
		return "Android"
	default:
		return "WEB"
	}
}

func collectXXPayNotifyParams(c *gin.Context) map[string]string {
	params := make(map[string]string)
	for k, values := range c.Request.URL.Query() {
		if len(values) > 0 && strings.TrimSpace(values[0]) != "" {
			params[k] = strings.TrimSpace(values[0])
		}
	}
	if c.Request.Method == http.MethodPost {
		if err := c.Request.ParseForm(); err == nil {
			for k, values := range c.Request.PostForm {
				if _, exists := params[k]; !exists && len(values) > 0 && strings.TrimSpace(values[0]) != "" {
					params[k] = strings.TrimSpace(values[0])
				}
			}
		}
	}
	return params
}

func isXXPayCallbackIPAllowed(ip string) bool {
	whitelist := strings.TrimSpace(model.PayConfigValue(model.PayConfigXXPayCallbackIPs, ""))
	if whitelist == "" {
		return true
	}
	for _, item := range splitCommaConfig(whitelist) {
		if item == ip {
			return true
		}
	}
	return false
}

func saveThirdPayCallbackLog(provider string, params map[string]string, clientIP string, signValid int, status int, msg string) {
	rawParams, err := json.Marshal(params)
	if err != nil {
		rawParams = []byte("{}")
	}
	callback := model.ThirdPayCallback{
		BaseModel: model.BaseModel{
			CreateTime: chinaNow(),
		},
		Provider:       provider,
		PayOrderId:     params["payOrderId"],
		MchOrderNo:     params["mchOrderNo"],
		MchId:          params["mchId"],
		ProductId:      params["productId"],
		Amount:         parseCallbackInt(params["amount"]),
		Income:         parseCallbackInt(params["income"]),
		Status:         params["status"],
		ChannelOrderNo: params["channelOrderNo"],
		PaySuccTime:    params["paySuccTime"],
		BackType:       params["backType"],
		ReqTime:        params["reqTime"],
		ClientIp:       clientIP,
		Sign:           params["sign"],
		SignValid:      signValid,
		ProcessStatus:  status,
		ProcessMsg:     truncateString(msg, 255),
		RawParams:      string(rawParams),
	}
	if err := model.DB.Create(&callback).Error; err != nil {
		log.Printf("third pay callback log save error provider=%s order_no=%s err=%v", provider, params["mchOrderNo"], err)
	}
}

func parseCallbackInt(value string) int {
	result, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0
	}
	return result
}

func completeXXPayOrder(params map[string]string) error {
	orderNo := strings.TrimSpace(params["mchOrderNo"])
	if orderNo == "" {
		return fmt.Errorf("mchOrderNo is required")
	}
	callbackAmount, err := strconv.Atoi(strings.TrimSpace(params["amount"]))
	if err != nil || callbackAmount <= 0 {
		return fmt.Errorf("amount invalid")
	}
	paySn := strings.TrimSpace(params["payOrderId"])
	if paySn == "" {
		paySn = strings.TrimSpace(params["channelOrderNo"])
	}
	now := chinaNow()
	return model.DB.Transaction(func(tx *gorm.DB) error {
		var order model.Order
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("order_no = ?", orderNo).
			First(&order).Error; err != nil {
			return err
		}
		if order.PayType != model.OrderPayTypeXXPay && order.PayType != model.OrderPayTypeH5 {
			return fmt.Errorf("order pay type mismatch")
		}
		if order.PayStatus == model.OrderPayStatusPaid {
			return nil
		}
		if order.PayStatus == model.OrderPayStatusRefund {
			return fmt.Errorf("order already refunded")
		}
		if callbackAmount != order.Money {
			return fmt.Errorf("amount mismatch callback=%d expected=%d", callbackAmount, order.Money)
		}

		var user model.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", order.Uid).
			Where("status = ?", 1).
			First(&user).Error; err != nil {
			return err
		}
		vipTime := addVipSeconds(user.VipTime, order.PakTime)
		if err := tx.Model(&model.Order{}).Where("id = ?", order.Id).Updates(map[string]interface{}{
			"pay_status":  model.OrderPayStatusPaid,
			"money":       callbackAmount,
			"pay_sn":      paySn,
			"pay_source":  model.OrderPaySourceXXCallback,
			"verify_time": &now,
			"pay_time":    &now,
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.User{}).Where("id = ?", user.Id).Updates(map[string]interface{}{
			"vip_time":  vipTime,
			"is_pay":    1,
			"pay_times": gorm.Expr("pay_times + ?", 1),
			"pay_all":   gorm.Expr("pay_all + ?", float64(callbackAmount)/100),
		}).Error; err != nil {
			return err
		}
		if user.Username != "" && user.Type == 2 {
			if err := tx.Model(&model.UserAccount{}).Where("username = ?", user.Username).Update("vip_time", vipTime).Error; err != nil {
				return err
			}
			if err := syncAccountVipTime(tx, user.Username, vipTime); err != nil {
				return err
			}
		}
		return tx.Create(&model.UserNotice{
			UserId: user.Id,
			Title:  multilingualTextValue("会员开通成功", "Membership activated"),
			Content: multilingualTextValue(
				fmt.Sprintf("您已成功开通%s，会员有效期至%s", localizedTextValue(order.PakName, simplifiedChineseLanguage), formatVipTime(vipTime)),
				fmt.Sprintf("Your %s is active until %s.", localizedTextValue(order.PakName, "en"), formatVipTime(vipTime)),
			),
			Priority: 100,
			Status:   model.UserNoticeStatusUnread,
		}).Error
	})
}

func verifyAppleTransaction(transactionId string) (appleTransactionInfo, error) {
	mode := strings.ToLower(strings.TrimSpace(model.PayConfigValue(model.PayConfigAppleVerifyMode, appleVerifyModeProduction)))
	if mode == "" || mode == appleVerifyModeMock {
		return appleTransactionInfo{
			TransactionId:   transactionId,
			ProductId:       strings.TrimSpace(model.PayConfigValue(model.PayConfigAppleMockProduct, "")),
			BundleId:        strings.TrimSpace(model.PayConfigValue(model.PayConfigAppleBundleId, "")),
			AppAccountToken: transactionId,
		}, nil
	}
	transaction, err := gopayAppleTransactionInfo(mode, transactionId)
	if err != nil {
		return appleTransactionInfo{}, err
	}
	if transaction.TransactionId != "" && transaction.TransactionId != transactionId {
		return appleTransactionInfo{}, fmt.Errorf("apple transaction id mismatch")
	}
	bundleId := strings.TrimSpace(model.PayConfigValue(model.PayConfigAppleBundleId, ""))
	if bundleId != "" && transaction.BundleId != "" && transaction.BundleId != bundleId {
		return appleTransactionInfo{}, fmt.Errorf("apple bundle id mismatch")
	}
	if transaction.RevocationDate > 0 {
		return appleTransactionInfo{}, fmt.Errorf("apple transaction revoked")
	}
	return transaction, nil
}

type appleTransactionInfo struct {
	TransactionId   string `json:"transactionId"`
	OriginalId      string `json:"originalTransactionId"`
	ProductId       string `json:"productId"`
	BundleId        string `json:"bundleId"`
	AppAccountToken string `json:"appAccountToken"`
	PurchaseDate    int64  `json:"purchaseDate"`
	RevocationDate  int64  `json:"revocationDate"`
}

func gopayAppleTransactionInfo(mode string, transactionId string) (appleTransactionInfo, error) {
	issuerId := strings.TrimSpace(model.PayConfigValue(model.PayConfigAppleIssuerId, ""))
	keyId := strings.TrimSpace(model.PayConfigValue(model.PayConfigAppleKeyId, ""))
	privateKey := strings.TrimSpace(model.PayConfigValue(model.PayConfigApplePrivateKey, ""))
	bundleId := strings.TrimSpace(model.PayConfigValue(model.PayConfigAppleBundleId, ""))
	if issuerId == "" || keyId == "" || privateKey == "" || bundleId == "" {
		return appleTransactionInfo{}, fmt.Errorf("apple server api config is required")
	}
	privateKey = strings.ReplaceAll(privateKey, `\n`, "\n")
	client, err := apple.NewClient(issuerId, bundleId, keyId, privateKey, mode == appleVerifyModeProduction)
	if err != nil {
		return appleTransactionInfo{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	rsp, err := client.GetTransactionInfo(ctx, transactionId)
	if err != nil {
		return appleTransactionInfo{}, err
	}
	transaction, err := rsp.DecodeSignedTransaction()
	if err != nil {
		return appleTransactionInfo{}, err
	}
	return appleTransactionInfo{
		TransactionId:   transaction.TransactionId,
		OriginalId:      transaction.OriginalTransactionId,
		ProductId:       transaction.ProductId,
		BundleId:        transaction.BundleId,
		AppAccountToken: transaction.AppAccountToken,
		PurchaseDate:    transaction.PurchaseDate,
		RevocationDate:  transaction.RevocationDate,
	}, nil
}

func completeAppleOrder(tx *gorm.DB, c *gin.Context, userId int, pack model.Package, transaction appleTransactionInfo, paySource string, orderNoOut *string, vipTimeOut **time.Time) error {
	if strings.TrimSpace(transaction.AppAccountToken) == "" {
		return fmt.Errorf("app_account_token is required")
	}
	releaseLock, err := lockAppleTransaction(tx, transaction.TransactionId)
	if err != nil {
		return err
	}
	defer releaseLock()

	var user model.User
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", userId).First(&user).Error; err != nil {
		return err
	}

	var existingByTransaction model.Order
	err = tx.Where("pay_sn = ?", transaction.TransactionId).
		Where("pay_status IN ?", []int{model.OrderPayStatusPaid, model.OrderPayStatusRefund}).
		First(&existingByTransaction).Error
	if err == nil {
		if existingByTransaction.Uid != user.Id {
			return fmt.Errorf("transaction_id already used")
		}
		if existingByTransaction.PayStatus == model.OrderPayStatusRefund {
			return fmt.Errorf("transaction_id already refunded")
		}
		*orderNoOut = existingByTransaction.OrderNo
		*vipTimeOut = user.VipTime
		return nil
	}
	if !model.IsOrderNotFound(err) {
		return err
	}

	var order model.Order
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("pay_type = ?", model.OrderPayTypeAppleIAP).
		Where("app_account_token = ?", transaction.AppAccountToken).
		First(&order).Error; err != nil {
		if model.IsOrderNotFound(err) {
			return fmt.Errorf("order not found by app_account_token")
		}
		return err
	}
	if order.Uid != user.Id {
		return fmt.Errorf("app_account_token user mismatch")
	}
	if order.PakId != pack.Id || order.PayProductId != pack.AppleId {
		return fmt.Errorf("apple product mismatch")
	}
	if order.PayStatus == model.OrderPayStatusRefund {
		return fmt.Errorf("order already refunded")
	}
	if order.PayStatus == model.OrderPayStatusPaid {
		*orderNoOut = order.OrderNo
		*vipTimeOut = user.VipTime
		return nil
	}

	now := chinaNow()
	vipTime := addVipSeconds(user.VipTime, pack.Day*24*60*60)
	updates := map[string]interface{}{
		"pay_status":  model.OrderPayStatusPaid,
		"pay_sn":      transaction.TransactionId,
		"pay_source":  paySource,
		"verify_time": &now,
		"pay_time":    &now,
	}
	if err := tx.Model(&model.Order{}).Where("id = ?", order.Id).Updates(updates).Error; err != nil {
		return err
	}

	if err := tx.Model(&model.User{}).Where("id = ?", user.Id).Updates(map[string]interface{}{
		"vip_time":  vipTime,
		"is_pay":    1,
		"pay_times": gorm.Expr("pay_times + ?", 1),
		"pay_all":   gorm.Expr("pay_all + ?", float64(pack.Price)/100),
	}).Error; err != nil {
		return err
	}
	if user.Username != "" && user.Type == 2 {
		if err := tx.Model(&model.UserAccount{}).Where("username = ?", user.Username).Update("vip_time", vipTime).Error; err != nil {
			return err
		}
		if err := syncAccountVipTime(tx, user.Username, vipTime); err != nil {
			return err
		}
	}
	if err := tx.Create(&model.UserNotice{
		UserId: user.Id,
		Title:  multilingualTextValue("会员开通成功", "Membership activated"),
		Content: multilingualTextValue(
			fmt.Sprintf("您已成功开通%s，会员有效期至%s", localizedTextValue(pack.Name, simplifiedChineseLanguage), formatVipTime(vipTime)),
			fmt.Sprintf("Your %s is active until %s.", localizedTextValue(pack.Name, "en"), formatVipTime(vipTime)),
		),
		Priority: 100,
		Status:   model.UserNoticeStatusUnread,
	}).Error; err != nil {
		return err
	}
	*orderNoOut = order.OrderNo
	*vipTimeOut = vipTime
	return nil
}

func processAppleNotification(c *gin.Context, payload *apple.NotificationV2Payload, transaction *apple.TransactionInfo) (int, string) {
	if payload == nil || transaction == nil {
		return model.AppleNotificationStatusIgnored, "empty notification"
	}
	switch payload.NotificationType {
	case apple.NotificationTypeV2Refund, apple.NotificationTypeV2Revoke:
		if err := processAppleRefundNotification(transaction); err != nil {
			return model.AppleNotificationStatusFailed, err.Error()
		}
		return model.AppleNotificationStatusDone, "refund processed"
	case apple.NotificationTypeV2Subscribed, apple.NotificationTypeV2DidRenew, apple.NotificationTypeV2OfferRedeemed, apple.NotificationTypeV2OneTimeCharge:
		if err := processApplePaidNotification(c, transaction); err != nil {
			if strings.Contains(err.Error(), "order not found") {
				return model.AppleNotificationStatusPending, err.Error()
			}
			return model.AppleNotificationStatusFailed, err.Error()
		}
		return model.AppleNotificationStatusDone, "payment processed"
	default:
		return model.AppleNotificationStatusIgnored, "ignored notification type:" + payload.NotificationType
	}
}

func processApplePaidNotification(c *gin.Context, transaction *apple.TransactionInfo) error {
	if transaction.RevocationDate > 0 {
		return fmt.Errorf("transaction revoked")
	}
	if order, err := model.GetAppleOrderByTransactionId(transaction.TransactionId); err == nil && order.Id > 0 {
		return nil
	} else if err != nil && !model.IsOrderNotFound(err) {
		return err
	}
	if strings.TrimSpace(transaction.AppAccountToken) == "" {
		return fmt.Errorf("order not found by empty app_account_token")
	}
	order, err := model.GetAppleOrderByAppAccountToken(transaction.AppAccountToken)
	if err != nil {
		if model.IsOrderNotFound(err) {
			return fmt.Errorf("order not found by app_account_token")
		}
		return err
	}
	pack, err := model.GetEnabledPackageByAppleId(transaction.ProductId)
	if err != nil {
		return fmt.Errorf("package not found")
	}
	var orderNo string
	var vipTime *time.Time
	return model.DB.Transaction(func(tx *gorm.DB) error {
		return completeAppleOrder(tx, c, order.Uid, pack, appleTransactionInfoFromNotification(transaction), model.OrderPaySourceAppleS2S, &orderNo, &vipTime)
	})
}

func appleTransactionInfoFromNotification(transaction *apple.TransactionInfo) appleTransactionInfo {
	if transaction == nil {
		return appleTransactionInfo{}
	}
	return appleTransactionInfo{
		TransactionId:   transaction.TransactionId,
		OriginalId:      transaction.OriginalTransactionId,
		ProductId:       transaction.ProductId,
		BundleId:        transaction.BundleId,
		AppAccountToken: transaction.AppAccountToken,
		PurchaseDate:    transaction.PurchaseDate,
		RevocationDate:  transaction.RevocationDate,
	}
}

func processAppleRefundNotification(transaction *apple.TransactionInfo) error {
	order, err := model.GetAppleOrderByTransactionId(transaction.TransactionId)
	if err != nil {
		if model.IsOrderNotFound(err) {
			return nil
		}
		return err
	}
	if order.PayStatus == model.OrderPayStatusRefund {
		return nil
	}
	return model.DB.Transaction(func(tx *gorm.DB) error {
		var lockedOrder model.Order
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", order.Id).First(&lockedOrder).Error; err != nil {
			return err
		}
		if lockedOrder.PayStatus == model.OrderPayStatusRefund {
			return nil
		}
		var user model.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", lockedOrder.Uid).First(&user).Error; err != nil {
			return err
		}
		vipTime := subtractVipSeconds(user.VipTime, lockedOrder.PakTime)
		if err := tx.Model(&model.Order{}).Where("id = ?", lockedOrder.Id).Updates(map[string]interface{}{
			"pay_status": model.OrderPayStatusRefund,
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.User{}).Where("id = ?", user.Id).Updates(map[string]interface{}{
			"vip_time":  vipTime,
			"pay_times": gorm.Expr("GREATEST(pay_times - 1, 0)"),
			"pay_all":   gorm.Expr("GREATEST(pay_all - ?, 0)", float64(lockedOrder.Money)/100),
		}).Error; err != nil {
			return err
		}
		if user.Username != "" && user.Type == 2 {
			if err := tx.Model(&model.UserAccount{}).Where("username = ?", user.Username).Update("vip_time", vipTime).Error; err != nil {
				return err
			}
			if err := syncAccountVipTime(tx, user.Username, vipTime); err != nil {
				return err
			}
		}
		return tx.Create(&model.UserNotice{
			UserId: user.Id,
			Title:  multilingualTextValue("会员权益已调整", "Membership updated"),
			Content: multilingualTextValue(
				fmt.Sprintf("苹果订单%s已退款或撤销，会员有效期已调整", lockedOrder.OrderNo),
				fmt.Sprintf("Apple order %s was refunded or revoked. Your membership expiry has been updated.", lockedOrder.OrderNo),
			),
			Priority: 100,
			Status:   model.UserNoticeStatusUnread,
		}).Error
	})
}

func appleNotificationAlreadyHandled(notificationUUID string) bool {
	if strings.TrimSpace(notificationUUID) == "" {
		return false
	}
	var existing model.AppleNotification
	err := model.DB.Where("notification_uuid = ?", notificationUUID).First(&existing).Error
	if err != nil {
		return false
	}
	return existing.ProcessStatus == model.AppleNotificationStatusDone ||
		existing.ProcessStatus == model.AppleNotificationStatusIgnored
}

func validateAppleNotification(payload *apple.NotificationV2Payload, transaction *apple.TransactionInfo) error {
	bundleId := strings.TrimSpace(model.PayConfigValue(model.PayConfigAppleBundleId, ""))
	if bundleId == "" {
		return nil
	}
	if payload != nil && payload.Data != nil && payload.Data.BundleID != "" && payload.Data.BundleID != bundleId {
		return fmt.Errorf("apple bundle id mismatch")
	}
	if transaction != nil && transaction.BundleId != "" && transaction.BundleId != bundleId {
		return fmt.Errorf("apple transaction bundle id mismatch")
	}
	return nil
}

func saveAppleNotificationLog(payload *apple.NotificationV2Payload, transaction *apple.TransactionInfo, rawPayload string, status int, msg string) error {
	if payload == nil {
		return nil
	}
	notificationUUID := payload.NotificationUUID
	if strings.TrimSpace(notificationUUID) == "" {
		sum := sha1.Sum([]byte(rawPayload))
		notificationUUID = "payload_" + hex.EncodeToString(sum[:])
	}
	notification := model.AppleNotification{
		BaseModel: model.BaseModel{
			CreateTime: chinaNow(),
		},
		NotificationUUID: notificationUUID,
		NotificationType: payload.NotificationType,
		Subtype:          payload.Subtype,
		ProcessStatus:    status,
		ProcessMsg:       truncateString(msg, 255),
		RawPayload:       rawPayload,
	}
	if payload.Data != nil {
		notification.Environment = payload.Data.Environment
	}
	if transaction != nil {
		notification.TransactionId = transaction.TransactionId
		notification.OriginalTransactionId = transaction.OriginalTransactionId
		notification.ProductId = transaction.ProductId
		notification.AppAccountToken = transaction.AppAccountToken
		if notification.Environment == "" {
			notification.Environment = transaction.Environment
		}
	}
	var existing model.AppleNotification
	err := model.DB.Where("notification_uuid = ?", notification.NotificationUUID).First(&existing).Error
	if err == nil {
		return model.DB.Model(&model.AppleNotification{}).Where("id = ?", existing.Id).Updates(map[string]interface{}{
			"process_status": status,
			"process_msg":    notification.ProcessMsg,
		}).Error
	}
	if !model.IsOrderNotFound(err) {
		return err
	}
	return model.DB.Create(&notification).Error
}

func truncateString(value string, max int) string {
	if max <= 0 || len(value) <= max {
		return value
	}
	return value[:max]
}

func lockAppleTransaction(tx *gorm.DB, transactionId string) (func(), error) {
	sum := sha1.Sum([]byte(transactionId))
	lockName := "apple_pay_" + hex.EncodeToString(sum[:])
	var locked int
	if err := tx.Raw("SELECT GET_LOCK(?, 10)", lockName).Scan(&locked).Error; err != nil {
		return nil, err
	}
	if locked != 1 {
		return nil, fmt.Errorf("apple transaction lock timeout")
	}
	return func() {
		_ = tx.Exec("SELECT RELEASE_LOCK(?)", lockName).Error
	}, nil
}
