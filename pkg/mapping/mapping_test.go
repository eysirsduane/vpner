package mapping

import (
	"encoding/json"
	"just-vpn/docs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type testInviteResponse struct {
	ValidHours int `json:"valid_hours"`
}

type testUserResponse struct {
	VipTime string             `json:"vip_time"`
	Invite  testInviteResponse `json:"invite"`
}

func TestMapResponseUsesJSONTagsAndStructuredMapping(t *testing.T) {
	oldConfig := RouteConfig
	defer func() { RouteConfig = oldConfig }()

	RouteConfig = &Config{
		Routes: map[string]RouteMapping{
			"/api/v1/test": {
				Response: map[string]interface{}{
					"code": "status",
					"msg":  "message",
					"result": map[string]interface{}{
						"vip_time": "vipExpire",
						"invite": map[string]interface{}{
							"valid_hours": "validHours",
						},
					},
				},
			},
		},
	}

	response := map[string]interface{}{
		"code": 200,
		"msg":  "success",
		"result": testUserResponse{
			VipTime: "2026-07-30 23:59:59",
			Invite: testInviteResponse{
				ValidHours: 72,
			},
		},
	}

	mapped := MapResponse("/api/v1/test", response)
	if _, ok := mapped["code"]; ok {
		t.Fatal("expected code to be renamed")
	}
	if mapped["status"] != int64(200) {
		t.Fatalf("expected status 200, got %#v", mapped["status"])
	}
	if mapped["message"] != "success" {
		t.Fatalf("expected message success, got %#v", mapped["message"])
	}

	result, ok := mapped["result"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected result map, got %#v", mapped["result"])
	}
	if _, ok := result["vip_time"]; ok {
		t.Fatal("expected result.vip_time to be renamed")
	}
	if result["vipExpire"] != "2026-07-30 23:59:59" {
		t.Fatalf("expected vipExpire to be mapped, got %#v", result["vipExpire"])
	}

	invite, ok := result["invite"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected result.invite map, got %#v", result["invite"])
	}
	if _, ok := invite["valid_hours"]; ok {
		t.Fatal("expected result.invite.valid_hours to be renamed")
	}
	if invite["validHours"] != int64(72) {
		t.Fatalf("expected validHours 72, got %#v", invite["validHours"])
	}
}

func TestMapResponseWithoutMappingStillNormalizesStruct(t *testing.T) {
	oldConfig := RouteConfig
	defer func() { RouteConfig = oldConfig }()

	RouteConfig = &Config{
		Routes: map[string]RouteMapping{},
	}

	mapped := MapResponse("/api/v1/test", map[string]interface{}{
		"result": testUserResponse{
			VipTime: "2026-07-30 23:59:59",
			Invite: testInviteResponse{
				ValidHours: 72,
			},
		},
	})

	result, ok := mapped["result"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected result map, got %#v", mapped["result"])
	}
	if result["vip_time"] != "2026-07-30 23:59:59" {
		t.Fatalf("expected vip_time from json tag, got %#v", result["vip_time"])
	}
}

func TestMapResponseMapsArrayItemFieldsWithStructuredMapping(t *testing.T) {
	oldConfig := RouteConfig
	defer func() { RouteConfig = oldConfig }()

	RouteConfig = &Config{
		Routes: map[string]RouteMapping{
			"/api/v1/test": {
				Response: map[string]interface{}{
					"result": map[string]interface{}{
						"devices": map[string]interface{}{
							"device_no": "deviceNo",
						},
					},
				},
			},
		},
	}

	mapped := MapResponse("/api/v1/test", map[string]interface{}{
		"result": map[string]interface{}{
			"devices": []interface{}{
				map[string]interface{}{"device_no": "device-a"},
				map[string]interface{}{"device_no": "device-b"},
			},
		},
	})

	result := mapped["result"].(map[string]interface{})
	devices := result["devices"].([]interface{})
	first := devices[0].(map[string]interface{})
	second := devices[1].(map[string]interface{})

	if _, ok := first["device_no"]; ok {
		t.Fatal("expected first device_no to be renamed")
	}
	if first["deviceNo"] != "device-a" {
		t.Fatalf("expected first deviceNo device-a, got %#v", first["deviceNo"])
	}
	if second["deviceNo"] != "device-b" {
		t.Fatalf("expected second deviceNo device-b, got %#v", second["deviceNo"])
	}
}

func TestMapResponseCanRenameNestedParentField(t *testing.T) {
	oldConfig := RouteConfig
	defer func() { RouteConfig = oldConfig }()

	RouteConfig = &Config{
		Routes: map[string]RouteMapping{
			"/api/v1/test": {
				Response: map[string]interface{}{
					"result": map[string]interface{}{
						"$field":   "payload",
						"vip_time": "vipExpire",
						"invite": map[string]interface{}{
							"$field":      "invitePack",
							"valid_hours": "validHours",
						},
					},
				},
			},
		},
	}

	mapped := MapResponse("/api/v1/test", map[string]interface{}{
		"result": testUserResponse{
			VipTime: "2026-07-30 23:59:59",
			Invite: testInviteResponse{
				ValidHours: 72,
			},
		},
	})

	if _, ok := mapped["result"]; ok {
		t.Fatal("expected result to be renamed")
	}
	payload, ok := mapped["payload"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected payload map, got %#v", mapped["payload"])
	}
	if payload["vipExpire"] != "2026-07-30 23:59:59" {
		t.Fatalf("expected vipExpire to be mapped, got %#v", payload["vipExpire"])
	}
	if _, ok := payload["invite"]; ok {
		t.Fatal("expected invite to be renamed")
	}
	invite, ok := payload["invitePack"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected invitePack map, got %#v", payload["invitePack"])
	}
	if invite["validHours"] != int64(72) {
		t.Fatalf("expected validHours 72, got %#v", invite["validHours"])
	}
}

func TestCandidatePaths(t *testing.T) {
	tests := []struct {
		name string
		path string
		want []string
	}{
		{
			name: "default",
			path: "",
			want: []string{"conf/mapping.json", "../conf/mapping.json", "../../conf/mapping.json"},
		},
		{
			name: "relative",
			path: "conf/mapping-product.json",
			want: []string{"conf/mapping-product.json", "../conf/mapping-product.json", "../../conf/mapping-product.json"},
		},
		{
			name: "absolute",
			path: "/tmp/mapping.json",
			want: []string{"/tmp/mapping.json"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := candidatePaths(tt.path)
			if len(got) != len(tt.want) {
				t.Fatalf("expected %d paths, got %d: %#v", len(tt.want), len(got), got)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Fatalf("expected path %d to be %q, got %q", i, tt.want[i], got[i])
				}
			}
		})
	}
}

func TestLoadAppliesFieldAffixToRequestAndResponseFields(t *testing.T) {
	oldConfig := RouteConfig
	defer func() { RouteConfig = oldConfig }()

	path := filepath.Join(t.TempDir(), "mapping.json")
	data := []byte(`{
		"product_code": "test",
		"swagger_path": "/app/docs",
		"field_affix": {
			"prefix": "p_",
			"suffix": "_s"
		},
		"headers": {
			"version": "X-App-Version"
		},
		"routes": {
			"/api/v1/test": {
				"path": "/app/test",
				"request": {
					"device_no": "deviceTag"
				},
				"response": {
					"code": "statusCode",
					"msg": "message",
					"result": {
						"$field": "payload",
						"device_no": "deviceTag",
						"invite": {
							"$field": "invitePack",
							"valid_hours": "validHours"
						}
					}
				}
			}
		}
	}`)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write mapping fixture: %v", err)
	}
	if err := Load(path); err != nil {
		t.Fatalf("load mapping: %v", err)
	}

	if got := RequestField("/api/v1/test", "device_no"); got != "p_deviceTag_s" {
		t.Fatalf("expected request field with affix, got %q", got)
	}
	if got := HeaderField("version"); got != "X-App-Version" {
		t.Fatalf("expected header not to use affix, got %q", got)
	}
	if got := ProductCode(); got != "test" {
		t.Fatalf("expected product code test, got %q", got)
	}
	if got := SwaggerPath(); got != "/app/docs" {
		t.Fatalf("expected swagger path /app/docs, got %q", got)
	}

	mapped := MapResponse("/api/v1/test", map[string]interface{}{
		"code": 200,
		"msg":  "success",
		"result": map[string]interface{}{
			"device_no": "device-a",
			"invite": map[string]interface{}{
				"valid_hours": 24,
			},
		},
	})
	if _, ok := mapped["statusCode"]; ok {
		t.Fatal("expected statusCode without affix to be absent")
	}
	if mapped["p_statusCode_s"] != int64(200) {
		t.Fatalf("expected affixed status code, got %#v", mapped["p_statusCode_s"])
	}
	payload, ok := mapped["p_payload_s"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected affixed payload, got %#v", mapped["p_payload_s"])
	}
	if payload["p_deviceTag_s"] != "device-a" {
		t.Fatalf("expected affixed payload device, got %#v", payload["p_deviceTag_s"])
	}
	invite, ok := payload["p_invitePack_s"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected affixed invite pack, got %#v", payload["p_invitePack_s"])
	}
	if invite["p_validHours_s"] != int64(24) {
		t.Fatalf("expected affixed valid hours, got %#v", invite["p_validHours_s"])
	}
}

func TestLoadRejectsMissingProductIdentity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mapping.json")
	data := []byte(`{"product_code":"test","routes":{}}`)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write mapping fixture: %v", err)
	}
	if err := Load(path); err == nil {
		t.Fatal("expected missing swagger_path to be rejected")
	}
}

func TestLoadRejectsResponseObjectWithoutTargetField(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mapping.json")
	data := []byte(`{
		"product_code": "test",
		"swagger_path": "/app/docs",
		"routes": {
			"/api/v1/test": {
				"response": {
					"result": {
						"id": "recordId"
					}
				}
			}
		}
	}`)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write mapping fixture: %v", err)
	}

	err := Load(path)
	if err == nil {
		t.Fatal("expected response object without $field to be rejected")
	}
	if !strings.Contains(err.Error(), "response.result object mapping requires non-empty $field") {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestMapSwaggerMapsPathRequestAndResponseFields(t *testing.T) {
	oldConfig := RouteConfig
	defer func() { RouteConfig = oldConfig }()

	RouteConfig = &Config{
		Routes: map[string]RouteMapping{
			"/api/v1/auto_login": {
				Path: "/app/session/guest",
				Request: map[string]string{
					"device_no": "deviceTag",
					"version":   "appVersion",
				},
				Response: map[string]interface{}{
					"code": "statusCode",
					"msg":  "message",
					"result": map[string]interface{}{
						nestedTargetFieldKey: "payload",
						"device_no":          "deviceTag",
						"vip_time":           "vipExpireAt",
					},
				},
			},
		},
	}

	raw := `{
		"swagger": "2.0",
		"basePath": "/api/v1",
		"paths": {
			"/auto_login": {
				"post": {
					"parameters": [{
						"in": "body",
						"schema": {"$ref": "#/definitions/AutoLoginRequest"}
					}],
					"responses": {
						"200": {
							"schema": {
								"allOf": [
									{"$ref": "#/definitions/Response"},
									{"type": "object", "properties": {
										"result": {"$ref": "#/definitions/LoginResponse"}
									}}
								]
							}
						}
					}
				}
			}
		},
		"definitions": {
			"AutoLoginRequest": {
				"type": "object",
				"required": ["device_no"],
				"properties": {
					"device_no": {"type": "string"},
					"version": {"type": "string"}
				}
			},
			"LoginResponse": {
				"type": "object",
				"properties": {
					"device_no": {"type": "string"},
					"vip_time": {"type": "string"}
				}
			},
			"Response": {
				"type": "object",
				"properties": {
					"code": {"type": "integer"},
					"msg": {"type": "string"},
					"result": {}
				}
			}
		}
	}`

	var doc map[string]interface{}
	if err := json.Unmarshal(MapSwagger(raw), &doc); err != nil {
		t.Fatalf("unmarshal mapped swagger: %v", err)
	}
	if doc["basePath"] != "" {
		t.Fatalf("expected empty basePath, got %#v", doc["basePath"])
	}

	paths := doc["paths"].(map[string]interface{})
	if _, ok := paths["/auto_login"]; ok {
		t.Fatal("expected original swagger path to be removed")
	}
	path := paths["/app/session/guest"].(map[string]interface{})
	post := path["post"].(map[string]interface{})

	parameters := post["parameters"].([]interface{})
	body := parameters[0].(map[string]interface{})
	requestSchema := body["schema"].(map[string]interface{})
	requestProperties := requestSchema["properties"].(map[string]interface{})
	if _, ok := requestProperties["device_no"]; ok {
		t.Fatal("expected request device_no to be renamed")
	}
	if _, ok := requestProperties["deviceTag"]; !ok {
		t.Fatal("expected request deviceTag field")
	}
	required := requestSchema["required"].([]interface{})
	if required[0] != "deviceTag" {
		t.Fatalf("expected required deviceTag, got %#v", required)
	}

	responses := post["responses"].(map[string]interface{})
	response := responses["200"].(map[string]interface{})
	responseSchema := response["schema"].(map[string]interface{})
	responseProperties := responseSchema["properties"].(map[string]interface{})
	if _, ok := responseProperties["code"]; ok {
		t.Fatal("expected response code to be renamed")
	}
	if _, ok := responseProperties["statusCode"]; !ok {
		t.Fatal("expected response statusCode field")
	}
	payload := responseProperties["payload"].(map[string]interface{})
	payloadProperties := payload["properties"].(map[string]interface{})
	if _, ok := payloadProperties["device_no"]; ok {
		t.Fatal("expected payload device_no to be renamed")
	}
	if _, ok := payloadProperties["deviceTag"]; !ok {
		t.Fatal("expected payload deviceTag field")
	}
}

func TestMapSwaggerBuildsPayloadSchemaFromMappingWhenResponseResultIsEmpty(t *testing.T) {
	oldConfig := RouteConfig
	defer func() { RouteConfig = oldConfig }()

	RouteConfig = &Config{
		Routes: map[string]RouteMapping{
			"/api/v1/auto_login": {
				Path: "/app/session/guest",
				Response: map[string]interface{}{
					"code": "statusCode",
					"msg":  "message",
					"result": map[string]interface{}{
						nestedTargetFieldKey: "payload",
						"id":                 "recordId",
						"device_no":          "deviceTag",
					},
				},
			},
		},
	}

	raw := `{
		"swagger": "2.0",
		"basePath": "/api/v1",
		"paths": {
			"/auto_login": {
				"post": {
					"responses": {
						"200": {"schema": {"$ref": "#/definitions/Response"}}
					}
				}
			}
		},
		"definitions": {
			"Response": {
				"type": "object",
				"properties": {
					"code": {"type": "integer"},
					"msg": {"type": "string"},
					"result": {}
				}
			}
		}
	}`

	var doc map[string]interface{}
	if err := json.Unmarshal(MapSwagger(raw), &doc); err != nil {
		t.Fatalf("unmarshal mapped swagger: %v", err)
	}
	paths := doc["paths"].(map[string]interface{})
	path := paths["/app/session/guest"].(map[string]interface{})
	post := path["post"].(map[string]interface{})
	responses := post["responses"].(map[string]interface{})
	response := responses["200"].(map[string]interface{})
	schema := response["schema"].(map[string]interface{})
	properties := schema["properties"].(map[string]interface{})
	payload := properties["payload"].(map[string]interface{})
	payloadProperties := payload["properties"].(map[string]interface{})
	if _, ok := payloadProperties["recordId"]; !ok {
		t.Fatal("expected synthesized payload recordId field")
	}
	if _, ok := payloadProperties["deviceTag"]; !ok {
		t.Fatal("expected synthesized payload deviceTag field")
	}
}

func TestMapSwaggerCurrentDocsUsesConfiguredMapping(t *testing.T) {
	var doc map[string]interface{}
	if err := json.Unmarshal(MapSwagger(docs.SwaggerInfo.ReadDoc()), &doc); err != nil {
		t.Fatalf("unmarshal mapped current swagger: %v", err)
	}

	paths := doc["paths"].(map[string]interface{})
	if _, ok := paths["/auto_login"]; ok {
		t.Fatal("expected original auto_login path to be removed")
	}
	autoLoginRoute := RouteConfig.Routes["/api/v1/auto_login"]
	path, ok := paths[swaggerWildcardPath(autoLoginRoute.Path)].(map[string]interface{})
	if !ok {
		t.Fatal("expected mapped auto login path")
	}
	assetRoute := RouteConfig.Routes["/api/v1/upload/*filepath"]
	if _, ok := paths[swaggerWildcardPath(assetRoute.Path)].(map[string]interface{}); !ok {
		t.Fatal("expected mapped upload asset path")
	}
	post := path["post"].(map[string]interface{})

	parameters := post["parameters"].([]interface{})
	hasHeader := false
	hasMappedBody := false
	expectedDeviceField := RouteConfig.Routes["/api/v1/auto_login"].Request["device_no"]
	for _, parameter := range parameters {
		param := parameter.(map[string]interface{})
		if param["in"] == "header" && param["name"] == HeaderField("version") {
			hasHeader = true
		}
		if param["in"] != "body" {
			continue
		}
		schema := param["schema"].(map[string]interface{})
		properties := schema["properties"].(map[string]interface{})
		if _, ok := properties[expectedDeviceField]; ok {
			hasMappedBody = true
		}
	}
	if !hasHeader {
		t.Fatalf("expected mapped client header %s", HeaderField("version"))
	}
	if !hasMappedBody {
		t.Fatalf("expected mapped body field %s", expectedDeviceField)
	}

	responses := post["responses"].(map[string]interface{})
	response := responses["200"].(map[string]interface{})
	schema := response["schema"].(map[string]interface{})
	properties := schema["properties"].(map[string]interface{})
	responseFields := RouteConfig.Routes["/api/v1/auto_login"].Response
	expectedCodeField, _ := responseFields["code"].(string)
	resultFields, _ := responseFields["result"].(map[string]interface{})
	expectedPayloadField, _ := resultFields["$field"].(string)
	if expectedPayloadField == "" {
		expectedPayloadField = "result"
	}
	expectedIDField, _ := resultFields["id"].(string)
	if _, ok := properties[expectedCodeField]; !ok {
		t.Fatalf("expected mapped response field %s", expectedCodeField)
	}
	payload := properties[expectedPayloadField].(map[string]interface{})
	payloadProperties := payload["properties"].(map[string]interface{})
	if _, ok := payloadProperties[expectedIDField]; !ok {
		t.Fatalf("expected mapped payload field %s", expectedIDField)
	}

	versionRoute := RouteConfig.Routes["/api/v1/version"]
	versionPath, ok := paths[swaggerWildcardPath(versionRoute.Path)].(map[string]interface{})
	if !ok {
		t.Fatal("expected mapped version path")
	}
	versionPost := versionPath["post"].(map[string]interface{})
	versionParameters := versionPost["parameters"].([]interface{})
	mappedHeaders := map[string]bool{}
	for _, field := range []string{"platform", "version", "build", "client_time", "time_zone"} {
		mappedHeaders[HeaderField(field)] = false
	}
	for _, parameter := range versionParameters {
		param := parameter.(map[string]interface{})
		if param["in"] != "header" {
			continue
		}
		name, _ := param["name"].(string)
		if _, ok := mappedHeaders[name]; ok {
			mappedHeaders[name] = true
		}
	}
	for name, ok := range mappedHeaders {
		if !ok {
			t.Fatalf("expected mapped version header %s", name)
		}
	}
}
