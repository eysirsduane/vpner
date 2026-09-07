package model

import "testing"

func TestPreferredNodesForTrialUserUsesRequiredRouteOrder(t *testing.T) {
	requested := []Node{
		{BaseModel: BaseModel{Id: 1}, IsTrial: 0},
		{BaseModel: BaseModel{Id: 2}, IsTrial: 1},
	}
	automatic := []Node{
		{BaseModel: BaseModel{Id: 3}, IsTrial: 0},
		{BaseModel: BaseModel{Id: 4}, IsTrial: 1},
	}

	tests := []struct {
		name      string
		requested []Node
		automatic []Node
		wantID    int
	}{
		{name: "trial requested country first", requested: requested, automatic: automatic, wantID: 2},
		{name: "trial automatic before normal requested country", requested: requested[:1], automatic: automatic, wantID: 4},
		{name: "normal requested after all trial routes", requested: requested[:1], automatic: automatic[:1], wantID: 1},
		{name: "normal automatic last", automatic: automatic[:1], wantID: 3},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := preferredNodesForTrialUser(test.requested, test.automatic)
			if len(got) == 0 || got[0].Id != test.wantID {
				t.Fatalf("preferred nodes = %#v, want node %d", got, test.wantID)
			}
		})
	}
}

func TestNodesForTrialStatusNeverReturnsTrialToNormalUser(t *testing.T) {
	nodes := []Node{
		{BaseModel: BaseModel{Id: 1}, IsTrial: 1},
		{BaseModel: BaseModel{Id: 2}, IsTrial: 0},
	}
	got := nodesForTrialStatus(nodes, false)
	if len(got) != 1 || got[0].Id != 2 {
		t.Fatalf("normal user nodes = %#v, want node 2", got)
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
