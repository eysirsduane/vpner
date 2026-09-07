package sspay

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"testing"
)

func TestSignDocumentRules(t *testing.T) {
	p := map[string]string{"c": "d", "a": "b", "empty": "", "sign_type": "MD5", "sign": "ignored"}
	const expected = "1ab8f8f68f35c55d22cc13e912e0a022"
	if got := Sign(p, "secret"); got != expected {
		t.Fatalf("sign = %s", got)
	}
	p["sign"] = expected
	if !VerifySign(p, "secret") {
		t.Fatal("valid signature rejected")
	}
	p["a"] = "changed"
	if VerifySign(p, "secret") {
		t.Fatal("tampered signature accepted")
	}
}

func TestCreateOrderProtocol(t *testing.T) {
	for _, field := range []string{"payurl", "qrcode", "urlscheme"} {
		t.Run(field, func(t *testing.T) {
			value := "https://example.com/pay"
			if field == "qrcode" {
				value = "weixin://wxpay/bizpayurl?pr=123"
			}
			if field == "urlscheme" {
				value = "weixin://dl/business/?ticket=123"
			}
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" || r.URL.Path != "/mapi.php" {
					t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
				}
				if err := r.ParseForm(); err != nil {
					t.Error(err)
				}
				if r.PostForm.Get("name") != " VIP + &会员 " || r.PostForm.Get("return_url") != "https://example.com/return?a=1&b=2" || r.PostForm.Get("device") != "pc" {
					t.Errorf("wrong form: %v", r.PostForm)
				}
				assertFormSign(t, r.PostForm)
				fmt.Fprintf(w, `{"code":1,"trade_no":"T1",%q:%q}`, field, value)
			}))
			defer srv.Close()
			info, err := CreateOrder(CreateOrderRequest{APIURL: srv.URL, Pid: "1001", OutTradeNo: "O1", NotifyURL: "https://example.com/notify", ReturnURL: "https://example.com/return?a=1&b=2", Name: " VIP + &会员 ", Money: "1.00", ClientIP: "127.0.0.1", Key: "secret"})
			if err != nil {
				t.Fatal(err)
			}
			if info.PayOrderId != "T1" {
				t.Fatalf("wrong order: %+v", info)
			}
			if field == "qrcode" {
				u, err := url.Parse(info.PayURL)
				if err != nil {
					t.Fatal(err)
				}
				if u.Path != "/submit.php" || u.Query().Get("out_trade_no") != "O1" {
					t.Fatalf("wrong checkout: %s", info.PayURL)
				}
				assertFormSign(t, u.Query())
			} else if info.PayURL != value {
				t.Fatalf("url = %q", info.PayURL)
			}
		})
	}
}

func assertFormSign(t *testing.T, form url.Values) {
	t.Helper()
	var parts []string
	for k, v := range form {
		if k != "sign" && k != "sign_type" && v[0] != "" {
			parts = append(parts, k+"="+v[0])
		}
	}
	sort.Strings(parts)
	want := fmt.Sprintf("%x", md5.Sum([]byte(strings.Join(parts, "&")+"secret")))
	if form.Get("sign") != want {
		t.Errorf("form signature mismatch")
	}
}

func TestQueryOrderNumericResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if r.Method != "GET" || r.URL.Path != "/api.php" || q.Get("act") != "order" || q.Get("pid") != "1001" || q.Get("key") != "secret" || q.Get("out_trade_no") != "O1" || q.Has("sign") {
			t.Errorf("unexpected query: %s", r.URL)
		}
		fmt.Fprint(w, `{"code":1,"pid":1001,"trade_no":"T1","out_trade_no":"O1","status":1,"money":"1.00"}`)
	}))
	defer srv.Close()
	info, err := QueryOrder(srv.URL, "1001", "O1", "secret")
	if err != nil {
		t.Fatal(err)
	}
	if info.Pid != 1001 || info.Status != 1 || info.Money != "1.00" {
		t.Fatalf("wrong response: %+v", info)
	}
}
