package controller

import "testing"

func TestLocalizedTextValue(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		language string
		want     string
	}{
		{name: "legacy plain chinese", value: "原有中文", language: "en", want: "原有中文"},
		{name: "chinese", value: `{"zh-Hans":"中文内容","en":"English content"}`, language: "zh-Hans", want: "中文内容"},
		{name: "english", value: `{"zh-Hans":"中文内容","en":"English content"}`, language: "en", want: "English content"},
		{name: "json language keys ignore case", value: `{"ZH-HANS":"中文内容","EN":"English content"}`, language: "en", want: "English content"},
		{name: "missing header defaults chinese", value: `{"zh-Hans":"中文内容","en":"English content"}`, want: "中文内容"},
		{name: "english falls back chinese", value: `{"zh-Hans":"中文内容"}`, language: "en", want: "中文内容"},
		{name: "chinese falls back english", value: `{"en":"English content"}`, language: "zh-Hans", want: "English content"},
		{name: "empty preferred falls back", value: `{"zh-Hans":"中文内容","en":""}`, language: "en", want: "中文内容"},
		{name: "unrelated json remains unchanged", value: `{"url":"https://example.com"}`, language: "en", want: `{"url":"https://example.com"}`},
		{name: "invalid json remains unchanged", value: `{"zh-Hans":`, language: "en", want: `{"zh-Hans":`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := localizedTextValue(test.value, test.language); got != test.want {
				t.Fatalf("localizedTextValue(%q, %q) = %q, want %q", test.value, test.language, got, test.want)
			}
		})
	}
}

func TestMultilingualTextValueRoundTrip(t *testing.T) {
	value := multilingualTextValue("中文内容", "English content")
	if got := localizedTextValue(value, "zh-Hans"); got != "中文内容" {
		t.Fatalf("Chinese value = %q", got)
	}
	if got := localizedTextValue(value, "en"); got != "English content" {
		t.Fatalf("English value = %q", got)
	}
}

func TestRewardTextUsesRequestLanguage(t *testing.T) {
	if got := rewardText(3600, "zh-Hans"); got != "1小时会员" {
		t.Fatalf("Chinese reward text = %q", got)
	}
	if got := rewardText(3600, "en"); got != "1-hour membership" {
		t.Fatalf("English reward text = %q", got)
	}
	if got := rewardText(3600, ""); got != "1小时会员" {
		t.Fatalf("default reward text = %q", got)
	}
}
