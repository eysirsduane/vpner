package controller

import (
	"testing"

	"just-vpn/model"
)

func TestReviewNodeFromConfig(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		code     string
		versions string
		link     string
		wantOK   bool
	}{
		{name: "single version matched", version: "1.0.0", code: "hk", versions: "1.0.0", link: "vless://review.example:443", wantOK: true},
		{name: "comma version matched", version: "1.0.1", code: "US", versions: "1.0.0, 1.0.1", link: "vless://review.example:443", wantOK: true},
		{name: "version not matched", version: "1.0.2", code: "US", versions: "1.0.0,1.0.1", link: "vless://review.example:443", wantOK: false},
		{name: "empty versions disables review node", version: "1.0.0", code: "US", versions: "", link: "vless://review.example:443", wantOK: false},
		{name: "empty link disables review node", version: "1.0.0", code: "US", versions: "1.0.0", link: "", wantOK: false},
		{name: "empty request version does not match", version: "", code: "US", versions: "1.0.0", link: "vless://review.example:443", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, ok := reviewNodeFromConfig(tt.version, tt.code, tt.versions, tt.link)
			if ok != tt.wantOK {
				t.Fatalf("reviewNodeFromConfig() ok = %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if node.Id != 0 {
				t.Fatalf("review node id = %d, want 0", node.Id)
			}
			if node.Status != model.NodeStatusEnabled {
				t.Fatalf("review node status = %d, want %d", node.Status, model.NodeStatusEnabled)
			}
			if node.Code != "HK" && tt.code == "hk" {
				t.Fatalf("review node code = %q, want HK", node.Code)
			}
			if node.LinkUrl != tt.link {
				t.Fatalf("review node link = %q, want %q", node.LinkUrl, tt.link)
			}
		})
	}
}
