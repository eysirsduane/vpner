package mapping

import (
	"encoding/json"
	"sort"
	"strings"

	justswagger "just-vpn/pkg/just-swagger"
)

var swaggerHeaderDefaults = []struct {
	field string
	value string
}{
	{field: "platform", value: "iphone"},
	{field: "version", value: "1.0.0"},
	{field: "build", value: "100"},
	{field: "device_no", value: "swagger-device-001"},
	{field: "client_time", value: "2026-07-11T20:30:15.123+08:00"},
	{field: "time_zone", value: "Asia/Shanghai"},
	{field: "app_store_region", value: "CHN"},
	{field: "language", value: "zh-Hans"},
}

// SwaggerGlobalHeaders returns Swagger-only public request headers from the active mapping
func SwaggerGlobalHeaders() []justswagger.Header {
	headers := make([]justswagger.Header, 0, len(swaggerHeaderDefaults))
	for _, item := range swaggerHeaderDefaults {
		name := strings.TrimSpace(HeaderField(item.field))
		if name == "" {
			continue
		}
		headers = append(headers, justswagger.Header{Key: name, Value: item.value})
	}
	return headers
}

// SwaggerTokenExtractRules derives token extraction rules from mapped response fields
func SwaggerTokenExtractRules() []justswagger.TokenExtractRule {
	paths := make([]string, 0)
	for originalPath, route := range RouteConfig.Routes {
		result, ok := route.Response["result"].(map[string]interface{})
		if !ok || strings.TrimSpace(route.Path) == "" {
			continue
		}
		resultField, _ := result[nestedTargetFieldKey].(string)
		if strings.TrimSpace(resultField) == "" {
			resultField = "result"
		}
		tokenField, _ := result["token"].(string)
		if strings.TrimSpace(resultField) == "" || strings.TrimSpace(tokenField) == "" {
			continue
		}
		paths = append(paths, originalPath)
	}
	sort.Strings(paths)

	rules := make([]justswagger.TokenExtractRule, 0, len(paths))
	for _, originalPath := range paths {
		route := RouteConfig.Routes[originalPath]
		result := route.Response["result"].(map[string]interface{})
		resultField, _ := result[nestedTargetFieldKey].(string)
		resultField = strings.TrimSpace(resultField)
		if resultField == "" {
			resultField = "result"
		}
		tokenField := strings.TrimSpace(result["token"].(string))
		rules = append(rules, justswagger.TokenExtractRule{
			Enabled:     true,
			PathPattern: route.Path,
			JSONPath:    resultField + "." + tokenField,
			HeaderKey:   "Authorization",
			Prefix:      "Bearer ",
		})
	}
	return rules
}

func MapSwagger(raw string) []byte {
	var doc map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return []byte(raw)
	}

	doc["basePath"] = ""
	paths, _ := doc["paths"].(map[string]interface{})
	definitions, _ := doc["definitions"].(map[string]interface{})
	mappedPaths := make(map[string]interface{}, len(paths))

	for originalPath, route := range RouteConfig.Routes {
		swaggerPath := strings.TrimPrefix(originalPath, "/api/v1")
		if swaggerPath == "" {
			swaggerPath = "/"
		}
		pathDoc, ok := paths[swaggerPath]
		if !ok {
			pathDoc, ok = paths[swaggerWildcardPath(swaggerPath)]
		}
		if !ok {
			continue
		}

		pathClone := cloneSwaggerValue(pathDoc).(map[string]interface{})
		mapSwaggerPath(pathClone, route, definitions)

		targetPath := route.Path
		if targetPath == "" {
			targetPath = originalPath
		}
		mappedPaths[swaggerWildcardPath(targetPath)] = pathClone
	}

	doc["paths"] = mappedPaths
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return []byte(raw)
	}
	return data
}

func swaggerWildcardPath(path string) string {
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		if strings.HasPrefix(segment, "*") && len(segment) > 1 {
			segments[i] = "{" + strings.TrimPrefix(segment, "*") + "}"
		}
	}
	return strings.Join(segments, "/")
}

func mapSwaggerPath(pathDoc map[string]interface{}, route RouteMapping, definitions map[string]interface{}) {
	for _, methodDoc := range pathDoc {
		method, ok := methodDoc.(map[string]interface{})
		if !ok {
			continue
		}
		mapSwaggerHeaders(method)
		mapSwaggerRequest(method, route.Request, definitions)
		mapSwaggerResponse(method, route.Response, definitions)
	}
}

func mapSwaggerHeaders(method map[string]interface{}) {
	if len(RouteConfig.Headers) == 0 {
		return
	}

	parameters, _ := method["parameters"].([]interface{})
	exists := make(map[string]bool, len(parameters))
	for _, parameter := range parameters {
		param, ok := parameter.(map[string]interface{})
		if !ok || param["in"] != "header" {
			continue
		}
		if name, ok := param["name"].(string); ok {
			if mappedName := mappedSwaggerHeaderName(name); mappedName != "" {
				param["name"] = mappedName
				name = mappedName
			}
			exists[strings.ToLower(name)] = true
		}
	}

	for _, header := range RouteConfig.Headers {
		header = strings.TrimSpace(header)
		if header == "" || exists[strings.ToLower(header)] {
			continue
		}
		parameters = append(parameters, map[string]interface{}{
			"name":        header,
			"in":          "header",
			"required":    false,
			"type":        "string",
			"description": "客户端公共请求头",
		})
		exists[strings.ToLower(header)] = true
	}
	if len(parameters) > 0 {
		method["parameters"] = parameters
	}
}

func mappedSwaggerHeaderName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	for field, mappedName := range RouteConfig.Headers {
		if strings.EqualFold(name, originalSwaggerHeaderName(field)) {
			return strings.TrimSpace(mappedName)
		}
	}
	return ""
}

func originalSwaggerHeaderName(field string) string {
	switch strings.TrimSpace(field) {
	case "platform":
		return "X-Platform"
	case "version":
		return "X-Version"
	case "build":
		return "X-Build"
	case "device_no":
		return "X-Device-No"
	case "client_time":
		return "X-Client-Time"
	case "time_zone":
		return "X-Time-Zone"
	case "app_store_region":
		return "X-App-Store-Region"
	case "language":
		return "X-Language"
	default:
		return field
	}
}

func mapSwaggerRequest(method map[string]interface{}, fields map[string]string, definitions map[string]interface{}) {
	if len(fields) == 0 {
		return
	}
	parameters, _ := method["parameters"].([]interface{})
	for _, parameter := range parameters {
		param, ok := parameter.(map[string]interface{})
		if !ok || param["in"] != "body" {
			continue
		}
		schema, ok := param["schema"].(map[string]interface{})
		if !ok {
			continue
		}
		param["schema"] = mapSwaggerSchema(resolveSwaggerSchema(schema, definitions), stringFieldsToInterface(fields), definitions)
	}
}

func mapSwaggerResponse(method map[string]interface{}, fields map[string]interface{}, definitions map[string]interface{}) {
	if len(fields) == 0 {
		return
	}
	responses, _ := method["responses"].(map[string]interface{})
	response, _ := responses["200"].(map[string]interface{})
	schema, ok := response["schema"].(map[string]interface{})
	if !ok {
		return
	}
	response["schema"] = mapSwaggerSchema(resolveSwaggerSchema(schema, definitions), fields, definitions)
}

func mapSwaggerSchema(schema map[string]interface{}, fields map[string]interface{}, definitions map[string]interface{}) map[string]interface{} {
	if len(fields) == 0 {
		return schema
	}
	schema = flattenSwaggerAllOf(schema, definitions)
	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		return schema
	}

	required := mapSwaggerRequired(schema["required"], fields)
	if len(required) > 0 {
		schema["required"] = required
	}

	mappedProperties := make(map[string]interface{}, len(properties))
	for field, property := range properties {
		fieldMapping, hasMapping := fields[field]
		if !hasMapping {
			mappedProperties[field] = property
			continue
		}

		targetField := field
		nestedFields := map[string]interface{}(nil)
		switch mappingValue := fieldMapping.(type) {
		case string:
			if strings.TrimSpace(mappingValue) != "" {
				targetField = mappingValue
			}
		case map[string]interface{}:
			targetField = field
			nestedFields = mappingValue
			if mappedField, ok := mappingValue[nestedTargetFieldKey].(string); ok && strings.TrimSpace(mappedField) != "" {
				targetField = mappedField
			}
			nestedFields = withoutMetaField(mappingValue)
		}

		propertySchema, ok := property.(map[string]interface{})
		if ok && len(nestedFields) > 0 {
			property = mapSwaggerNestedSchema(propertySchema, nestedFields, definitions)
		}
		mappedProperties[targetField] = property
	}
	schema["properties"] = mappedProperties
	return schema
}

func mapSwaggerNestedSchema(schema map[string]interface{}, fields map[string]interface{}, definitions map[string]interface{}) map[string]interface{} {
	schema = resolveSwaggerSchema(schema, definitions)
	schema = flattenSwaggerAllOf(schema, definitions)
	if schema["type"] == "array" {
		if items, ok := schema["items"].(map[string]interface{}); ok {
			schema["items"] = mapSwaggerNestedSchema(items, fields, definitions)
		}
		return schema
	}
	if swaggerSchemaIsEmpty(schema) {
		return swaggerSchemaFromMapping(fields)
	}
	return mapSwaggerSchema(schema, fields, definitions)
}

func swaggerSchemaIsEmpty(schema map[string]interface{}) bool {
	_, hasRef := schema["$ref"]
	_, hasAllOf := schema["allOf"]
	_, hasProperties := schema["properties"]
	_, hasItems := schema["items"]
	_, hasType := schema["type"]
	return !hasRef && !hasAllOf && !hasProperties && !hasItems && !hasType
}

func swaggerSchemaFromMapping(fields map[string]interface{}) map[string]interface{} {
	properties := make(map[string]interface{})
	for sourceField, fieldMapping := range fields {
		targetField := sourceField
		var nestedFields map[string]interface{}
		switch mappingValue := fieldMapping.(type) {
		case string:
			if strings.TrimSpace(mappingValue) != "" {
				targetField = mappingValue
			}
		case map[string]interface{}:
			if mappedField, ok := mappingValue[nestedTargetFieldKey].(string); ok && strings.TrimSpace(mappedField) != "" {
				targetField = mappedField
			}
			nestedFields = withoutMetaField(mappingValue)
		}
		if targetField == nestedTargetFieldKey {
			continue
		}
		if len(nestedFields) > 0 {
			properties[targetField] = swaggerSchemaFromMapping(nestedFields)
			continue
		}
		properties[targetField] = map[string]interface{}{}
	}
	return map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}
}

func mapSwaggerRequired(requiredValue interface{}, fields map[string]interface{}) []interface{} {
	required, ok := requiredValue.([]interface{})
	if !ok {
		return nil
	}
	result := make([]interface{}, 0, len(required))
	for _, item := range required {
		field, ok := item.(string)
		if !ok {
			result = append(result, item)
			continue
		}
		if targetField := swaggerTargetField(field, fields); targetField != "" {
			result = append(result, targetField)
			continue
		}
		result = append(result, field)
	}
	return result
}

func swaggerTargetField(field string, fields map[string]interface{}) string {
	fieldMapping, ok := fields[field]
	if !ok {
		return ""
	}
	switch mappingValue := fieldMapping.(type) {
	case string:
		if strings.TrimSpace(mappingValue) != "" {
			return mappingValue
		}
	case map[string]interface{}:
		if mappedField, ok := mappingValue[nestedTargetFieldKey].(string); ok && strings.TrimSpace(mappedField) != "" {
			return mappedField
		}
	}
	return field
}

func flattenSwaggerAllOf(schema map[string]interface{}, definitions map[string]interface{}) map[string]interface{} {
	allOf, ok := schema["allOf"].([]interface{})
	if !ok {
		return schema
	}
	merged := make(map[string]interface{})
	for key, value := range schema {
		if key != "allOf" {
			merged[key] = value
		}
	}
	properties := make(map[string]interface{})
	var required []interface{}
	for _, item := range allOf {
		itemSchema, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		itemSchema = flattenSwaggerAllOf(resolveSwaggerSchema(itemSchema, definitions), definitions)
		if itemType, ok := itemSchema["type"]; ok {
			merged["type"] = itemType
		}
		if itemProperties, ok := itemSchema["properties"].(map[string]interface{}); ok {
			for key, value := range itemProperties {
				properties[key] = value
			}
		}
		if itemRequired, ok := itemSchema["required"].([]interface{}); ok {
			required = append(required, itemRequired...)
		}
	}
	if len(properties) > 0 {
		merged["properties"] = properties
	}
	if len(required) > 0 {
		merged["required"] = required
	}
	return merged
}

func resolveSwaggerSchema(schema map[string]interface{}, definitions map[string]interface{}) map[string]interface{} {
	ref, ok := schema["$ref"].(string)
	if !ok || !strings.HasPrefix(ref, "#/definitions/") {
		return cloneSwaggerValue(schema).(map[string]interface{})
	}
	name := strings.TrimPrefix(ref, "#/definitions/")
	definition, ok := definitions[name].(map[string]interface{})
	if !ok {
		return cloneSwaggerValue(schema).(map[string]interface{})
	}
	return cloneSwaggerValue(definition).(map[string]interface{})
}

func stringFieldsToInterface(fields map[string]string) map[string]interface{} {
	result := make(map[string]interface{}, len(fields))
	for key, value := range fields {
		result[key] = value
	}
	return result
}

func cloneSwaggerValue(value interface{}) interface{} {
	switch typed := value.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{}, len(typed))
		for key, item := range typed {
			result[key] = cloneSwaggerValue(item)
		}
		return result
	case []interface{}:
		result := make([]interface{}, 0, len(typed))
		for _, item := range typed {
			result = append(result, cloneSwaggerValue(item))
		}
		return result
	default:
		return typed
	}
}
