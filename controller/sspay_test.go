package controller

import (
	"github.com/gin-gonic/gin"
	"just-vpn/model"
	"just-vpn/pkg/pay/sspay"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestSSPayCallbackPreservesSignedValues(t *testing.T) {
	p := map[string]string{"name": " VIP + &会员 ", "param": " ", "money": "1.00", "sign_type": "MD5"}
	p["sign"] = sspay.Sign(p, "secret")
	q := url.Values{}
	for k, v := range p {
		q.Set(k, v)
	}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/?"+q.Encode(), nil)
	got := collectPayNotifyParams(c)
	if got["name"] != p["name"] || got["param"] != " " || !sspay.VerifySign(got, "secret") {
		t.Fatalf("callback altered: %v", got)
	}
}

func TestSSPayProviderDecision(t *testing.T) {
	original := payProviderConfigValue
	t.Cleanup(func() { payProviderConfigValue = original })
	for _, provider := range []string{"xxpay", "sspay"} {
		payProviderConfigValue = func(code, fallback string) string {
			if code == model.PayConfigThirdPayProvider {
				return provider
			}
			return fallback
		}
		d := h5PayDecision("allowed")
		want := PayLaunchTypeH5
		if provider == "sspay" {
			want = PayLaunchTypeSSPay
		}
		if d.payType != want || d.reason != "allowed" {
			t.Fatalf("wrong decision: %+v", d)
		}
	}
}

func TestSSPayDevice(t *testing.T) {
	for platform, want := range map[string]string{"iphone": "mobile", "iOS": "mobile", "ipad": "mobile", "android": "mobile", "windows": "pc", "": "pc"} {
		if got := sspayDevice(platform); got != want {
			t.Errorf("%s: %s", platform, got)
		}
	}
}

func TestYuanToFen(t *testing.T) {
	for value, want := range map[string]int{"1": 100, "1.2": 120, "0.01": 1, "99.99": 9999} {
		got, err := YuanToFen(value)
		if err != nil || got != want {
			t.Errorf("%s: %d %v", value, got, err)
		}
	}
	for _, value := range []string{"", "-1", "1.234", "NaN", "1e2", "99999999999999999999999999"} {
		if _, err := YuanToFen(value); err == nil {
			t.Errorf("accepted %q", value)
		}
	}
}
