package controller

import (
	"encoding/json"
	"reflect"
	"testing"
)

const nodeConfigVLESSTestURL = "vless://1c95dc1f-c0a2-4557-b0b5-6f8dff000016@38.180.188.71:443?flow=xtls-rprx-vision&security=tls&sni=www.digicert.com&fp=chrome&alpn=h2&insecure=1&pcs=C6BDD94AFC8B7D7223DBC3FE57A246F2FC7B09750113976733E4ADC437FB5E91"
const nodeConfigAnyTLSDomainTestURL = "anytls://password@example.com:443?security=tls&sni=cdn.example.com&fp=chrome&alpn=h2"
const nodeConfigChimneyTestURL = "chimney://123a48fd-9a6b-4a6d-8801-97288e665bed@ts6dh42309db4se3g5sds35g4s3dg.oylfmxz.cn:443?security=tls&sni=www.cloudflare.com+developers.cloudflare.com+dash.cloudflare.com+community.cloudflare.com+ot.www.cloudflare.com+static.cloudflareinsights.com&fp=chrome&tagLen=16&poolSize=4&tcpBufferSize=65536&connectTimeoutMs=10000&handshakeTimeoutMs=10000#123456"

func TestBuildNodeClientConfigFastWithIPNode(t *testing.T) {
	config, err := buildNodeClientConfig(nodeConfigVLESSTestURL, nodeConfigTypeFast)
	if err != nil {
		t.Fatalf("buildNodeClientConfig returned error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(config), &result); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	outbounds := result["outbounds"].([]interface{})
	if len(outbounds) != 4 {
		t.Fatalf("outbounds count = %d, want 4", len(outbounds))
	}
	proxy := outbounds[0].(map[string]interface{})
	if proxy["type"] != "vless" || proxy["server"] != "38.180.188.71" || proxy["server_port"] != float64(443) {
		t.Fatalf("unexpected proxy: %#v", proxy)
	}
	tls := proxy["tls"].(map[string]interface{})
	if tls["server_name"] != "www.digicert.com" || tls["insecure"] != true {
		t.Fatalf("unexpected tls: %#v", tls)
	}

	route := result["route"].(map[string]interface{})
	if _, ok := route["rule_set"]; !ok {
		t.Fatal("fast mode rule_set is missing")
	}
	rules := route["rules"].([]interface{})
	ipRule := rules[1].(map[string]interface{})
	if ipRule["ip_cidr"].([]interface{})[0] != "38.180.188.71/32" {
		t.Fatalf("unexpected ip direct rule: %#v", ipRule)
	}
}

func TestBuildNodeClientConfigGlobalWithDomainNode(t *testing.T) {
	config, err := buildNodeClientConfig(nodeConfigAnyTLSDomainTestURL, nodeConfigTypeGlobal)
	if err != nil {
		t.Fatalf("buildNodeClientConfig returned error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(config), &result); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	dns := result["dns"].(map[string]interface{})
	rules := dns["rules"].([]interface{})
	if len(rules) != 2 {
		t.Fatalf("dns rules count = %d, want 2", len(rules))
	}
	domainRule := rules[1].(map[string]interface{})
	if domainRule["domain"].([]interface{})[0] != "example.com" {
		t.Fatalf("unexpected dns domain rule: %#v", domainRule)
	}

	route := result["route"].(map[string]interface{})
	if _, ok := route["rule_set"]; ok {
		t.Fatal("global mode must not include rule_set")
	}
	routeRules := route["rules"].([]interface{})
	if routeRules[1].(map[string]interface{})["domain"].([]interface{})[0] != "example.com" {
		t.Fatalf("unexpected route domain rule: %#v", routeRules[1])
	}
}

func TestBuildNodeClientConfigChimney(t *testing.T) {
	config, err := buildNodeClientConfig(nodeConfigChimneyTestURL, nodeConfigTypeFast)
	if err != nil {
		t.Fatalf("buildNodeClientConfig returned error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(config), &result); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	const expectedJSON = `[
		{
			"type":"chimney",
			"tag":"proxy",
			"server":"ts6dh42309db4se3g5sds35g4s3dg.oylfmxz.cn",
			"server_port":443,
			"snis":[
				"www.cloudflare.com",
				"developers.cloudflare.com",
				"dash.cloudflare.com",
				"community.cloudflare.com",
				"ot.www.cloudflare.com",
				"static.cloudflareinsights.com"
			],
			"user_id":"123a48fd-9a6b-4a6d-8801-97288e665bed",
			"fingerprint":"chrome",
			"tag_len":16,
			"pool_size":4,
			"tcp_buffer_size":65536,
			"connect_timeout":"10s",
			"handshake_timeout":"10s"
		},
		{"tag":"direct","type":"direct"},
		{"tag":"block","type":"block"},
		{"tag":"dns_out","type":"dns"}
	]`
	var expected interface{}
	if err := json.Unmarshal([]byte(expectedJSON), &expected); err != nil {
		t.Fatalf("unmarshal expected outbounds: %v", err)
	}
	if !reflect.DeepEqual(result["outbounds"], expected) {
		t.Fatalf("unexpected Chimney outbounds\ngot:  %#v\nwant: %#v", result["outbounds"], expected)
	}
	if result["dns"] == nil || result["route"] == nil || result["inbounds"] == nil {
		t.Fatal("Chimney config must retain the shared DNS, route and inbound sections")
	}
}

func TestNormalizeNodeConfigType(t *testing.T) {
	for _, value := range []string{"fast", "极速", "global", "全局"} {
		if _, err := normalizeNodeConfigType(value); err != nil {
			t.Fatalf("normalizeNodeConfigType(%q) returned error: %v", value, err)
		}
	}
	if _, err := normalizeNodeConfigType("unknown"); err == nil {
		t.Fatal("expected invalid mode error")
	}
}
