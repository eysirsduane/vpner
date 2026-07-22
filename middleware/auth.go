package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"just-vpn/model"
	"just-vpn/pkg/errmsg"
	"just-vpn/pkg/jwt"
	"just-vpn/pkg/mapping"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	CodeError      = 500
	ContextUserKey = "current_user"
)

func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := authUser(c)
		if err != nil {
			jsonReturn(c, CodeError, err.Error(), nil)
			c.Abort()
			return
		}

		c.Set(ContextUserKey, user)
		_ = model.RecordDailyActive(user.Id)
		c.Next()
	}
}

func CurrentUser(c *gin.Context) (model.User, error) {
	value, ok := c.Get(ContextUserKey)
	if !ok {
		return model.User{}, fmt.Errorf("current user missing")
	}
	user, ok := value.(model.User)
	if !ok {
		return model.User{}, fmt.Errorf("current user invalid")
	}
	return user, nil
}

func authUser(c *gin.Context) (model.User, error) {
	token := bearerToken(c)
	if token == "" {
		return model.User{}, fmt.Errorf("authorization token is required")
	}
	claims, err := jwt.Parse(token)
	if err != nil {
		return model.User{}, err
	}
	banned, err := model.IsUserBannedInCache(claims.UserId, claims.DeviceNo)
	if err != nil {
		return model.User{}, fmt.Errorf("ban check failed")
	}
	if banned {
		return model.User{}, fmt.Errorf("user is banned")
	}
	user, err := model.GetUserByID(claims.UserId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.User{}, fmt.Errorf("login user not found")
		}
		return model.User{}, err
	}
	if user.DeviceNo != claims.DeviceNo {
		return model.User{}, fmt.Errorf("token device invalid")
	}
	if user.Status == model.UserStatusBanned {
		return model.User{}, fmt.Errorf("user is banned")
	}
	return user, nil
}

func bearerToken(c *gin.Context) string {
	auth := strings.TrimSpace(c.GetHeader("Authorization"))
	if auth == "" {
		return ""
	}
	parts := strings.Fields(auth)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	return ""
}

func jsonReturn(c *gin.Context, code int, msg string, result interface{}) {
	if result == nil {
		result = gin.H{}
	}
	if code != http.StatusOK {
		msg = errmsg.Friendly(msg)
	}
	originalPath := mapping.OriginalEndpoint(c.FullPath())
	response := map[string]interface{}{
		"code":   code,
		"msg":    msg,
		"result": result,
	}
	c.JSON(http.StatusOK, mapping.MapResponse(originalPath, response))
}
