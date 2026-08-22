package errmsg

import "testing"

func TestFriendly(t *testing.T) {
	tests := []struct {
		name string
		msg  string
		want string
	}{
		{name: "invalid body", msg: "invalid json body", want: "请求参数格式不正确"},
		{name: "auth missing", msg: "authorization token is required", want: "请先登录后再操作"},
		{name: "auth user missing", msg: "login user not found", want: "登录状态已失效，请重新登录"},
		{name: "record not found", msg: "record not found", want: "数据不存在"},
		{name: "required mapped header", msg: "X-App-Version is required", want: "版本号不能为空"},
		{name: "mysql duplicate", msg: "Error 1062: Duplicate entry", want: "数据已存在，请勿重复提交"},
		{name: "system connection refused", msg: "dial tcp 127.0.0.1:3306: connection refused", want: "系统连接失败，请稍后再试"},
		{name: "unknown", msg: "some internal error", want: "系统错误，请稍后再试"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Friendly(tt.msg); got != tt.want {
				t.Fatalf("Friendly(%q) = %q, want %q", tt.msg, got, tt.want)
			}
		})
	}
}

func TestFriendlyForLanguage(t *testing.T) {
	if got := FriendlyForLanguage("package not found", "EN"); got != "The selected plan is unavailable." {
		t.Fatalf("FriendlyForLanguage() = %q", got)
	}
	if got := FriendlyForLanguage("package not found", "Zh-CN"); got != "套餐不存在或已下架" {
		t.Fatalf("FriendlyForLanguage() = %q", got)
	}
	if got := FriendlyForLanguage("some internal error", "fr"); got != "System error. Please try again later." {
		t.Fatalf("FriendlyForLanguage() = %q", got)
	}
	if got := Localize("会员已过期，请断开连接", "EN-us"); got != "Membership has expired. Please disconnect." {
		t.Fatalf("Localize() = %q", got)
	}
}

func TestEnglishMessagesCoverFriendlyMessages(t *testing.T) {
	messages := map[string]bool{defaultErrorMessage: true}
	for _, message := range exactMessages {
		messages[message] = true
	}
	for _, item := range containsMessages {
		messages[item.msg] = true
	}
	for field := range map[string]bool{
		"platform": true, "version": true, "build": true, "device_no": true,
		"X-Platform": true, "X-Version": true, "X-Build": true,
		"username": true, "password": true, "old_password": true, "new_password": true,
		"package_id": true, "transaction_id": true, "code": true, "flow": true,
		"invite_code": true, "id": true, "unknown": true,
	} {
		messages[requiredMessage(field+" is required")] = true
	}
	for message := range messages {
		if _, ok := englishMessages[message]; !ok {
			t.Errorf("missing English translation for %q", message)
		}
	}
}
