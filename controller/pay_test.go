package controller

import (
	"testing"
	"time"

	"just-vpn/middleware"
	"just-vpn/model"
)

func TestBuildLaunchOrderUsesPaymentRequestIPAndRegion(t *testing.T) {
	clientInfo := middleware.ClientInfo{
		Platform: "iphone",
		TimeZone: "Europe/London",
	}
	createdAt := time.Now().In(time.Local).Truncate(time.Second)
	user := model.User{BaseModel: model.BaseModel{CreateTime: createdAt}}
	order := buildLaunchOrder(user, model.Package{}, payLaunchDecision{reason: "overseas_region:美国"}, "203.0.113.8", "0|美国|0|加州|洛杉矶|0", clientInfo)

	if order.Ip != "203.0.113.8" {
		t.Fatalf("order.Ip = %q, want payment request IP", order.Ip)
	}
	if order.RegIpRegion != "0|美国|0|加州|洛杉矶|0" {
		t.Fatalf("order.RegIpRegion = %q, want payment request region", order.RegIpRegion)
	}
	if order.PayReason != "overseas_region:美国" {
		t.Fatalf("order.PayReason = %q, want pay decision reason", order.PayReason)
	}
	if order.ClientTimeZone != "Europe/London" {
		t.Fatalf("order.ClientTimeZone = %q, want payment request time zone", order.ClientTimeZone)
	}
	if !order.RegTime.Equal(createdAt) {
		t.Fatalf("order.RegTime = %v, want %v", order.RegTime, createdAt)
	}
}

func TestShouldForceAppleByTimeZone(t *testing.T) {
	tests := []struct {
		name     string
		timeZone string
		want     bool
	}{
		{name: "Shanghai", timeZone: "Asia/Shanghai", want: false},
		{name: "Shanghai case insensitive", timeZone: " asia/shanghai ", want: false},
		{name: "Chongqing alias", timeZone: "Asia/Chongqing", want: false},
		{name: "Urumqi", timeZone: "Asia/Urumqi", want: false},
		{name: "Hong Kong", timeZone: "Asia/Hong_Kong", want: true},
		{name: "Taipei", timeZone: "Asia/Taipei", want: true},
		{name: "Singapore", timeZone: "Asia/Singapore", want: true},
		{name: "Los Angeles", timeZone: "America/Los_Angeles", want: true},
		{name: "missing", timeZone: "", want: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			matched, reason := shouldForceAppleByTimeZone(payContext{timeZone: test.timeZone})
			if matched != test.want {
				t.Fatalf("matched = %v, want %v", matched, test.want)
			}
			wantReason := "mainland_china_timezone"
			if test.want {
				wantReason = "non_mainland_timezone"
			}
			if reason != wantReason {
				t.Fatalf("reason = %q, want %q", reason, wantReason)
			}
		})
	}
}

func TestDecidePayLaunchPrioritizesNonMainlandTimeZone(t *testing.T) {
	pack := model.Package{AppleId: "vip_month"}
	decision, err := decidePayLaunch(payContext{
		pack:     pack,
		timeZone: "Europe/London",
	})
	if err != nil {
		t.Fatalf("decidePayLaunch() error = %v", err)
	}
	if decision.payType != PayLaunchTypeAppleIAP {
		t.Fatalf("decision.payType = %q, want %q", decision.payType, PayLaunchTypeAppleIAP)
	}
	if decision.target != pack.AppleId {
		t.Fatalf("decision.target = %q, want %q", decision.target, pack.AppleId)
	}
	if decision.reason != "non_mainland_timezone" {
		t.Fatalf("decision.reason = %q, want %q", decision.reason, "non_mainland_timezone")
	}
}
