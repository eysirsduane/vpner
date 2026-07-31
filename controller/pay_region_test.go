package controller

import "testing"

func TestShouldForceAppleByOverseasCanBeDisabled(t *testing.T) {
	original := payOverseasIPConfigValue
	payOverseasIPConfigValue = func(string, string) string { return "0" }
	t.Cleanup(func() { payOverseasIPConfigValue = original })

	matched, reason := shouldForceAppleByOverseas(payContext{regions: "995|美国|0|加利福尼亚|洛杉矶|谷歌"})
	if matched {
		t.Fatal("matched = true, want false when overseas IP Apple-only switch is disabled")
	}
	if reason != "overseas_apple_only_disabled" {
		t.Fatalf("reason = %q, want %q", reason, "overseas_apple_only_disabled")
	}
}

func TestIPRegionCountry(t *testing.T) {
	testCases := []struct {
		name    string
		region  string
		country string
	}{
		{name: "中国大陆", region: "995|中国|0|上海|上海市|电信", country: "中国"},
		{name: "海外", region: "995|美国|0|加利福尼亚|洛杉矶|谷歌", country: "美国"},
		{name: "未知地区", region: "995|0|0|0|0|0", country: "0"},
		{name: "空地区", region: "", country: ""},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			country := ipRegionCountry(testCase.region)
			if country != testCase.country {
				t.Fatalf("country = %q, want %q", country, testCase.country)
			}
		})
	}
}
