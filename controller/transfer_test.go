package controller

import "testing"

func TestCurrentProductRouteCodeUsesFixedProductCode(t *testing.T) {
	if got := currentProductRouteCode(); got != "origin" {
		t.Fatalf("currentProductRouteCode() = %q, want %q", got, "origin")
	}
}
