package controller

import (
	crand "crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"just-vpn/middleware"
	"just-vpn/model"
	"just-vpn/pkg/jwt"
	"just-vpn/pkg/mapping"
	"just-vpn/pkg/setting"
	"just-vpn/pkg/util"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	CodeError = 500
)

type autoLoginParams struct {
	DeviceNo        string
	Platform        string
	Version         string
	AppStoreRegion  string
	MobileName      string
	MobileVersion   string
	MobileModelName string
	SourceChannel   string
}

type AutoLoginRequest struct {
	DeviceNo        string `json:"device_no" binding:"required" example:"device-001"`
	Platform        string `json:"platform" example:"iphone"`
	Version         string `json:"version" example:"1.0.0"`
	MobileName      string `json:"mobile_name" example:"iPhone"`
	MobileVersion   string `json:"mobile_version" example:"iOS 17.5"`
	MobileModelName string `json:"mobile_model_name" example:"iPhone 15"`
	SourceChannel   string `json:"source_channel" example:"default"`
}

type registerParams struct {
	DeviceNo string
	Username string
	Password string
}

type loginParams struct {
	DeviceNo        string
	Username        string
	Password        string
	Platform        string
	Version         string
	MobileName      string
	MobileVersion   string
	MobileModelName string
	SourceChannel   string
}

type changePasswordParams struct {
	NewPassword string
}

type deviceLogoutParams struct {
	Id int
}

type inviteCodeParams struct {
	InviteCode string
}

type RegisterRequest struct {
	DeviceNo string `json:"device_no" binding:"required" example:"device-001"`
	Username string `json:"username" binding:"required" example:"user001"`
	Password string `json:"password" binding:"required" example:"pass001"`
}

type LoginRequest struct {
	DeviceNo        string `json:"device_no" binding:"required" example:"device-001"`
	Username        string `json:"username" binding:"required" example:"user001"`
	Password        string `json:"password" binding:"required" example:"pass001"`
	Platform        string `json:"platform" example:"iphone"`
	Version         string `json:"version" example:"1.0.0"`
	MobileName      string `json:"mobile_name" example:"iPhone"`
	MobileVersion   string `json:"mobile_version" example:"iOS 17.5"`
	MobileModelName string `json:"mobile_model_name" example:"iPhone 15"`
	SourceChannel   string `json:"source_channel" example:"default"`
}

type ChangePasswordRequest struct {
	NewPassword string `json:"new_password" binding:"required" example:"pass002"`
}

type DeviceLogoutRequest struct {
	Id int `json:"id" binding:"required" example:"2"`
}

type InviteCodeRequest struct {
	InviteCode string `json:"invite_code" binding:"required" example:"A8K29QXZ"`
}

type UserInfoResponse struct {
	Id            int    `json:"id" example:"1"`                                          // 用户ID
	Token         string `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."` // JWT登录凭证
	DeviceNo      string `json:"device_no" example:"device-001"`                          // 设备唯一标识
	Username      string `json:"username" example:""`                                     // 账号名，游客为空
	Type          int    `json:"type" example:"1"`                                        // 用户类型(1=游客,2=账号用户)
	IsVip         int    `json:"is_vip" example:"0"`                                      // 会员状态(0=非会员,1=会员有效)
	VipTime       string `json:"vip_time" example:"2026-07-30 23:59:59"`                  // 会员到期时间，格式yyyy-MM-dd HH:mm:ss
	IsNewUser     int    `json:"is_new_user" example:"1"`                                 // 是否新用户(0=否,1=是)
	Platform      string `json:"platform" example:"iphone"`                               // 客户端平台
	Version       string `json:"version" example:"1.0.0"`                                 // 客户端版本号
	LoginTimes    int    `json:"login_times" example:"1"`                                 // 登录次数
	CreateTime    string `json:"create_time" example:"2026-06-29 12:00:00"`               // 用户创建时间
	LastLoginTime string `json:"last_login_time" example:"2026-06-29 12:00:00"`           // 本次登录时间
	Password      string `json:"password" example:"pass001"`                              // 绑定账号的明文密码，游客为空
}

type LogoffResponse struct {
	Token string `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

type DeviceResponse struct {
	Id              int    `json:"id" example:"1"`
	DeviceNo        string `json:"device_no" example:"device-001"`
	MobileName      string `json:"mobile_name" example:"iPhone"`
	MobileVersion   string `json:"mobile_version" example:"iOS 17.5"`
	MobileModelName string `json:"mobile_model_name" example:"iPhone 15"`
	Platform        string `json:"platform" example:"iphone"`
	Version         string `json:"version" example:"1.0.0"`
	LastLoginTime   string `json:"last_login_time" example:"2026-06-29 12:00:00"`
}

type DeviceInfoResponse struct {
	LocalDevice DeviceResponse   `json:"local_device"`
	TwoDevice   DeviceResponse   `json:"two_device"`
	Devices     []DeviceResponse `json:"devices"`
}

type InviteCodeResponse struct {
	InviteCode string `json:"invite_code" example:"A8K29QXZ"`
}

type InviteMilestoneResponse struct {
	Count         int    `json:"count" example:"24"`
	RewardSeconds int    `json:"reward_seconds" example:"2592000"`
	RewardText    string `json:"reward_text" example:"1个月会员"`
}

type InviteDetailResponse struct {
	InviteCode       string                    `json:"invite_code" example:"A8K29QXZ"`
	InviteValid      bool                      `json:"invite_valid" example:"true"`
	ShareCount       int                       `json:"share_count" example:"3"`
	RewardTime       int                       `json:"reward_time" example:"10800"`
	PerInviteSeconds int                       `json:"per_invite_seconds" example:"3600"`
	PerInviteText    string                    `json:"per_invite_text" example:"1小时会员"`
	Milestones       []InviteMilestoneResponse `json:"milestones"`
}

type inviteRewardConfig struct {
	PerInviteSeconds int               `json:"per_invite_seconds"`
	Milestones       []inviteMilestone `json:"milestones"`
}

type inviteMilestone struct {
	Count         int `json:"count"`
	RewardSeconds int `json:"reward_seconds"`
}

func getMappedBodyString(body map[string]interface{}, originalPath string, field string, defaultValue string) string {
	value, ok := body[mapping.RequestField(originalPath, field)]
	if !ok || value == nil {
		return defaultValue
	}
	switch v := value.(type) {
	case string:
		if v == "" {
			return defaultValue
		}
		return v
	default:
		return fmt.Sprint(v)
	}
}

func getMappedBodyInt(body map[string]interface{}, originalPath string, field string, defaultValue int) int {
	value, ok := body[mapping.RequestField(originalPath, field)]
	if !ok || value == nil {
		return defaultValue
	}
	switch v := value.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case string:
		if v == "" {
			return defaultValue
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			return defaultValue
		}
		return n
	default:
		n, err := strconv.Atoi(fmt.Sprint(v))
		if err != nil {
			return defaultValue
		}
		return n
	}
}

func normalizePlatform(platform string) string {
	platform = strings.TrimSpace(platform)
	if platform == "ios" || platform == "iphone" || platform == "ipad" {
		return "iphone"
	}
	return platform
}

func getAutoLoginParams(body map[string]interface{}) autoLoginParams {
	const originalPath = "/api/v1/auto_login"
	return autoLoginParams{
		DeviceNo:        getMappedBodyString(body, originalPath, "device_no", ""),
		Platform:        normalizePlatform(getMappedBodyString(body, originalPath, "platform", "")),
		Version:         getMappedBodyString(body, originalPath, "version", ""),
		MobileName:      getMappedBodyString(body, originalPath, "mobile_name", ""),
		MobileVersion:   getMappedBodyString(body, originalPath, "mobile_version", ""),
		MobileModelName: getMappedBodyString(body, originalPath, "mobile_model_name", ""),
		SourceChannel:   getMappedBodyString(body, originalPath, "source_channel", "default"),
	}
}

func applyClientInfoToAutoLoginParams(c *gin.Context, params autoLoginParams) autoLoginParams {
	client := middleware.CurrentClientInfo(c)
	if client.Version != "" {
		params.Version = client.Version
	}
	if client.AppStoreRegion != "" {
		params.AppStoreRegion = client.AppStoreRegion
	}
	return params
}

func getRegisterParams(body map[string]interface{}) registerParams {
	const originalPath = "/api/v1/register"
	return registerParams{
		DeviceNo: getMappedBodyString(body, originalPath, "device_no", ""),
		Username: getMappedBodyString(body, originalPath, "username", ""),
		Password: getMappedBodyString(body, originalPath, "password", ""),
	}
}

func getLoginParams(body map[string]interface{}) loginParams {
	const originalPath = "/api/v1/login"
	return loginParams{
		DeviceNo:        getMappedBodyString(body, originalPath, "device_no", ""),
		Username:        getMappedBodyString(body, originalPath, "username", ""),
		Password:        getMappedBodyString(body, originalPath, "password", ""),
		Platform:        normalizePlatform(getMappedBodyString(body, originalPath, "platform", "")),
		Version:         getMappedBodyString(body, originalPath, "version", ""),
		MobileName:      getMappedBodyString(body, originalPath, "mobile_name", ""),
		MobileVersion:   getMappedBodyString(body, originalPath, "mobile_version", ""),
		MobileModelName: getMappedBodyString(body, originalPath, "mobile_model_name", ""),
		SourceChannel:   getMappedBodyString(body, originalPath, "source_channel", "default"),
	}
}

func getChangePasswordParams(body map[string]interface{}) changePasswordParams {
	const originalPath = "/api/v1/change_password"
	return changePasswordParams{
		NewPassword: getMappedBodyString(body, originalPath, "new_password", ""),
	}
}

func getDeviceLogoutParams(body map[string]interface{}) deviceLogoutParams {
	const originalPath = "/api/v1/device_logout"
	return deviceLogoutParams{
		Id: getMappedBodyInt(body, originalPath, "id", 0),
	}
}

func getInviteCodeParams(body map[string]interface{}) inviteCodeParams {
	const originalPath = "/api/v1/invite_code"
	return inviteCodeParams{
		InviteCode: strings.ToUpper(getMappedBodyString(body, originalPath, "invite_code", "")),
	}
}

func validateAccountParams(username string, password string) string {
	if username == "" {
		return "username is required"
	}
	if !util.CheckUsername(username) {
		return "username format error"
	}
	if password == "" {
		return "password is required"
	}
	if !util.CheckPwd(password) {
		return "password format error"
	}
	return ""
}

func validatePassword(password string, fieldName string) string {
	if password == "" {
		return fieldName + " is required"
	}
	if !util.CheckPwd(password) {
		return fieldName + " format error"
	}
	return ""
}

func chinaNow() time.Time {
	return time.Now().In(time.Local).Truncate(time.Second)
}

func isValidVipTime(vipTime *time.Time) bool {
	return vipTime != nil && vipTime.After(chinaNow())
}

func formatVipTime(vipTime *time.Time) string {
	if vipTime == nil {
		return ""
	}
	return vipTime.In(time.Local).Format("2006-01-02 15:04:05")
}

func equalVipTime(left *time.Time, right *time.Time) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Equal(*right)
}

func mergeVipTime(vipTimes ...*time.Time) *time.Time {
	now := chinaNow()
	var totalRemaining time.Duration
	for _, vipTime := range vipTimes {
		if vipTime == nil {
			continue
		}
		remaining := vipTime.In(time.Local).Sub(now)
		if remaining <= 0 {
			continue
		}
		totalRemaining += remaining
	}
	if totalRemaining <= 0 {
		return nil
	}
	merged := now.Add(totalRemaining).Truncate(time.Second)
	return &merged
}

func addVipSeconds(vipTime *time.Time, seconds int) *time.Time {
	if seconds <= 0 {
		return vipTime
	}
	now := chinaNow()
	base := now
	if vipTime != nil && vipTime.After(now) {
		base = vipTime.In(time.Local)
	}
	result := base.Add(time.Duration(seconds) * time.Second).Truncate(time.Second)
	return &result
}

func subtractVipSeconds(vipTime *time.Time, seconds int) *time.Time {
	if vipTime == nil || seconds <= 0 {
		return vipTime
	}
	now := chinaNow()
	remaining := vipTime.In(time.Local).Sub(now) - time.Duration(seconds)*time.Second
	if remaining <= 0 {
		return nil
	}
	result := now.Add(remaining).Truncate(time.Second)
	return &result
}

func syncAccountVipTime(tx *gorm.DB, username string, vipTime *time.Time) error {
	return tx.Model(&model.User{}).
		Where("username = ?", username).
		Where("type = ?", 2).
		Where("status = ?", model.UserStatusNormal).
		Update("vip_time", vipTime).Error
}

func newAppleAccountToken() string {
	var b [16]byte
	if _, err := crand.Read(b[:]); err != nil {
		seed := time.Now().UnixNano()
		for i := range b {
			b[i] = byte(seed >> ((i % 8) * 8))
		}
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return strings.ToUpper(fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]))
}

func syncInviterVipTime(tx *gorm.DB, user model.User, vipTime *time.Time) error {
	if user.Username == "" || user.Type != 2 {
		return nil
	}
	if err := tx.Model(&model.UserAccount{}).Where("username = ?", user.Username).Updates(map[string]interface{}{
		"vip_time": vipTime,
	}).Error; err != nil {
		return err
	}
	return syncAccountVipTime(tx, user.Username, vipTime)
}

func inviteConfigInt(key string, defaultValue int) int {
	value, err := model.GetInviteConfigValue(key)
	if err != nil || value == "" {
		return defaultValue
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return n
}

func configValueInt(code string, defaultValue int) int {
	value := model.ConfigValue(code, "")
	if value == "" {
		return defaultValue
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return n
}

func inviteValid(user model.User) bool {
	if user.InviterId != nil {
		return false
	}
	validHours := inviteConfigInt(model.InviteConfigValidHours, 72)
	if validHours <= 0 {
		return true
	}
	expireAt := user.CreateTime.In(time.Local).Add(time.Duration(validHours) * time.Hour)
	return !chinaNow().After(expireAt)
}

func inviteRewardConfigValue() (inviteRewardConfig, error) {
	value, err := model.GetInviteConfigValue(model.InviteConfigRewardRules)
	if err != nil {
		return inviteRewardConfig{}, err
	}

	var config inviteRewardConfig
	if err := json.Unmarshal([]byte(value), &config); err == nil && config.PerInviteSeconds > 0 {
		config.Milestones = validInviteMilestones(config.Milestones)
		return config, nil
	}

	var legacyRules []struct {
		MinCount      int `json:"min_count"`
		RewardSeconds int `json:"reward_seconds"`
	}
	if err := json.Unmarshal([]byte(value), &legacyRules); err != nil {
		return inviteRewardConfig{}, err
	}
	for _, rule := range legacyRules {
		if rule.MinCount == 1 && rule.RewardSeconds > 0 {
			config.PerInviteSeconds = rule.RewardSeconds
			continue
		}
		if rule.MinCount > 1 && rule.RewardSeconds > 0 {
			config.Milestones = append(config.Milestones, inviteMilestone{
				Count:         rule.MinCount,
				RewardSeconds: rule.RewardSeconds,
			})
		}
	}
	if config.PerInviteSeconds <= 0 {
		return inviteRewardConfig{}, fmt.Errorf("invite reward config empty")
	}
	config.Milestones = validInviteMilestones(config.Milestones)
	return config, nil
}

func validInviteMilestones(milestones []inviteMilestone) []inviteMilestone {
	valid := make([]inviteMilestone, 0, len(milestones))
	for _, milestone := range milestones {
		if milestone.Count > 0 && milestone.RewardSeconds > 0 {
			valid = append(valid, milestone)
		}
	}
	return valid
}

func calculateInviteReward(shareCount int) (int, error) {
	if shareCount <= 0 {
		return 0, nil
	}
	config, err := inviteRewardConfigValue()
	if err != nil {
		return 0, err
	}
	reward := config.PerInviteSeconds
	for _, milestone := range config.Milestones {
		if shareCount%milestone.Count == 0 {
			reward += milestone.RewardSeconds
		}
	}
	return reward, nil
}

func limitInviteRewardByMaxReward(currentRewardTime int, rewardSeconds int) int {
	maxRewardDays := inviteConfigInt(model.InviteConfigMaxRewardDays, 300)
	if maxRewardDays <= 0 {
		return rewardSeconds
	}
	remaining := maxRewardDays*24*60*60 - currentRewardTime
	if remaining <= 0 {
		return 0
	}
	if rewardSeconds > remaining {
		return remaining
	}
	return rewardSeconds
}

func rewardText(seconds int) string {
	if seconds%(30*24*60*60) == 0 {
		return strconv.Itoa(seconds/(30*24*60*60)) + "个月会员"
	}
	if seconds%(24*60*60) == 0 {
		return strconv.Itoa(seconds/(24*60*60)) + "天会员"
	}
	if seconds%(60*60) == 0 {
		return strconv.Itoa(seconds/(60*60)) + "小时会员"
	}
	if seconds%60 == 0 {
		return strconv.Itoa(seconds/60) + "分钟会员"
	}
	return strconv.Itoa(seconds) + "秒会员"
}

func inviteRewardDetail() (inviteRewardConfig, []InviteMilestoneResponse, error) {
	config, err := inviteRewardConfigValue()
	if err != nil {
		return inviteRewardConfig{}, nil, err
	}
	responses := make([]InviteMilestoneResponse, 0, len(config.Milestones))
	for _, milestone := range config.Milestones {
		responses = append(responses, InviteMilestoneResponse{
			Count:         milestone.Count,
			RewardSeconds: milestone.RewardSeconds,
			RewardText:    rewardText(milestone.RewardSeconds),
		})
	}
	return config, responses, nil
}

func ensureInviteCode(userID int) (model.UserInviteCode, error) {
	inviteCode, err := model.GetUserInviteCodeByUserID(userID)
	if err == nil && inviteCode.Id != 0 {
		return inviteCode, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return model.UserInviteCode{}, err
	}
	for i := 0; i < 8; i++ {
		code := util.RandomString(8)
		now := chinaNow()
		inviteCode = model.UserInviteCode{
			BaseModel: model.BaseModel{
				CreateTime: now,
			},
			UserId:     userID,
			InviteCode: code,
			Status:     1,
		}
		if err := model.DB.Create(&inviteCode).Error; err == nil {
			return inviteCode, nil
		}
		if existing, err := model.GetUserInviteCodeByUserID(userID); err == nil && existing.Id != 0 {
			return existing, nil
		}
	}
	return model.UserInviteCode{}, fmt.Errorf("generate invite code failed")
}

func reloadUser(id int) (model.User, error) {
	return model.GetUserByID(id)
}

func ensureLoginAllowed(userId int, deviceNo string) error {
	banned, err := model.IsUserBannedInCache(userId, deviceNo)
	if err != nil {
		return fmt.Errorf("ban check failed")
	}
	if banned {
		return fmt.Errorf("user is banned")
	}
	return nil
}

func buildDeviceResponse(user model.User, isLocal bool) DeviceResponse {
	lastLoginTime := ""
	if user.LastLoginTime != nil {
		lastLoginTime = formatVipTime(user.LastLoginTime)
	}
	mobileName := user.MobileName
	if mobileName == "" && isLocal {
		mobileName = "本机设备"
	}
	return DeviceResponse{
		Id:              user.Id,
		DeviceNo:        user.DeviceNo,
		MobileName:      mobileName,
		MobileVersion:   user.MobileVersion,
		MobileModelName: user.MobileModelName,
		Platform:        user.Platform,
		Version:         user.Version,
		LastLoginTime:   lastLoginTime,
	}
}

func buildDeviceInfo(user model.User) (DeviceInfoResponse, error) {
	deviceInfo := DeviceInfoResponse{
		LocalDevice: buildDeviceResponse(user, true),
		Devices:     make([]DeviceResponse, 0),
	}
	if user.Type == 1 || user.Username == "" {
		return deviceInfo, nil
	}
	users, err := model.GetUsersByUsername(user.Username)
	if err != nil {
		return DeviceInfoResponse{}, err
	}
	for _, item := range users {
		if item.Id == user.Id {
			continue
		}
		device := buildDeviceResponse(item, false)
		if deviceInfo.TwoDevice.Id == 0 {
			deviceInfo.TwoDevice = device
		}
		deviceInfo.Devices = append(deviceInfo.Devices, device)
	}
	return deviceInfo, nil
}

func buildGuestUser(c *gin.Context, params autoLoginParams, vipTime *time.Time, loginTimes int) model.User {
	now := chinaNow()
	ip := c.ClientIP()
	ipRegion := loginIPRegion(ip)
	return model.User{
		BaseModel: model.BaseModel{
			CreateTime: now,
		},
		DeviceNo:        params.DeviceNo,
		Status:          model.UserStatusNormal,
		Type:            1,
		VipTime:         vipTime,
		Platform:        params.Platform,
		Version:         params.Version,
		AppStoreRegion:  params.AppStoreRegion,
		IsPay:           -1,
		RegIp:           ip,
		RegIpRegion:     ipRegion,
		LastIp:          ip,
		LastIpRegion:    ipRegion,
		LastLoginTime:   &now,
		LoginTimes:      loginTimes,
		IsReal:          -1,
		IsTodayActive:   1,
		IsFlowClose:     1,
		DeviceStatus:    1,
		MobileName:      params.MobileName,
		MobileVersion:   params.MobileVersion,
		MobileModelName: params.MobileModelName,
		SourceChannel:   params.SourceChannel,
	}
}

func createDeviceUser(c *gin.Context, params autoLoginParams) (model.User, error) {
	freeSeconds := configValueInt(model.ConfigNewUserFreeSeconds, 3600)
	firstVipTime := addVipSeconds(nil, freeSeconds)
	user := buildGuestUser(c, params, firstVipTime, 0)

	if err := model.DB.Transaction(func(tx *gorm.DB) error {
		if err := model.CreateUserWithRandomIDAndTransferCode(tx, &user, currentProductRouteCode()); err != nil {
			return err
		}
		if freeSeconds <= 0 {
			return nil
		}
		return tx.Create(&model.UserNotice{
			UserId:   user.Id,
			Title:    "欢迎使用",
			Content:  "欢迎使用，已为你免费赠送" + rewardText(freeSeconds) + "时长，可立即体验高速线路",
			Priority: 100,
			Status:   model.UserNoticeStatusUnread,
		}).Error
	}); err != nil {
		return model.User{}, err
	}
	return reloadUser(user.Id)
}

func updateLogin(user model.User, ip string, params autoLoginParams) error {
	if params.DeviceNo == "" {
		params.DeviceNo = user.DeviceNo
	}
	platform := params.Platform
	if platform == "" {
		platform = user.Platform
	}
	user.DeviceNo = params.DeviceNo
	now := chinaNow()
	if err := model.UpdateUserByID(user.Id, model.User{
		LastIp:          ip,
		LastIpRegion:    loginIPRegion(ip),
		LastLoginTime:   &now,
		LoginTimes:      user.LoginTimes + 1,
		IsTodayActive:   1,
		Platform:        platform,
		Version:         params.Version,
		AppStoreRegion:  params.AppStoreRegion,
		MobileName:      params.MobileName,
		MobileVersion:   params.MobileVersion,
		MobileModelName: params.MobileModelName,
	}); err != nil {
		return err
	}
	return nil
}

func loginIPRegion(ip string) string {
	if ip == "" || setting.AppConfig.IpDbPath == "" {
		return ""
	}
	info := util.GetIpInfo(setting.AppConfig.IpDbPath, ip)
	return info.String()
}

func buildUserInfo(user model.User, isNewUser int) (UserInfoResponse, error) {
	isVip := 0
	if isValidVipTime(user.VipTime) {
		isVip = 1
	}
	token, err := jwt.Generate(user.Id, user.DeviceNo)
	if err != nil {
		return UserInfoResponse{}, err
	}
	return UserInfoResponse{
		Id:            user.Id,
		Token:         token,
		DeviceNo:      user.DeviceNo,
		Username:      user.Username,
		Type:          user.Type,
		IsVip:         isVip,
		VipTime:       formatVipTime(user.VipTime),
		IsNewUser:     isNewUser,
		Platform:      user.Platform,
		Version:       user.Version,
		LoginTimes:    user.LoginTimes,
		CreateTime:    user.CreateTime.Format("2006-01-02 15:04:05"),
		LastLoginTime: chinaNow().Format("2006-01-02 15:04:05"),
	}, nil
}

func fillUserInfoPassword(info *UserInfoResponse, user model.User) error {
	if user.Username == "" {
		return nil
	}

	account, err := model.GetUserAccountByUsername(user.Username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	info.Password = account.Password
	return nil
}

func returnUserInfo(c *gin.Context, user model.User, isNewUser int) {
	data, err := buildUserInfo(user, isNewUser)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	if err := fillUserInfoPassword(&data, user); err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	if isNewUser == 1 {
		_ = model.RecordDailyNewUser(user.Id)
	} else {
		_ = model.RecordDailyActive(user.Id)
	}
	JsonReturn(c, CodeSuccess, "success", data)
}

func returnLogoffInfo(c *gin.Context, user model.User) {
	token, err := jwt.Generate(user.Id, user.DeviceNo)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	JsonReturn(c, CodeSuccess, "success", LogoffResponse{
		Token: token,
	})
}

// AutoLoginHandler 自动登录
// @Summary 自动登录
// @Description 通过设备号登录，设备不存在时自动创建普通用户，新用户会按配置赠送会员时长并生成欢迎通知
// @Tags 账号
// @Accept json
// @Produce json
// @Param request body AutoLoginRequest true "自动登录请求"
// @Success 200 {object} Response{result=UserInfoResponse}
// @Router /auto_login [post]
func AutoLoginHandler(c *gin.Context) {
	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		JsonReturn(c, CodeError, "invalid json body", nil)
		return
	}

	params := applyClientInfoToAutoLoginParams(c, getAutoLoginParams(body))
	if params.DeviceNo == "" {
		JsonReturn(c, CodeError, "device_no is required", nil)
		return
	}
	if err := ensureLoginAllowed(0, params.DeviceNo); err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}

	isNewUser := 0
	user, err := model.GetUserByDeviceNoAnyStatus(params.DeviceNo)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			JsonReturn(c, CodeError, err.Error(), nil)
			return
		}
		user, err = createDeviceUser(c, params)
		if err != nil {
			if model.IsDuplicateEntryError(err) {
				existingUser, loadErr := model.GetUserByDeviceNoAnyStatus(params.DeviceNo)
				if loadErr == nil {
					user = existingUser
				} else {
					JsonReturn(c, CodeError, loadErr.Error(), nil)
					return
				}
			} else {
				JsonReturn(c, CodeError, err.Error(), nil)
				return
			}
		} else {
			isNewUser = 1
		}
	}
	if user.Status == model.UserStatusBanned {
		JsonReturn(c, CodeError, "user is banned", nil)
		return
	}
	if isNewUser == 0 && isFirstMigrationActivation(user) {
		isNewUser = 1
	}

	if err := updateLogin(user, c.ClientIP(), params); err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	user, err = reloadUser(user.Id)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	returnUserInfo(c, user, isNewUser)
}

func isFirstMigrationActivation(user model.User) bool {
	return user.MigrationSource != nil && user.LastLoginTime == nil
}

// RegisterHandler 注册账号
// @Summary 注册账号
// @Description 将当前设备用户绑定为账号用户，注册成功后返回 token
// @Tags 账号
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "注册请求"
// @Success 200 {object} Response{result=UserInfoResponse}
// @Security BearerAuth
// @Router /register [post]
func RegisterHandler(c *gin.Context) {
	currentUser, err := middleware.CurrentUser(c)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}

	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		JsonReturn(c, CodeError, "invalid json body", nil)
		return
	}

	params := getRegisterParams(body)
	if params.DeviceNo == "" {
		JsonReturn(c, CodeError, "device_no is required", nil)
		return
	}
	if params.DeviceNo != currentUser.DeviceNo {
		JsonReturn(c, CodeError, "token device invalid", nil)
		return
	}
	if msg := validateAccountParams(params.Username, params.Password); msg != "" {
		JsonReturn(c, CodeError, msg, nil)
		return
	}

	if user, err := model.GetUserByUsername(params.Username); err == nil && user.Id != 0 {
		JsonReturn(c, CodeError, "account already registered", nil)
		return
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	if account, err := model.GetUserAccountByUsername(params.Username); err == nil && account.Id != 0 {
		JsonReturn(c, CodeError, "account already registered", nil)
		return
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}

	user := currentUser
	if user.Status == model.UserStatusBanned {
		JsonReturn(c, CodeError, "user is banned", nil)
		return
	}
	if user.Username != "" || user.Type == 2 {
		JsonReturn(c, CodeError, "account already registered", nil)
		return
	}

	mergedVipTime := mergeVipTime(user.VipTime)
	if err := model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&model.UserAccount{
			BaseModel: model.BaseModel{
				CreateTime: chinaNow(),
			},
			Username: params.Username,
			Password: params.Password,
			VipTime:  mergedVipTime,
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.User{}).Where("id = ?", user.Id).Updates(map[string]interface{}{
			"username": params.Username,
			"type":     2,
			"vip_time": mergedVipTime,
		}).Error; err != nil {
			return err
		}
		return syncAccountVipTime(tx, params.Username, mergedVipTime)
	}); err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}

	user, err = model.GetUserByID(user.Id)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}

	loginParams := autoLoginParams{
		DeviceNo: user.DeviceNo,
		Platform: user.Platform,
	}
	if err := updateLogin(user, c.ClientIP(), loginParams); err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	user, err = model.GetUserByID(user.Id)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}

	_ = model.CreateUserRecord(model.UserRecord{
		BaseModel: model.BaseModel{
			CreateTime: chinaNow(),
		},
		UserId: user.Id,
		Remark: "注册账号:" + params.Username,
	})

	returnUserInfo(c, user, 0)
}

// LoginHandler 账号登录
// @Summary 账号登录
// @Description 当前设备登录账号，会员以账号表为主并和当前设备会员合并
// @Tags 账号
// @Accept json
// @Produce json
// @Param request body LoginRequest true "登录请求"
// @Success 200 {object} Response{result=UserInfoResponse}
// @Security BearerAuth
// @Router /login [post]
func LoginHandler(c *gin.Context) {
	currentUser, err := middleware.CurrentUser(c)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}

	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		JsonReturn(c, CodeError, "invalid json body", nil)
		return
	}

	params := getLoginParams(body)
	if params.DeviceNo == "" {
		JsonReturn(c, CodeError, "device_no is required", nil)
		return
	}
	if params.DeviceNo != currentUser.DeviceNo {
		JsonReturn(c, CodeError, "token device invalid", nil)
		return
	}
	if msg := validateAccountParams(params.Username, params.Password); msg != "" {
		JsonReturn(c, CodeError, msg, nil)
		return
	}

	user := currentUser
	if user.Status == model.UserStatusBanned {
		JsonReturn(c, CodeError, "user is banned", nil)
		return
	}
	if user.Username != "" && user.Username != params.Username {
		JsonReturn(c, CodeError, "current device already logged in", nil)
		return
	}

	if err := model.DB.Transaction(func(tx *gorm.DB) error {
		account, err := getUserAccountForLogin(tx, params.Username)
		if err != nil {
			return err
		}
		if account.Password != params.Password {
			return fmt.Errorf("password error")
		}
		if err := ensureAccountDeviceLimit(tx, user, params.Username); err != nil {
			return err
		}

		mergedVipTime := mergeVipTime(user.VipTime, account.VipTime)
		if !equalVipTime(mergedVipTime, account.VipTime) {
			if err := tx.Model(&model.UserAccount{}).Where("username = ?", params.Username).Updates(map[string]interface{}{
				"vip_time": mergedVipTime,
			}).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&model.User{}).Where("id = ?", user.Id).Updates(map[string]interface{}{
			"username": params.Username,
			"type":     2,
			"vip_time": mergedVipTime,
		}).Error; err != nil {
			return err
		}
		return syncAccountVipTime(tx, params.Username, mergedVipTime)
	}); err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}

	user, err = model.GetUserByID(user.Id)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}

	if err := updateLogin(user, c.ClientIP(), autoLoginParams{
		DeviceNo:        user.DeviceNo,
		Platform:        params.Platform,
		Version:         params.Version,
		MobileName:      params.MobileName,
		MobileVersion:   params.MobileVersion,
		MobileModelName: params.MobileModelName,
		SourceChannel:   params.SourceChannel,
	}); err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}

	user, err = model.GetUserByID(user.Id)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}

	_ = model.CreateUserRecord(model.UserRecord{
		BaseModel: model.BaseModel{
			CreateTime: chinaNow(),
		},
		UserId: user.Id,
		Remark: "登录账号:" + params.Username,
	})

	returnUserInfo(c, user, 0)
}

// LogoutHandler 退出登录
// @Summary 退出登录
// @Description 当前设备退出账号并清空设备会员，账号表会员保持不变
// @Tags 账号
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} Response{result=UserInfoResponse}
// @Router /logout [post]
func LogoutHandler(c *gin.Context) {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	if user.Type == 1 || user.Username == "" {
		JsonReturn(c, CodeError, "account not logged in", nil)
		return
	}
	username := user.Username

	if err := model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.User{}).Where("id = ?", user.Id).Updates(map[string]interface{}{
			"username": "",
			"type":     1,
			"vip_time": nil,
		}).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}

	user, err = model.GetUserByID(user.Id)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	if err := updateLogin(user, c.ClientIP(), autoLoginParams{
		DeviceNo: user.DeviceNo,
		Platform: user.Platform,
	}); err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	user, err = model.GetUserByID(user.Id)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}

	_ = model.CreateUserRecord(model.UserRecord{
		BaseModel: model.BaseModel{
			CreateTime: chinaNow(),
		},
		UserId: user.Id,
		Remark: "退出账号:" + username,
	})

	returnUserInfo(c, user, 0)
}

// LogoffHandler 账号注销
// @Summary 账号注销
// @Description 配置命中的版本会真实注销账号，删除当前设备用户；已登录账号时会删除账号并让该账号所有设备下线，未命中版本只返回成功
// @Tags 账号
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} Response{result=LogoffResponse}
// @Router /logoff [post]
func LogoffHandler(c *gin.Context) {
	client := middleware.CurrentClientInfo(c)
	if !realLogoffVersionEnabled(client.Version) {
		JsonReturn(c, CodeSuccess, "success", nil)
		return
	}

	user, err := middleware.CurrentUser(c)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}

	newUser, err := realLogoffUser(c, user, client)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}

	returnLogoffInfo(c, newUser)
}

func realLogoffVersionEnabled(version string) bool {
	version = strings.TrimSpace(version)
	if version == "" {
		return false
	}
	return csvContainsExact(model.ConfigValue(model.ConfigRealLogoffVersions, ""), version)
}

func csvContainsExact(rule string, value string) bool {
	rule = strings.TrimSpace(rule)
	value = strings.TrimSpace(value)
	if rule == "" || value == "" {
		return false
	}
	for _, item := range strings.Split(rule, ",") {
		if strings.TrimSpace(item) == value {
			return true
		}
	}
	return false
}

func realLogoffUser(c *gin.Context, user model.User, client middleware.ClientInfo) (model.User, error) {
	var newUser model.User
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		username := strings.TrimSpace(user.Username)
		if username != "" {
			if err := tx.Where("username = ?", username).Delete(&model.User{}).Error; err != nil {
				return err
			}
			if err := tx.Where("username = ?", username).Delete(&model.UserAccount{}).Error; err != nil {
				return err
			}
		} else if err := tx.Where("id = ?", user.Id).Delete(&model.User{}).Error; err != nil {
			return err
		}

		params := buildRealLogoffGuestParams(user, client)
		newUser = buildGuestUser(c, params, nil, 1)
		if err := model.CreateUserWithRandomIDAndTransferCode(tx, &newUser, currentProductRouteCode()); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return model.User{}, err
	}
	_ = model.DeleteNodeOnlineStatus(user.Id)
	return reloadUser(newUser.Id)
}

func buildRealLogoffGuestParams(user model.User, client middleware.ClientInfo) autoLoginParams {
	platform := client.Platform
	if platform == "" {
		platform = user.Platform
	}
	version := client.Version
	if version == "" {
		version = user.Version
	}
	return autoLoginParams{
		DeviceNo:        user.DeviceNo,
		Platform:        platform,
		Version:         version,
		MobileName:      user.MobileName,
		MobileVersion:   user.MobileVersion,
		MobileModelName: user.MobileModelName,
		SourceChannel:   user.SourceChannel,
	}
}

func getUserAccountForLogin(tx *gorm.DB, username string) (model.UserAccount, error) {
	var account model.UserAccount
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("username = ?", username).First(&account).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.UserAccount{}, fmt.Errorf("account not found")
		}
		return model.UserAccount{}, err
	}
	return account, nil
}

func ensureAccountDeviceLimit(tx *gorm.DB, user model.User, username string) error {
	if user.Username == username && user.Type == 2 {
		return nil
	}

	maxDevices := configuredMaxLoginDevices(model.ConfigValue(model.ConfigMaxLoginDevices, "2"))
	if maxDevices <= 0 {
		return nil
	}

	var count int64
	if err := tx.Model(&model.User{}).
		Where("username = ?", username).
		Where("type = ?", 2).
		Where("status = ?", model.UserStatusNormal).
		Count(&count).Error; err != nil {
		return err
	}
	if count >= int64(maxDevices) {
		return fmt.Errorf("account device limit reached")
	}
	return nil
}

func configuredMaxLoginDevices(value string) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 2
	}
	maxDevices, err := strconv.Atoi(value)
	if err != nil {
		return 2
	}
	return maxDevices
}

// ChangePasswordHandler 修改密码
// @Summary 修改密码
// @Description 当前登录账号无需提交旧密码，直接设置新密码
// @Tags 账号
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ChangePasswordRequest true "修改密码请求"
// @Success 200 {object} Response
// @Router /change_password [post]
func ChangePasswordHandler(c *gin.Context) {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	if user.Type == 1 || user.Username == "" {
		JsonReturn(c, CodeError, "account not logged in", nil)
		return
	}

	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		JsonReturn(c, CodeError, "invalid json body", nil)
		return
	}

	params := getChangePasswordParams(body)
	if msg := validatePassword(params.NewPassword, "new_password"); msg != "" {
		JsonReturn(c, CodeError, msg, nil)
		return
	}

	account, err := model.GetUserAccountByUsername(user.Username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			JsonReturn(c, CodeError, "account not found", nil)
			return
		}
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	if account.Password == params.NewPassword {
		JsonReturn(c, CodeError, "new_password must be different", nil)
		return
	}

	if err := model.UpdateUserAccountByUsername(user.Username, map[string]interface{}{
		"password": params.NewPassword,
	}); err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}

	_ = model.CreateUserRecord(model.UserRecord{
		BaseModel: model.BaseModel{
			CreateTime: chinaNow(),
		},
		UserId: user.Id,
		Remark: "修改密码:" + user.Username,
	})

	JsonReturn(c, CodeSuccess, "success", nil)
}

// DeviceInfoHandler 获取设备信息
// @Summary 获取设备信息
// @Description 获取当前设备和当前账号下其它已登录设备
// @Tags 账号
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} Response{result=DeviceInfoResponse}
// @Router /device_info [post]
func DeviceInfoHandler(c *gin.Context) {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	deviceInfo, err := buildDeviceInfo(user)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	JsonReturn(c, CodeSuccess, "success", deviceInfo)
}

// DeviceLogoutHandler 移除设备
// @Summary 移除设备
// @Description 将当前账号下指定的其它设备移除登录并清空设备会员
// @Tags 账号
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body DeviceLogoutRequest true "移除设备请求"
// @Success 200 {object} Response{result=DeviceInfoResponse}
// @Router /device_logout [post]
func DeviceLogoutHandler(c *gin.Context) {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	if user.Type == 1 || user.Username == "" {
		JsonReturn(c, CodeError, "account not logged in", nil)
		return
	}

	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		JsonReturn(c, CodeError, "invalid json body", nil)
		return
	}

	params := getDeviceLogoutParams(body)
	if params.Id <= 0 {
		JsonReturn(c, CodeError, "id is required", nil)
		return
	}
	if params.Id == user.Id {
		JsonReturn(c, CodeError, "can not remove local device", nil)
		return
	}

	target, err := model.GetUserByID(params.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			JsonReturn(c, CodeError, "device not found", nil)
			return
		}
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	if target.Type != 2 || target.Username != user.Username {
		JsonReturn(c, CodeError, "invalid operation", nil)
		return
	}

	if err := model.UpdateUserFieldsByID(target.Id, map[string]interface{}{
		"username": "",
		"type":     1,
		"vip_time": nil,
	}); err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}

	_ = model.CreateUserRecord(model.UserRecord{
		BaseModel: model.BaseModel{
			CreateTime: chinaNow(),
		},
		UserId: user.Id,
		Remark: "移除设备:" + strconv.Itoa(target.Id) + ",当前登录账号:" + user.Username,
	})

	user, err = model.GetUserByID(user.Id)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	deviceInfo, err := buildDeviceInfo(user)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	JsonReturn(c, CodeSuccess, "success", deviceInfo)
}

// DrawInviteCodeHandler 获取邀请码
// @Summary 获取邀请码
// @Description 获取当前用户长期有效的邀请码，没有则自动生成
// @Tags 账号
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} Response{result=InviteCodeResponse}
// @Router /draw_invite_code [post]
func DrawInviteCodeHandler(c *gin.Context) {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	inviteCode, err := ensureInviteCode(user.Id)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	JsonReturn(c, CodeSuccess, "success", InviteCodeResponse{
		InviteCode: inviteCode.InviteCode,
	})
}

// InviteCodeHandler 填写邀请码
// @Summary 填写邀请码
// @Description 新用户在有效期内填写邀请码，给邀请人叠加会员奖励
// @Tags 账号
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body InviteCodeRequest true "填写邀请码请求"
// @Success 200 {object} Response
// @Router /invite_code [post]
func InviteCodeHandler(c *gin.Context) {
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
	params := getInviteCodeParams(body)
	if params.InviteCode == "" {
		JsonReturn(c, CodeError, "invite_code is required", nil)
		return
	}
	if user.InviterId != nil {
		JsonReturn(c, CodeError, "invite code already used", nil)
		return
	}
	if !inviteValid(user) {
		JsonReturn(c, CodeError, "invite code expired", nil)
		return
	}

	inviteCode, err := model.GetUserInviteCodeByCode(params.InviteCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			JsonReturn(c, CodeError, "invite code invalid", nil)
			return
		}
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	if inviteCode.UserId == user.Id {
		JsonReturn(c, CodeError, "can not use local invite code", nil)
		return
	}

	now := chinaNow()
	inviterId := inviteCode.UserId
	realReward := 0

	if err := model.DB.Transaction(func(tx *gorm.DB) error {
		var currentUser model.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", user.Id).
			Where("status = ?", model.UserStatusNormal).
			First(&currentUser).Error; err != nil {
			return err
		}
		if currentUser.InviterId != nil {
			return fmt.Errorf("invite code already used")
		}
		if !inviteValid(currentUser) {
			return fmt.Errorf("invite code expired")
		}

		var inviter model.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", inviterId).
			Where("status = ?", model.UserStatusNormal).
			First(&inviter).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("inviter not found")
			}
			return err
		}
		if inviter.Id == currentUser.Id {
			return fmt.Errorf("can not use local invite code")
		}
		if inviter.CreateTime.After(currentUser.CreateTime) {
			return fmt.Errorf("invite code not available")
		}

		nextShareCount := inviter.ShareCount + 1
		realReward, err = calculateInviteReward(nextShareCount)
		if err != nil {
			return err
		}
		realReward = limitInviteRewardByMaxReward(inviter.RewardTime, realReward)
		if realReward <= 0 {
			return fmt.Errorf("invite reward invalid")
		}
		inviterVipTime := addVipSeconds(inviter.VipTime, realReward)

		inviterIdValue := inviter.Id
		if err := tx.Model(&model.User{}).Where("id = ?", currentUser.Id).Updates(map[string]interface{}{
			"inviter_id": &inviterIdValue,
			"invited_at": now,
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.User{}).Where("id = ?", inviter.Id).Updates(map[string]interface{}{
			"vip_time":    inviterVipTime,
			"share_count": nextShareCount,
			"reward_time": inviter.RewardTime + realReward,
		}).Error; err != nil {
			return err
		}
		return syncInviterVipTime(tx, inviter, inviterVipTime)
	}); err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}

	_ = model.CreateUserRecord(model.UserRecord{
		BaseModel: model.BaseModel{
			CreateTime: now,
		},
		UserId: user.Id,
		Remark: "填写邀请码:" + params.InviteCode,
	})
	_ = model.CreateUserRecord(model.UserRecord{
		BaseModel: model.BaseModel{
			CreateTime: now,
		},
		UserId: inviterId,
		Remark: "邀请用户:" + strconv.Itoa(user.Id) + ",奖励秒数:" + strconv.Itoa(realReward),
	})

	JsonReturn(c, CodeSuccess, "success", nil)
}

// InviteDetailHandler 邀请详情
// @Summary 邀请详情
// @Description 获取当前用户的邀请码、填写资格和邀请奖励规则
// @Tags 账号
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} Response{result=InviteDetailResponse}
// @Router /invite_detail [post]
func InviteDetailHandler(c *gin.Context) {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	inviteCode, err := ensureInviteCode(user.Id)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	rewardConfig, milestones, err := inviteRewardDetail()
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	JsonReturn(c, CodeSuccess, "success", InviteDetailResponse{
		InviteCode:       inviteCode.InviteCode,
		InviteValid:      inviteValid(user),
		ShareCount:       user.ShareCount,
		RewardTime:       user.RewardTime,
		PerInviteSeconds: rewardConfig.PerInviteSeconds,
		PerInviteText:    rewardText(rewardConfig.PerInviteSeconds),
		Milestones:       milestones,
	})
}

// UserInfoHandler 用户信息
// @Summary 用户信息
// @Description 通过 Authorization Bearer Token 获取当前用户信息
// @Tags 账号
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} Response{result=UserInfoResponse}
// @Router /user_info [get]
func UserInfoHandler(c *gin.Context) {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	appStoreRegion := middleware.CurrentClientInfo(c).AppStoreRegion
	if appStoreRegion != "" && appStoreRegion != user.AppStoreRegion {
		if err := model.UpdateUserFieldsByID(user.Id, map[string]interface{}{"app_store_region": appStoreRegion}); err != nil {
			JsonReturn(c, CodeError, err.Error(), nil)
			return
		}
		user.AppStoreRegion = appStoreRegion
	}
	returnUserInfo(c, user, 0)
}
