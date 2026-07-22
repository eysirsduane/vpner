package controller

import (
	"net/http/httptest"
	"testing"
	"time"

	"just-vpn/model"

	"github.com/gin-gonic/gin"
)

func TestBuildLaunchOrderUsesPaymentRequestIPAndRegion(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	createdAt := time.Now().In(time.Local).Truncate(time.Second)
	user := model.User{BaseModel: model.BaseModel{CreateTime: createdAt}}
	order := buildLaunchOrder(c, user, model.Package{}, payLaunchDecision{reason: "overseas_region:美国"}, "203.0.113.8", "0|美国|0|加州|洛杉矶|0")

	if order.Ip != "203.0.113.8" {
		t.Fatalf("order.Ip = %q, want payment request IP", order.Ip)
	}
	if order.RegIpRegion != "0|美国|0|加州|洛杉矶|0" {
		t.Fatalf("order.RegIpRegion = %q, want payment request region", order.RegIpRegion)
	}
	if order.PayReason != "overseas_region:美国" {
		t.Fatalf("order.PayReason = %q, want pay decision reason", order.PayReason)
	}
	if !order.RegTime.Equal(createdAt) {
		t.Fatalf("order.RegTime = %v, want %v", order.RegTime, createdAt)
	}
}
