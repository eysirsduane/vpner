package model

import "testing"

func TestParseNodeSubscription(t *testing.T) {
	content := "\n# 节点列表\n vless://first@example.com:443#first \nanytls://password@1.2.3.4:443?security=tls#second\nvless://first@example.com:443#first\ninvalid-node\n"

	got := parseNodeSubscription(content)
	if len(got) != 2 {
		t.Fatalf("parseNodeSubscription() count = %d, want 2", len(got))
	}
	if got[0] != "vless://first@example.com:443#first" || got[1] != "anytls://password@1.2.3.4:443?security=tls#second" {
		t.Fatalf("parseNodeSubscription() = %#v", got)
	}
}

func TestNodeSubscriptionMetadata(t *testing.T) {
	nodeType, address, name := nodeSubscriptionMetadata("anytls://password@1.2.3.4:443?security=tls#香港节点")
	if nodeType != "anytls" || address != "1.2.3.4" || name != "香港节点" {
		t.Fatalf("nodeSubscriptionMetadata() = (%q, %q, %q)", nodeType, address, name)
	}
}

func TestShouldReplacePulledNodes(t *testing.T) {
	for _, nodeCount := range []int{0, 1, 4} {
		if shouldReplacePulledNodes(nodeCount) {
			t.Fatalf("shouldReplacePulledNodes(%d) = true, want false", nodeCount)
		}
	}
	for _, nodeCount := range []int{5, 6} {
		if !shouldReplacePulledNodes(nodeCount) {
			t.Fatalf("shouldReplacePulledNodes(%d) = false, want true", nodeCount)
		}
	}
}

func TestParseNodeSubscriptionResponseFromDispatch(t *testing.T) {
	nodes, err := parseNodeSubscriptionResponse([]byte(`{
  "code": 200,
  "message": "success",
  "data": {
    "nodes": [{
      "ip": "47.245.116.113",
      "content": "vmess://example",
      "code": "HK",
      "code_name": "香港",
      "node_type": "vmess"
    }]
  }
}`))
	if err != nil || len(nodes) != 1 {
		t.Fatalf("parseNodeSubscriptionResponse() = %#v, %v", nodes, err)
	}
	got := nodes[0]
	if got.IP != "47.245.116.113" || got.Content != "vmess://example" || got.Code != "HK" || got.CodeName != "香港" || got.NodeType != "vmess" {
		t.Fatalf("unexpected node: %#v", got)
	}
}
