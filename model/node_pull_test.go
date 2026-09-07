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
	for _, nodeCount := range []int{0, 1} {
		if shouldReplacePulledNodes(nodeCount) {
			t.Fatalf("shouldReplacePulledNodes(%d) = true, want false", nodeCount)
		}
	}
	for _, nodeCount := range []int{2, 4, 5, 6} {
		if !shouldReplacePulledNodes(nodeCount) {
			t.Fatalf("shouldReplacePulledNodes(%d) = false, want true", nodeCount)
		}
	}
}

func TestNormalizeNodeSubscriptionItemsKeepsLatestItemByAddress(t *testing.T) {
	nodes := normalizeNodeSubscriptionItems([]nodeSubscriptionItem{
		{IP: " SAME.EXAMPLE.COM ", Content: "vmess://old", Code: "AUTO", CodeName: "自动", NodeType: "vmess"},
		{IP: "other.example.com", Content: "vmess://other", Code: "AUTO", CodeName: "自动", NodeType: "vmess"},
		{IP: "same.example.com", Content: "vmess://new", Code: "AUTO", CodeName: "自动", NodeType: "vmess"},
	})
	if len(nodes) != 2 {
		t.Fatalf("normalizeNodeSubscriptionItems() count = %d, want 2", len(nodes))
	}
	if nodes[0].IP != "same.example.com" || nodes[0].Content != "vmess://new" {
		t.Fatalf("same address item = %#v, want latest content", nodes[0])
	}
	if nodes[1].IP != "other.example.com" {
		t.Fatalf("other address item = %#v", nodes[1])
	}
}

func TestBuildNodeSubscriptionSyncPlanMatchesAddressAndUpdatesAllFields(t *testing.T) {
	nodes := []nodeSubscriptionItem{
		{IP: "same.example.com", Content: "anytls://new", Code: "US", CodeName: "美国", NodeType: "anytls", IncludeAuto: 1},
		{IP: "fresh.example.com", Content: "vmess://fresh", Code: "AUTO", CodeName: "自动", NodeType: "vmess"},
	}
	existing := []Node{
		{BaseModel: BaseModel{Id: 5}, Code: "AUTO", CodeName: "自动", Name: "自动节点", NodeType: "vmess", LinkUrl: "vmess://old", Address: "SAME.EXAMPLE.COM", Status: NodeStatusEnabled},
		{BaseModel: BaseModel{Id: 9}, Code: "HK", CodeName: "香港", Name: "重复节点", NodeType: "vmess", LinkUrl: "vmess://duplicate", Address: "same.example.com", Status: NodeStatusEnabled},
	}

	plan := buildNodeSubscriptionSyncPlan(nodes, existing)
	if len(plan.ActivateIDs) != 1 || plan.ActivateIDs[0] != 5 {
		t.Fatalf("ActivateIDs = %#v, want [5]", plan.ActivateIDs)
	}
	if len(plan.DuplicateIDs) != 1 || plan.DuplicateIDs[0] != 9 {
		t.Fatalf("DuplicateIDs = %#v, want [9]", plan.DuplicateIDs)
	}
	if len(plan.Updates) != 1 || plan.Updates[0].ID != 5 {
		t.Fatalf("Updates = %#v, want one update for node 5", plan.Updates)
	}
	fields := plan.Updates[0].Fields
	wantFields := map[string]interface{}{
		"code":         "US",
		"code_name":    "美国",
		"include_auto": 1,
		"node_type":    "anytls",
		"link_url":     "anytls://new",
		"address":      "same.example.com",
		"name":         "美国节点",
	}
	for key, want := range wantFields {
		if got := fields[key]; got != want {
			t.Fatalf("Updates[0].Fields[%q] = %#v, want %#v", key, got, want)
		}
	}
	if len(plan.NewNodes) != 1 || plan.NewNodes[0].Address != "fresh.example.com" || plan.NewNodes[0].LinkUrl != "vmess://fresh" {
		t.Fatalf("NewNodes = %#v, want fresh node", plan.NewNodes)
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
      "node_type": "vmess",
	  "include_auto": 1,
	  "is_trial": 1
    }]
  }
}`))
	if err != nil || len(nodes) != 1 {
		t.Fatalf("parseNodeSubscriptionResponse() = %#v, %v", nodes, err)
	}
	got := nodes[0]
	if got.IP != "47.245.116.113" || got.Content != "vmess://example" || got.Code != "HK" || got.CodeName != "香港" || got.NodeType != "vmess" || got.IncludeAuto != 1 || got.IsTrial == nil || *got.IsTrial != 1 {
		t.Fatalf("unexpected node: %#v", got)
	}
}

func TestNodeForRequestedCodeUsesAutomaticRouteWithoutChangingPhysicalCountry(t *testing.T) {
	node := Node{Code: "HK", CodeName: "香港", Name: "香港节点", IncludeAuto: 1}
	got := nodeForRequestedCode(node, "AUTO")
	if got.Code != "AUTO" || got.CodeName != nodeSubscriptionName || got.Name != nodeSubscriptionName+"节点" {
		t.Fatalf("automatic route node = %#v", got)
	}
	if node.Code != "HK" || node.CodeName != "香港" {
		t.Fatalf("physical node was mutated: %#v", node)
	}
}
