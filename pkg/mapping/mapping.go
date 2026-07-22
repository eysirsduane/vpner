package mapping

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"
	"sync"

	"just-vpn/pkg/setting"
)

type structField struct {
	Name      string
	Index     []int
	OmitEmpty bool
}

type RouteMapping struct {
	Path     string                 `json:"path"`
	Request  map[string]string      `json:"request"`
	Response map[string]interface{} `json:"response"`
}

type FieldAffix struct {
	Prefix string `json:"prefix"`
	Suffix string `json:"suffix"`
}

type Config struct {
	ProductCode string                  `json:"product_code"`
	SwaggerPath string                  `json:"swagger_path"`
	Headers     map[string]string       `json:"headers"`
	FieldAffix  FieldAffix              `json:"field_affix"`
	Routes      map[string]RouteMapping `json:"routes"`
}

const nestedTargetFieldKey = "$field"

var RouteConfig = &Config{
	Routes: map[string]RouteMapping{},
}

var structFieldsCache sync.Map

func init() {
	for _, path := range candidatePaths(setting.AppConfig.MappingFile) {
		if err := Load(path); err == nil {
			return
		}
	}
	log.Fatalf("Fail to load mapping file %q", setting.AppConfig.MappingFile)
}

func candidatePaths(path string) []string {
	path = strings.TrimSpace(path)
	if path == "" {
		path = "conf/mapping.json"
	}
	if strings.HasPrefix(path, "/") {
		return []string{path}
	}
	return []string{path, "../" + path, "../../" + path}
}

func Load(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	cfg := &Config{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return err
	}
	if err := validateIdentityConfig(cfg); err != nil {
		return err
	}
	if cfg.Routes == nil {
		cfg.Routes = map[string]RouteMapping{}
	}
	if cfg.Headers == nil {
		cfg.Headers = map[string]string{}
	}
	applyFieldAffix(cfg)

	RouteConfig = cfg
	return nil
}

func validateIdentityConfig(cfg *Config) error {
	if strings.TrimSpace(cfg.ProductCode) == "" {
		return fmt.Errorf("mapping product_code is required")
	}
	if !strings.HasPrefix(strings.TrimSpace(cfg.SwaggerPath), "/") {
		return fmt.Errorf("mapping swagger_path must start with /")
	}
	return nil
}

// ProductCode returns the immutable code used by transfer codes and daily statistics
func ProductCode() string {
	return strings.TrimSpace(RouteConfig.ProductCode)
}

// SwaggerPath returns the externally exposed Swagger UI path for the active product
func SwaggerPath() string {
	return strings.TrimSpace(RouteConfig.SwaggerPath)
}

func applyFieldAffix(cfg *Config) {
	if cfg == nil || (cfg.FieldAffix.Prefix == "" && cfg.FieldAffix.Suffix == "") {
		return
	}
	for originalPath, route := range cfg.Routes {
		route.Request = applyRequestFieldAffix(route.Request, cfg.FieldAffix)
		route.Response = applyResponseFieldAffix(route.Response, cfg.FieldAffix)
		cfg.Routes[originalPath] = route
	}
}

func applyRequestFieldAffix(fields map[string]string, affix FieldAffix) map[string]string {
	if len(fields) == 0 {
		return fields
	}
	result := make(map[string]string, len(fields))
	for sourceField, targetField := range fields {
		result[sourceField] = affixedField(targetField, affix)
	}
	return result
}

func applyResponseFieldAffix(fields map[string]interface{}, affix FieldAffix) map[string]interface{} {
	if len(fields) == 0 {
		return fields
	}
	result := make(map[string]interface{}, len(fields))
	for sourceField, fieldMapping := range fields {
		switch mappingValue := fieldMapping.(type) {
		case string:
			result[sourceField] = affixedField(mappingValue, affix)
		case map[string]interface{}:
			result[sourceField] = applyResponseFieldAffix(mappingValue, affix)
		default:
			result[sourceField] = fieldMapping
		}
	}
	return result
}

func affixedField(field string, affix FieldAffix) string {
	field = strings.TrimSpace(field)
	if field == "" {
		return field
	}
	return affix.Prefix + field + affix.Suffix
}

func Endpoint(originalPath string) string {
	route, ok := RouteConfig.Routes[originalPath]
	if !ok || route.Path == "" {
		return originalPath
	}
	return route.Path
}

func OriginalEndpoint(mappedPath string) string {
	for originalPath, route := range RouteConfig.Routes {
		if route.Path == mappedPath {
			return originalPath
		}
	}
	for originalPath, route := range RouteConfig.Routes {
		if route.Path != "" && strings.HasSuffix(mappedPath, route.Path) {
			return originalPath
		}
	}
	return mappedPath
}

func HeaderField(originalField string) string {
	if len(RouteConfig.Headers) == 0 {
		return originalField
	}
	mappedField, ok := RouteConfig.Headers[originalField]
	if !ok || mappedField == "" {
		return originalField
	}
	return mappedField
}

func RequestFields(originalPath string) map[string]string {
	if route, ok := RouteConfig.Routes[originalPath]; ok {
		return route.Request
	}
	return nil
}

func RequestField(originalPath string, originalField string) string {
	fields := RequestFields(originalPath)
	if len(fields) == 0 {
		return originalField
	}
	mappedField, ok := fields[originalField]
	if !ok || mappedField == "" {
		return originalField
	}
	return mappedField
}

func ResponseFields(originalPath string) map[string]interface{} {
	if route, ok := RouteConfig.Routes[originalPath]; ok {
		return route.Response
	}
	return nil
}

func MapFields(fields map[string]interface{}, source map[string]interface{}) map[string]interface{} {
	if len(fields) == 0 {
		return source
	}
	return mapObject(source, fields)
}

func MapResponse(originalPath string, source interface{}) map[string]interface{} {
	return MapFields(ResponseFields(originalPath), normalizeMap(source))
}

func normalizeMap(source interface{}) map[string]interface{} {
	switch value := source.(type) {
	case nil:
		return map[string]interface{}{}
	case map[string]interface{}:
		return normalizeValue(value).(map[string]interface{})
	default:
		normalized := normalizeValue(value)
		if result, ok := normalized.(map[string]interface{}); ok {
			return result
		}
		return map[string]interface{}{}
	}
}

func normalizeValue(value interface{}) interface{} {
	return normalizeReflectValue(reflect.ValueOf(value))
}

func cloneMap(source map[string]interface{}) map[string]interface{} {
	target := make(map[string]interface{}, len(source))
	for key, value := range source {
		target[key] = value
	}
	return target
}

func mapObject(source map[string]interface{}, fields map[string]interface{}) map[string]interface{} {
	target := cloneMap(source)
	for sourceField, fieldMapping := range fields {
		sourceValue, ok := source[sourceField]
		if !ok {
			continue
		}
		switch mappingValue := fieldMapping.(type) {
		case string:
			targetField := strings.TrimSpace(mappingValue)
			if targetField == "" {
				targetField = sourceField
			}
			target[targetField] = sourceValue
			if targetField != sourceField {
				delete(target, sourceField)
			}
		case map[string]interface{}:
			targetField := sourceField
			nestedFields := mappingValue
			if mappedField, ok := mappingValue[nestedTargetFieldKey].(string); ok {
				mappedField = strings.TrimSpace(mappedField)
				if mappedField != "" {
					targetField = mappedField
				}
				nestedFields = withoutMetaField(mappingValue)
			}
			target[targetField] = mapNestedValue(sourceValue, nestedFields)
			if targetField != sourceField {
				delete(target, sourceField)
			}
		}
	}
	return target
}

func withoutMetaField(fields map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(fields))
	for key, value := range fields {
		if key == nestedTargetFieldKey {
			continue
		}
		result[key] = value
	}
	return result
}

func mapNestedValue(source interface{}, fields map[string]interface{}) interface{} {
	switch value := source.(type) {
	case map[string]interface{}:
		return mapObject(value, fields)
	case []interface{}:
		result := make([]interface{}, 0, len(value))
		for _, item := range value {
			result = append(result, mapNestedValue(item, fields))
		}
		return result
	default:
		return source
	}
}

func normalizeReflectValue(value reflect.Value) interface{} {
	if !value.IsValid() {
		return nil
	}
	for value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}

	switch value.Kind() {
	case reflect.Struct:
		return normalizeStruct(value)
	case reflect.Map:
		return normalizeMapValue(value)
	case reflect.Slice, reflect.Array:
		return normalizeSliceValue(value)
	case reflect.Bool:
		return value.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return value.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return value.Uint()
	case reflect.Float32, reflect.Float64:
		return value.Float()
	case reflect.String:
		return value.String()
	default:
		if value.CanInterface() {
			return value.Interface()
		}
		return nil
	}
}

func normalizeStruct(value reflect.Value) map[string]interface{} {
	fields := cachedStructFields(value.Type())
	result := make(map[string]interface{}, len(fields))
	for _, field := range fields {
		fieldValue := value.FieldByIndex(field.Index)
		if field.OmitEmpty && isEmptyValue(fieldValue) {
			continue
		}
		result[field.Name] = normalizeReflectValue(fieldValue)
	}
	return result
}

func normalizeMapValue(value reflect.Value) map[string]interface{} {
	if value.IsNil() {
		return map[string]interface{}{}
	}
	result := make(map[string]interface{}, value.Len())
	iter := value.MapRange()
	for iter.Next() {
		keyValue := normalizeReflectValue(iter.Key())
		key, ok := keyValue.(string)
		if !ok {
			key = strings.TrimSpace(toString(keyValue))
		}
		if key == "" {
			continue
		}
		result[key] = normalizeReflectValue(iter.Value())
	}
	return result
}

func normalizeSliceValue(value reflect.Value) []interface{} {
	if value.Kind() == reflect.Slice && value.IsNil() {
		return []interface{}{}
	}
	result := make([]interface{}, 0, value.Len())
	for index := 0; index < value.Len(); index++ {
		result = append(result, normalizeReflectValue(value.Index(index)))
	}
	return result
}

func cachedStructFields(valueType reflect.Type) []structField {
	if cached, ok := structFieldsCache.Load(valueType); ok {
		return cached.([]structField)
	}
	fields := collectStructFields(valueType)
	structFieldsCache.Store(valueType, fields)
	return fields
}

func collectStructFields(valueType reflect.Type) []structField {
	fields := make([]structField, 0, valueType.NumField())
	for index := 0; index < valueType.NumField(); index++ {
		field := valueType.Field(index)
		if field.PkgPath != "" && !field.Anonymous {
			continue
		}
		if field.Anonymous && field.Type.Kind() == reflect.Struct && field.Tag.Get("json") == "" {
			fields = append(fields, collectStructFields(field.Type)...)
			continue
		}
		name, omitEmpty, ok := jsonFieldName(field)
		if !ok {
			continue
		}
		fields = append(fields, structField{
			Name:      name,
			Index:     field.Index,
			OmitEmpty: omitEmpty,
		})
	}
	return fields
}

func jsonFieldName(field reflect.StructField) (string, bool, bool) {
	tag := field.Tag.Get("json")
	if tag == "-" {
		return "", false, false
	}
	parts := strings.Split(tag, ",")
	name := strings.TrimSpace(parts[0])
	if name == "" {
		name = field.Name
	}
	omitEmpty := false
	for _, part := range parts[1:] {
		if strings.TrimSpace(part) == "omitempty" {
			omitEmpty = true
			break
		}
	}
	return name, omitEmpty, true
}

func isEmptyValue(value reflect.Value) bool {
	if !value.IsValid() {
		return true
	}
	switch value.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return value.Len() == 0
	case reflect.Bool:
		return !value.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return value.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return value.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return value.Float() == 0
	case reflect.Interface, reflect.Pointer:
		return value.IsNil()
	}
	return false
}

func toString(value interface{}) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	return strings.TrimSpace(fmt.Sprint(value))
}
