package controller

import (
	"encoding/base64"
	"testing"
	"time"

	"just-vpn/pkg/util"
)

func TestEncryptedNodeLinkURLEncryptsBase64EncodedJSONData(t *testing.T) {
	linkUrl := `{"server":"127.0.0.1","port":443}`

	encrypted, err := encryptedNodeLinkURLWithKey(linkUrl, defaultNodeLinkAESKey)
	if err != nil {
		t.Fatalf("encryptedNodeLinkURL returned error: %v", err)
	}

	decrypted := decryptNodeLinkForTest(t, encrypted)
	decoded, err := base64.StdEncoding.DecodeString(decrypted)
	if err != nil {
		t.Fatalf("base64 decode decrypted link failed: %v", err)
	}
	if string(decoded) != linkUrl {
		t.Fatalf("unexpected decoded link: %s", decoded)
	}
}

func TestEncryptedNodeLinkURLEncryptsBase64EncodedURLData(t *testing.T) {
	linkUrl := "vless://example.com:443"

	encrypted, err := encryptedNodeLinkURLWithKey(linkUrl, defaultNodeLinkAESKey)
	if err != nil {
		t.Fatalf("encryptedNodeLinkURL returned error: %v", err)
	}

	decrypted := decryptNodeLinkForTest(t, encrypted)
	decoded, err := base64.StdEncoding.DecodeString(decrypted)
	if err != nil {
		t.Fatalf("base64 decode decrypted link failed: %v", err)
	}
	if string(decoded) != linkUrl {
		t.Fatalf("unexpected decoded link: %s", decoded)
	}
}

func TestShouldRejectNodeForVipExpired(t *testing.T) {
	now := time.Now().In(time.Local)
	expired := now.Add(-time.Second)
	valid := now.Add(time.Hour)

	tests := []struct {
		name    string
		code    string
		vipTime *time.Time
		want    bool
	}{
		{name: "paid line without vip", code: "HK", vipTime: nil, want: true},
		{name: "paid line with expired vip", code: "HK", vipTime: &expired, want: true},
		{name: "paid line with valid vip", code: "HK", vipTime: &valid, want: false},
		{name: "free line without vip", code: "FREE", vipTime: nil, want: false},
		{name: "free prefixed line with expired vip", code: "FREE_HK", vipTime: &expired, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldRejectNodeForVipExpired(tt.code, tt.vipTime)
			if got != tt.want {
				t.Fatalf("shouldRejectNodeForVipExpired(%q) = %v, want %v", tt.code, got, tt.want)
			}
		})
	}
}

func decryptNodeLinkForTest(t *testing.T, encrypted string) string {
	t.Helper()

	decrypted, err := util.AesDeCode(encrypted, []byte(defaultNodeLinkAESKey))
	if err != nil {
		t.Fatalf("decrypt node link failed: %v", err)
	}
	return string(decrypted)
}
