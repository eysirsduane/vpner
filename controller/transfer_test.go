package controller

import (
	"testing"

	"just-vpn/pkg/mapping"
)

func TestCurrentProductRouteCodeUsesMappingProductCode(t *testing.T) {
	if got, want := currentProductRouteCode(), mapping.ProductCode(); got != want {
		t.Fatalf("currentProductRouteCode() = %q, want %q", got, want)
	}
}
