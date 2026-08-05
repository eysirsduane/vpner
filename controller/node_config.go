package controller

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	nodeConfigTypeFast   = "fast"
	nodeConfigTypeGlobal = "global"
)

type NodeConfigRequest struct {
	Code string `json:"code" binding:"required" example:"HK"`   // 线路代码
	Type string `json:"type" binding:"required" example:"fast"` // 连接模式(fast=极速,global=全局)
}

type NodeConfigResponse struct {
	Config string `json:"config" example:"x/k5A0v9kiJjL0r3m6X9dA=="` // 加密后的完整JSON节点配置
}

type nodeConfigOutbound struct {
	Value  map[string]interface{}
	Host   string
	Domain string
}

// NodeConfigHandler 获取客户端 JSON 节点配置
// @Summary 获取 JSON 节点配置
// @Description 根据线路代码和连接模式获取加密后的完整 JSON 节点配置
// @Tags 线路
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body NodeConfigRequest true "获取 JSON 节点配置请求"
// @Success 200 {object} Response{result=NodeConfigResponse}
// @Router /node_config [post]
func NodeConfigHandler(c *gin.Context) {
	var req NodeConfigRequest
	if err := BindMappedJSON(c, &req); err != nil {
		if strings.Contains(err.Error(), "NodeConfigRequest.Code") || strings.Contains(err.Error(), "'Code'") {
			JsonReturn(c, CodeError, "code is required", nil)
			return
		}
		if strings.Contains(err.Error(), "NodeConfigRequest.Type") || strings.Contains(err.Error(), "'Type'") {
			JsonReturn(c, CodeError, "type is required", nil)
			return
		}
		JsonReturn(c, CodeError, "invalid json body", nil)
		return
	}

	code := strings.ToUpper(strings.TrimSpace(req.Code))
	if code == "" {
		JsonReturn(c, CodeError, "code is required", nil)
		return
	}
	configType, err := normalizeNodeConfigType(req.Type)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}

	node, ok := getNodeForCode(c, code)
	if !ok {
		return
	}

	config, err := buildNodeClientConfig(node.LinkUrl, configType)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	encrypted, err := encryptedNodeLinkURL(config)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	JsonReturn(c, CodeSuccess, "success", NodeConfigResponse{Config: encrypted})
}

func normalizeNodeConfigType(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case nodeConfigTypeFast, "极速":
		return nodeConfigTypeFast, nil
	case nodeConfigTypeGlobal, "全局":
		return nodeConfigTypeGlobal, nil
	default:
		return "", fmt.Errorf("type must be fast or global")
	}
}

func buildNodeClientConfig(link, configType string) (string, error) {
	outbound, err := parseNodeConfigOutbound(link)
	if err != nil {
		return "", err
	}
	config := map[string]interface{}{
		"inbounds": []interface{}{map[string]interface{}{
			"sniff":          true,
			"strict_route":   true,
			"mtu":            1420,
			"auto_route":     true,
			"interface_name": "utun12",
			"tag":            "tun-in",
			"stack":          "system",
			"type":           "tun",
			"inet4_address":  "172.19.0.1/30",
		}},
		"dns":       buildNodeConfigDNS(outbound.Domain, configType),
		"log":       map[string]interface{}{"level": "debug", "timestamp": true, "output": "singboxlog.log"},
		"outbounds": []interface{}{outbound.Value, fixedNodeOutbound("direct"), fixedNodeOutbound("block"), fixedNodeOutbound("dns")},
		"route":     buildNodeConfigRoute(outbound.Host, outbound.Domain, configType),
	}

	data, err := json.Marshal(config)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func buildNodeConfigDNS(domain, configType string) map[string]interface{} {
	rules := []interface{}{map[string]interface{}{"server": "dns_local", "outbound": "any"}}
	if domain != "" {
		rules = append(rules, map[string]interface{}{"server": "dns_local", "domain": []string{domain}})
	}
	if configType == nodeConfigTypeFast {
		rules = append(rules,
			map[string]interface{}{"server": "dns_local", "rewrite_ttl": 900, "rule_set": "geosite-cn"},
			map[string]interface{}{"server": "dns_proxy", "rewrite_ttl": 900, "rule_set": "geosite-geolocation-!cn"},
		)
	}

	dns := map[string]interface{}{
		"servers": []interface{}{
			map[string]interface{}{"strategy": "prefer_ipv4", "detour": "proxy", "address_strategy": "prefer_ipv4", "tag": "dns_proxy", "address": "1.1.1.1"},
			map[string]interface{}{"strategy": "prefer_ipv4", "detour": "direct", "address_strategy": "prefer_ipv4", "tag": "dns_local", "address": "223.5.5.5"},
			map[string]interface{}{"tag": "dns_block", "address": "rcode://refused"},
		},
		"rules":    rules,
		"strategy": "prefer_ipv4",
		"final":    "dns_proxy",
	}
	if configType == nodeConfigTypeGlobal {
		dns["disable_expire"] = false
	}
	return dns
}

func buildNodeConfigRoute(host, domain, configType string) map[string]interface{} {
	rules := []interface{}{map[string]interface{}{"protocol": "dns", "outbound": "dns_out"}}
	if ip := net.ParseIP(host); ip != nil {
		bits := 32
		if ip.To4() == nil {
			bits = 128
		}
		rules = append(rules, map[string]interface{}{"ip_cidr": []string{ip.String() + "/" + strconv.Itoa(bits)}, "outbound": "direct"})
	} else if domain != "" {
		rules = append(rules, map[string]interface{}{"outbound": "direct", "domain": []string{domain}})
	}
	rules = append(rules, map[string]interface{}{"protocol": "quic", "outbound": "block"})

	route := map[string]interface{}{"rules": rules, "final": "proxy", "auto_detect_interface": true}
	if configType != nodeConfigTypeFast {
		rules = append(rules, map[string]interface{}{"ip_is_private": true, "outbound": "direct"})
		route["rules"] = rules
		return route
	}

	rules = append(rules,
		map[string]interface{}{
			"rules": []interface{}{
				map[string]interface{}{"invert": true, "rule_set": "geoip-cn"},
				map[string]interface{}{"rule_set": "geosite-geolocation-!cn"},
			},
			"outbound": "proxy",
			"type":     "logical",
			"mode":     "and",
		},
		map[string]interface{}{"outbound": "direct", "rule_set": "geoip-cn"},
		map[string]interface{}{"outbound": "direct", "ip_is_private": true},
	)
	route["rules"] = rules
	route["rule_set"] = []interface{}{
		map[string]interface{}{"path": "geosite-geolocation-!cn.srs", "type": "local", "tag": "geosite-geolocation-!cn", "format": "binary"},
		map[string]interface{}{"path": "geosite-cn.srs", "type": "local", "tag": "geosite-cn", "format": "binary"},
		map[string]interface{}{"path": "geoip-cn.srs", "type": "local", "tag": "geoip-cn", "format": "binary"},
	}
	return route
}

func fixedNodeOutbound(kind string) map[string]interface{} {
	return map[string]interface{}{"type": kind, "tag": map[string]string{"direct": "direct", "block": "block", "dns": "dns_out"}[kind]}
}

func parseNodeConfigOutbound(link string) (nodeConfigOutbound, error) {
	trimmed := strings.TrimSpace(link)
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return nodeConfigOutbound{}, fmt.Errorf("invalid node")
	}
	switch strings.ToLower(parsed.Scheme) {
	case "anytls":
		return parseAnyTLSConfigOutbound(parsed)
	case "vless":
		return parseVLESSConfigOutbound(parsed)
	case "vmess":
		return parseVMessConfigOutbound(trimmed)
	case "chimney":
		return parseChimneyConfigOutbound(parsed)
	default:
		return nodeConfigOutbound{}, fmt.Errorf("unsupported node protocol")
	}
}

func parseAnyTLSConfigOutbound(parsed *url.URL) (nodeConfigOutbound, error) {
	if parsed.User == nil || strings.TrimSpace(parsed.User.Username()) == "" {
		return nodeConfigOutbound{}, fmt.Errorf("anytls password is required")
	}
	host, port, err := parseNodeHostPort(parsed)
	if err != nil {
		return nodeConfigOutbound{}, err
	}
	query := parsed.Query()
	proxy := map[string]interface{}{"server": host, "server_port": port, "password": parsed.User.Username(), "type": "anytls", "tag": "proxy"}
	if strings.EqualFold(query.Get("security"), "tls") {
		proxy["tls"] = buildNodeTLS(query, host)
	}
	return nodeConfigOutbound{Value: proxy, Host: host, Domain: nodeConfigDomain(host)}, nil
}

func parseVLESSConfigOutbound(parsed *url.URL) (nodeConfigOutbound, error) {
	if parsed.User == nil || strings.TrimSpace(parsed.User.Username()) == "" {
		return nodeConfigOutbound{}, fmt.Errorf("vless uuid is required")
	}
	host, port, err := parseNodeHostPort(parsed)
	if err != nil {
		return nodeConfigOutbound{}, err
	}
	query := parsed.Query()
	proxy := map[string]interface{}{
		"server":          host,
		"server_port":     port,
		"uuid":            parsed.User.Username(),
		"packet_encoding": "xudp",
		"type":            "vless",
		"tag":             "proxy",
		"transport":       map[string]interface{}{},
	}
	if flow := strings.TrimSpace(query.Get("flow")); flow != "" {
		proxy["flow"] = flow
	}
	if strings.EqualFold(query.Get("security"), "tls") {
		proxy["tls"] = buildNodeTLS(query, host)
	}
	return nodeConfigOutbound{Value: proxy, Host: host, Domain: nodeConfigDomain(host)}, nil
}

func parseVMessConfigOutbound(link string) (nodeConfigOutbound, error) {
	payload := strings.TrimSpace(strings.TrimPrefix(link, "vmess://"))
	data, err := decodeVMessPayload(payload)
	if err != nil {
		return nodeConfigOutbound{}, fmt.Errorf("invalid vmess config")
	}
	var source map[string]interface{}
	if err := json.Unmarshal(data, &source); err != nil {
		return nodeConfigOutbound{}, fmt.Errorf("invalid vmess config")
	}
	host := stringValue(source["add"])
	uuid := stringValue(source["id"])
	port, err := intValue(source["port"])
	if host == "" || uuid == "" || err != nil || port <= 0 {
		return nodeConfigOutbound{}, fmt.Errorf("vmess server, port and uuid are required")
	}
	alterID, err := intValue(source["aid"])
	if err != nil || alterID < 0 {
		return nodeConfigOutbound{}, fmt.Errorf("vmess alter id is invalid")
	}
	proxy := map[string]interface{}{
		"server":      host,
		"server_port": port,
		"uuid":        uuid,
		"security":    firstNodeConfigValue(stringValue(source["scy"]), "auto"),
		"alter_id":    alterID,
		"type":        "vmess",
		"tag":         "proxy",
	}
	if strings.EqualFold(stringValue(source["tls"]), "tls") {
		query := url.Values{}
		query.Set("sni", firstNodeConfigValue(stringValue(source["sni"]), host))
		query.Set("insecure", stringValue(source["insecure"]))
		query.Set("alpn", stringValue(source["alpn"]))
		query.Set("fp", stringValue(source["fp"]))
		query.Set("pcs", stringValue(source["pcs"]))
		proxy["tls"] = buildNodeTLS(query, host)
	}
	if strings.EqualFold(stringValue(source["net"]), "ws") {
		proxy["transport"] = map[string]interface{}{
			"type":    "ws",
			"path":    firstNodeConfigValue(stringValue(source["path"]), "/"),
			"headers": map[string]string{"Host": firstNodeConfigValue(stringValue(source["host"]), host)},
		}
	}
	return nodeConfigOutbound{Value: proxy, Host: host, Domain: nodeConfigDomain(host)}, nil
}

func parseChimneyConfigOutbound(parsed *url.URL) (nodeConfigOutbound, error) {
	if parsed.User == nil || strings.TrimSpace(parsed.User.Username()) == "" {
		return nodeConfigOutbound{}, fmt.Errorf("chimney user id is required")
	}
	host := strings.TrimSpace(parsed.Hostname())
	if host == "" {
		return nodeConfigOutbound{}, fmt.Errorf("chimney server is required")
	}
	port, err := strconv.Atoi(strings.TrimSpace(parsed.Port()))
	if err != nil || port <= 0 || port > 65535 {
		return nodeConfigOutbound{}, fmt.Errorf("chimney port is invalid")
	}

	query := parsed.Query()
	snis := splitChimneyNodeConfigSNIs(query.Get("sni"))
	if len(snis) == 0 {
		return nodeConfigOutbound{}, fmt.Errorf("chimney sni is required")
	}
	fingerprint := strings.TrimSpace(query.Get("fp"))
	if fingerprint == "" {
		return nodeConfigOutbound{}, fmt.Errorf("chimney fingerprint is required")
	}

	settings := map[string]interface{}{
		"relayAddr":   host,
		"server_port": port,
		"snis":        snis,
		"userId":      strings.TrimSpace(parsed.User.Username()),
		"fingerprint": fingerprint,
	}
	for _, field := range []string{"tagLen", "poolSize", "tcpBufferSize", "connectTimeoutMs", "handshakeTimeoutMs"} {
		value, err := positiveNodeConfigQueryInt(query, field)
		if err != nil {
			return nodeConfigOutbound{}, fmt.Errorf("chimney %s is invalid", field)
		}
		settings[field] = value
	}

	proxy := map[string]interface{}{
		"tag":      "proxy",
		"protocol": "chimney",
		"settings": settings,
	}
	return nodeConfigOutbound{Value: proxy, Host: host, Domain: nodeConfigDomain(host)}, nil
}

func splitChimneyNodeConfigSNIs(value string) []string {
	value = strings.NewReplacer("+", " ", ",", " ").Replace(value)
	return strings.Fields(value)
}

func positiveNodeConfigQueryInt(query url.Values, field string) (int, error) {
	value, err := strconv.Atoi(strings.TrimSpace(query.Get(field)))
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("invalid positive integer")
	}
	return value, nil
}

func parseNodeHostPort(parsed *url.URL) (string, int, error) {
	host := strings.TrimSpace(parsed.Hostname())
	port, err := strconv.Atoi(parsed.Port())
	if host == "" || err != nil || port <= 0 {
		return "", 0, fmt.Errorf("node server and port are required")
	}
	return host, port, nil
}

func buildNodeTLS(query url.Values, defaultServerName string) map[string]interface{} {
	tls := map[string]interface{}{
		"enabled":     true,
		"server_name": firstNodeConfigValue(strings.TrimSpace(query.Get("sni")), defaultServerName),
		"insecure":    nodeConfigTruthy(query.Get("insecure")) || nodeConfigTruthy(query.Get("allowInsecure")),
	}
	if alpn := splitNodeConfigCSV(query.Get("alpn")); len(alpn) > 0 {
		tls["alpn"] = alpn
	}
	if fingerprint := strings.TrimSpace(query.Get("fp")); fingerprint != "" {
		tls["utls"] = map[string]interface{}{"enabled": true, "fingerprint": fingerprint}
	}
	if pcs := strings.TrimSpace(query.Get("pcs")); pcs != "" {
		tls["pinned_peer_certificate_chain_sha256"] = []string{pcs}
	}
	return tls
}

func decodeVMessPayload(payload string) ([]byte, error) {
	payload = strings.TrimSpace(payload)
	for _, encoding := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		if decoded, err := encoding.DecodeString(payload); err == nil {
			return decoded, nil
		}
	}
	return nil, fmt.Errorf("invalid vmess payload")
}

func nodeConfigDomain(host string) string {
	if net.ParseIP(host) != nil {
		return ""
	}
	return host
}

func splitNodeConfigCSV(value string) []string {
	var values []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			values = append(values, item)
		}
	}
	return values
}

func nodeConfigTruthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func firstNodeConfigValue(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func stringValue(value interface{}) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case float64:
		return strconv.FormatInt(int64(typed), 10)
	case json.Number:
		return typed.String()
	default:
		return ""
	}
}

func intValue(value interface{}) (int, error) {
	switch typed := value.(type) {
	case float64:
		return int(typed), nil
	case string:
		return strconv.Atoi(strings.TrimSpace(typed))
	case json.Number:
		return strconv.Atoi(typed.String())
	default:
		return 0, fmt.Errorf("invalid integer")
	}
}
