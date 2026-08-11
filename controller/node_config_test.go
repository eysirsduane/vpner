package controller

import (
	"encoding/base64"
	"encoding/json"
	"just-vpn/pkg/mapping"
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
	inbounds := result["inbounds"].([]interface{})
	if len(inbounds) != 1 || inbounds[0].(map[string]interface{})["sniff_override_destination"] != true {
		t.Fatalf("unexpected inbounds: %#v", inbounds)
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

func TestBuildNodeClientConfigAddsSkipProxyDomains(t *testing.T) {
	config, err := buildNodeClientConfig(
		nodeConfigVLESSTestURL,
		nodeConfigTypeGlobal,
		" direct.example.com ",
		"api.example.com",
		"DIRECT.example.com",
		"",
	)
	if err != nil {
		t.Fatalf("buildNodeClientConfig returned error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(config), &result); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	want := []interface{}{"direct.example.com", "api.example.com"}
	dnsRules := result["dns"].(map[string]interface{})["rules"].([]interface{})
	if !reflect.DeepEqual(dnsRules[1].(map[string]interface{})["domain"], want) {
		t.Fatalf("unexpected DNS domains: %#v", dnsRules[1])
	}
	routeRules := result["route"].(map[string]interface{})["rules"].([]interface{})
	if !reflect.DeepEqual(routeRules[2].(map[string]interface{})["domain"], want) {
		t.Fatalf("unexpected route domains: %#v", routeRules[2])
	}
}

func TestBuildNodeConfigOutboundsMatchesFullConfig(t *testing.T) {
	_, outbounds, err := buildNodeConfigOutbounds(nodeConfigVLESSTestURL)
	if err != nil {
		t.Fatalf("buildNodeConfigOutbounds returned error: %v", err)
	}
	config, err := buildNodeClientConfig(nodeConfigVLESSTestURL, nodeConfigTypeFast)
	if err != nil {
		t.Fatalf("buildNodeClientConfig returned error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(config), &result); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	normalizedOutbounds, err := json.Marshal(outbounds)
	if err != nil {
		t.Fatalf("marshal outbounds: %v", err)
	}
	var normalized interface{}
	if err := json.Unmarshal(normalizedOutbounds, &normalized); err != nil {
		t.Fatalf("unmarshal outbounds: %v", err)
	}
	if !reflect.DeepEqual(normalized, result["outbounds"]) {
		t.Fatalf("outbounds differ\ngot:  %#v\nwant: %#v", normalized, result["outbounds"])
	}
}

func TestEncryptedNodeConfigOutboundsUsesNodeEncryption(t *testing.T) {
	encrypt := func(value string) (string, error) {
		return encryptedNodeLinkURLWithKey(value, defaultNodeLinkAESKey)
	}

	_, outbounds, err := buildNodeConfigOutbounds(nodeConfigVLESSTestURL)
	if err != nil {
		t.Fatalf("buildNodeConfigOutbounds returned error: %v", err)
	}
	encrypted, err := encryptedNodeConfigOutboundsWith(outbounds, encrypt)
	if err != nil {
		t.Fatalf("encryptedNodeConfigOutbounds returned error: %v", err)
	}
	decrypted := decryptNodeLinkForTest(t, encrypted)
	decoded, err := base64.StdEncoding.DecodeString(decrypted)
	if err != nil {
		t.Fatalf("base64 decode decrypted outbounds failed: %v", err)
	}

	var got interface{}
	if err := json.Unmarshal(decoded, &got); err != nil {
		t.Fatalf("unmarshal decrypted outbounds: %v", err)
	}
	normalizedOutbounds, err := json.Marshal(outbounds)
	if err != nil {
		t.Fatalf("marshal expected outbounds: %v", err)
	}
	var want interface{}
	if err := json.Unmarshal(normalizedOutbounds, &want); err != nil {
		t.Fatalf("unmarshal expected outbounds: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("decrypted outbounds differ\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestNodeOutboundsDefaultMapping(t *testing.T) {
	route, ok := mapping.RouteConfig.Routes["/api/v1/node_outbounds"]
	if !ok {
		t.Fatal("node_outbounds mapping is missing")
	}
	if route.Path == "" || route.Request["code"] == "" {
		t.Fatalf("unexpected node_outbounds route: %#v", route)
	}

	mapped := mapping.MapResponse("/api/v1/node_outbounds", map[string]interface{}{
		"code": 200,
		"msg":  "success",
		"result": NodeOutboundsResponse{
			LinkUrl:   "encrypted-link",
			Outbounds: "encrypted-outbounds",
		},
	})
	resultMapping, ok := route.Response["result"].(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected node_outbounds result mapping: %#v", route.Response["result"])
	}
	resultField, _ := resultMapping["$field"].(string)
	linkField, _ := resultMapping["link_url"].(string)
	outboundsField, _ := resultMapping["outbounds"].(string)
	result, ok := mapped[resultField].(map[string]interface{})
	if !ok || result[linkField] != "encrypted-link" || result[outboundsField] != "encrypted-outbounds" {
		t.Fatalf("unexpected mapped response: %#v", mapped)
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
