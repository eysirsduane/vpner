package model

import "testing"

func TestPreferredNodesForTrialStatus(t *testing.T) {
	nodes := []Node{
		{BaseModel: BaseModel{Id: 1}, IsTrial: 0},
		{BaseModel: BaseModel{Id: 2}, IsTrial: 1},
		{BaseModel: BaseModel{Id: 3}, IsTrial: 0},
	}

	trialNodes := preferredNodesForTrialStatus(nodes, true)
	if len(trialNodes) != 1 || trialNodes[0].Id != 2 {
		t.Fatalf("trial nodes = %#v, want node 2", trialNodes)
	}

	normalNodes := preferredNodesForTrialStatus(nodes, false)
	if len(normalNodes) != 2 || normalNodes[0].Id != 1 || normalNodes[1].Id != 3 {
		t.Fatalf("normal nodes = %#v, want nodes 1 and 3", normalNodes)
	}
}

func TestPreferredNodesForTrialStatusFallsBackToNormal(t *testing.T) {
	nodes := []Node{
		{BaseModel: BaseModel{Id: 1}, IsTrial: 0},
		{BaseModel: BaseModel{Id: 2}, IsTrial: 0},
	}

	got := preferredNodesForTrialStatus(nodes, true)
	if len(got) != 2 || got[0].Id != 1 || got[1].Id != 2 {
		t.Fatalf("fallback nodes = %#v, want normal nodes 1 and 2", got)
	}
}

func TestPreferredNodesForTrialStatusNeverReturnsTrialToNormalUser(t *testing.T) {
	nodes := []Node{{BaseModel: BaseModel{Id: 1}, IsTrial: 1}}
	if got := preferredNodesForTrialStatus(nodes, false); len(got) != 0 {
		t.Fatalf("normal user nodes = %#v, want empty", got)
	}
}

func TestNodeSubscriptionSyncKeepsLocalTrialFlag(t *testing.T) {
	items := []nodeSubscriptionItem{{
		IP:       "same.example.com",
		Content:  "vmess://new",
		Code:     "AUTO",
		CodeName: "自动",
		NodeType: "vmess",
	}}
	existing := []Node{{
		BaseModel: BaseModel{Id: 1},
		Address:   "same.example.com",
		LinkUrl:   "vmess://old",
		IsTrial:   1,
	}}

	plan := buildNodeSubscriptionSyncPlan(items, existing)
	if len(plan.Updates) != 1 {
		t.Fatalf("sync updates = %#v, want one update", plan.Updates)
	}
	if _, exists := plan.Updates[0].Fields["is_trial"]; exists {
		t.Fatalf("sync unexpectedly overwrites local is_trial: %#v", plan.Updates[0].Fields)
	}
}
