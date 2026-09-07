package model

import "testing"

func TestNodeSubscriptionSyncAppliesProvidedTrialFlag(t *testing.T) {
	trial := 1
	items := []nodeSubscriptionItem{{
		IP:       "same.example.com",
		Content:  "vmess://new",
		Code:     "AUTO",
		CodeName: "自动",
		NodeType: "vmess",
		IsTrial:  &trial,
	}}
	existing := []Node{{
		BaseModel: BaseModel{Id: 1},
		Address:   "same.example.com",
		LinkUrl:   "vmess://old",
		IsTrial:   0,
	}}

	plan := buildNodeSubscriptionSyncPlan(items, existing)
	if len(plan.Updates) != 1 {
		t.Fatalf("sync updates = %#v, want one update", plan.Updates)
	}
	if got := plan.Updates[0].Fields["is_trial"]; got != 1 {
		t.Fatalf("sync is_trial = %#v, want 1", got)
	}
}
