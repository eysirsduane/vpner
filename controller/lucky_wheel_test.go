package controller

import (
	"encoding/json"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"just-vpn/middleware"
	"just-vpn/model"
	"just-vpn/pkg/mapping"

	"github.com/gin-gonic/gin"
)

func TestIsLuckyWheelNewUser(t *testing.T) {
	now := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name      string
		createdAt time.Time
		want      bool
	}{
		{"just registered", now, true},
		{"within 24 hours", now.Add(-23 * time.Hour), true},
		{"exactly 24 hours", now.Add(-24 * time.Hour), true},
		{"expired", now.Add(-24*time.Hour - time.Second), false},
		{"missing creation time", time.Time{}, false},
		{"future creation time", now.Add(time.Second), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := isLuckyWheelNewUser(test.createdAt, now); got != test.want {
				t.Fatalf("isLuckyWheelNewUser() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestLuckyWheelPlayRejectsExistingUserBeforeQuery(t *testing.T) {
	oldDB := model.DB
	model.DB = nil
	t.Cleanup(func() { model.DB = oldDB })

	router := gin.New()
	path := mapping.Endpoint("/api/v1/lucky_wheel_play")
	router.POST(path, func(c *gin.Context) {
		c.Set(middleware.ContextUserKey, model.User{
			BaseModel: model.BaseModel{Id: 1, CreateTime: chinaNow().Add(-48 * time.Hour)},
		})
		LuckyWheelPlayHandler(c)
	})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest("POST", path, nil))
	var response map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	expected := mapping.MapResponse("/api/v1/lucky_wheel_play", map[string]interface{}{
		"code":   CodeError,
		"msg":    "幸运转盘仅限新用户参与",
		"result": map[string]interface{}{},
	})
	expectedJSON, err := json.Marshal(expected)
	if err != nil {
		t.Fatal(err)
	}
	var expectedResponse map[string]interface{}
	if err := json.Unmarshal(expectedJSON, &expectedResponse); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(response, expectedResponse) {
		t.Fatalf("response = %#v, want %#v", response, expectedResponse)
	}
	if recorder.Code != 200 {
		t.Fatalf("HTTP status = %d", recorder.Code)
	}
}
