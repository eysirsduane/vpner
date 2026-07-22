package controller

import (
	"sort"
	"time"

	"just-vpn/middleware"
	"just-vpn/model"

	"github.com/gin-gonic/gin"
)

type ReadNoticeRequest struct {
	SystemIds []int `json:"system_ids" example:"1,2"`
	UserIds   []int `json:"user_ids" example:"3,4"`
}

type NoticeItemResponse struct {
	Id         int    `json:"id" example:"1"`
	Title      string `json:"title" example:"系统维护通知"`
	Content    string `json:"content" example:"今晚 02:00-03:00 进行维护"`
	IsRead     int    `json:"is_read" example:"0"`
	CreateTime string `json:"create_time" example:"2026-07-05 18:30:00"`
}

type NoticeGroupResponse struct {
	List        []NoticeItemResponse `json:"list"`
	Total       int                  `json:"total" example:"1"`
	UnreadCount int64                `json:"unread_count" example:"1"`
}

type NoticeResponse struct {
	System NoticeGroupResponse `json:"system"`
	User   NoticeGroupResponse `json:"user"`
}

type sortableNoticeItem struct {
	NoticeItemResponse
	priority   int
	createTime time.Time
}

// NoticeHandler 获取通知列表
// @Summary 获取通知列表
// @Description 全量获取当前用户可见的系统通知和个人通知
// @Tags 通知
// @Produce json
// @Security BearerAuth
// @Success 200 {object} Response{result=NoticeResponse}
// @Router /notice [get]
func NoticeHandler(c *gin.Context) {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}

	result, err := buildNoticeResponse(user.Id, currentNoticePlatform(c), false)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}

	JsonReturn(c, CodeSuccess, "success", result)
}

// UnreadNoticeHandler 获取未读通知列表
// @Summary 获取未读通知列表
// @Description 增量获取当前用户未读的系统通知和个人通知
// @Tags 通知
// @Produce json
// @Security BearerAuth
// @Success 200 {object} Response{result=NoticeResponse}
// @Router /notice/unread [get]
func UnreadNoticeHandler(c *gin.Context) {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}

	result, err := buildNoticeResponse(user.Id, currentNoticePlatform(c), true)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}

	JsonReturn(c, CodeSuccess, "success", result)
}

// ReadNoticeHandler 批量标记通知已读
// @Summary 批量标记通知已读
// @Description 根据 system_ids 和 user_ids 批量标记通知已读
// @Tags 通知
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ReadNoticeRequest true "批量标记通知已读请求"
// @Success 200 {object} Response
// @Router /read_notice [post]
func ReadNoticeHandler(c *gin.Context) {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}

	var req ReadNoticeRequest
	if err := BindMappedJSON(c, &req); err != nil {
		JsonReturn(c, CodeError, "invalid json body", nil)
		return
	}
	if hasInvalidNoticeIds(req.SystemIds) || hasInvalidNoticeIds(req.UserIds) {
		JsonReturn(c, CodeError, "notice id must be greater than 0", nil)
		return
	}

	platform := currentNoticePlatform(c)
	if err := model.MarkSystemNoticesRead(user.Id, uniqueNoticeIds(req.SystemIds), platform); err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	if err := model.MarkUserNoticesRead(user.Id, uniqueNoticeIds(req.UserIds)); err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}

	JsonReturn(c, CodeSuccess, "success", nil)
}

func buildNoticeResponse(userId int, platform string, onlyUnread bool) (NoticeResponse, error) {
	systemNotices, err := noticeSystemRows(userId, platform, onlyUnread)
	if err != nil {
		return NoticeResponse{}, err
	}
	userNotices, err := noticeUserRows(userId, onlyUnread)
	if err != nil {
		return NoticeResponse{}, err
	}
	systemUnreadCount, err := model.CountUnreadSystemNotices(userId, platform)
	if err != nil {
		return NoticeResponse{}, err
	}
	userUnreadCount, err := model.CountUnreadUserNotices(userId)
	if err != nil {
		return NoticeResponse{}, err
	}

	return NoticeResponse{
		System: NoticeGroupResponse{
			List:        systemNotices,
			Total:       len(systemNotices),
			UnreadCount: systemUnreadCount,
		},
		User: NoticeGroupResponse{
			List:        userNotices,
			Total:       len(userNotices),
			UnreadCount: userUnreadCount,
		},
	}, nil
}

func noticeSystemRows(userId int, platform string, onlyUnread bool) ([]NoticeItemResponse, error) {
	var notices []model.SystemNotice
	var err error
	if onlyUnread {
		notices, err = model.GetUnreadSystemNotices(userId, platform)
	} else {
		notices, err = model.GetAllVisibleSystemNotices(platform)
	}
	if err != nil {
		return nil, err
	}

	readMap := map[int]bool{}
	if !onlyUnread {
		readMap, err = model.GetReadSystemNoticeIDMap(userId, collectSystemNoticeIds(notices))
		if err != nil {
			return nil, err
		}
	}

	items := make([]sortableNoticeItem, 0, len(notices))
	for _, notice := range notices {
		isRead := 0
		if readMap[notice.Id] {
			isRead = 1
		}
		items = append(items, sortableNoticeItem{
			NoticeItemResponse: NoticeItemResponse{
				Id:         notice.Id,
				Title:      notice.Title,
				Content:    notice.Content,
				IsRead:     isRead,
				CreateTime: notice.CreateTime.Format("2006-01-02 15:04:05"),
			},
			priority:   notice.Priority,
			createTime: notice.CreateTime,
		})
	}
	return sortNoticeItems(items), nil
}

func noticeUserRows(userId int, onlyUnread bool) ([]NoticeItemResponse, error) {
	var notices []model.UserNotice
	var err error
	if onlyUnread {
		notices, err = model.GetUnreadUserNotices(userId)
	} else {
		notices, err = model.GetVisibleUserNotices(userId)
	}
	if err != nil {
		return nil, err
	}

	items := make([]sortableNoticeItem, 0, len(notices))
	for _, notice := range notices {
		isRead := 0
		if notice.Status == model.UserNoticeStatusRead {
			isRead = 1
		}
		items = append(items, sortableNoticeItem{
			NoticeItemResponse: NoticeItemResponse{
				Id:         notice.Id,
				Title:      notice.Title,
				Content:    notice.Content,
				IsRead:     isRead,
				CreateTime: notice.CreateTime.Format("2006-01-02 15:04:05"),
			},
			priority:   notice.Priority,
			createTime: notice.CreateTime,
		})
	}
	return sortNoticeItems(items), nil
}

func sortNoticeItems(items []sortableNoticeItem) []NoticeItemResponse {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].priority != items[j].priority {
			return items[i].priority > items[j].priority
		}
		return items[i].createTime.After(items[j].createTime)
	})
	result := make([]NoticeItemResponse, 0, len(items))
	for _, item := range items {
		result = append(result, item.NoticeItemResponse)
	}
	return result
}

func currentNoticePlatform(c *gin.Context) string {
	platform := middleware.CurrentClientInfo(c).Platform
	if platform == "" {
		return "all"
	}
	return platform
}

func collectSystemNoticeIds(notices []model.SystemNotice) []int {
	ids := make([]int, 0, len(notices))
	for _, notice := range notices {
		ids = append(ids, notice.Id)
	}
	return ids
}

func hasInvalidNoticeIds(ids []int) bool {
	for _, id := range ids {
		if id <= 0 {
			return true
		}
	}
	return false
}

func uniqueNoticeIds(ids []int) []int {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[int]bool, len(ids))
	result := make([]int, 0, len(ids))
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		result = append(result, id)
	}
	return result
}
