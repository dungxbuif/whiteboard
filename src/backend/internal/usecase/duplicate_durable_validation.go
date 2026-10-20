package usecase

import (
	"encoding/json"
	"math"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/google/uuid"
)

// Keep these types aligned with collab-contracts' sceneContentSchema. The
// library validator checks names and cross-references; this layer checks the
// JSON value types and bounds before a v1 copy becomes a durable scene.
var duplicateID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,256}$`)
var duplicateIndex = regexp.MustCompile(`^[A-Za-z0-9]+$`)
var duplicateColor = regexp.MustCompile(`^(?:#[0-9a-fA-F]{3,8}|[a-zA-Z]+|(?:rgb|rgba|hsl|hsla)\([0-9.,%\s+-]+\))$`)
var duplicateUnsafeMetadataKey = regexp.MustCompile(`(?i)^(?:html|srcdoc|script|dataurl|password|authorization|accesstoken|refreshtoken|on[a-z]+)$`)

func duplicateNumber(v any, min, max float64) bool {
	n, ok := v.(float64)
	return ok && !math.IsNaN(n) && !math.IsInf(n, 0) && n >= min && n <= max
}
func duplicateInteger(v any) bool {
	return duplicateNumber(v, 0, 9007199254740991) && math.Trunc(v.(float64)) == v.(float64)
}
func duplicateString(v any) bool { _, ok := v.(string); return ok }
func duplicateIDValue(v any) bool {
	value, ok := v.(string)
	return ok && duplicateID.MatchString(value)
}
func duplicateOptionalID(v any) bool { return v == nil || duplicateIDValue(v) }
func duplicateColorValue(v any) bool {
	value, ok := v.(string)
	return ok && len(value) <= 128 && duplicateColor.MatchString(value)
}
func duplicatePoint(v any) bool {
	point, ok := v.([]any)
	return ok && len(point) == 2 && duplicateNumber(point[0], -math.MaxFloat64, math.MaxFloat64) && duplicateNumber(point[1], -math.MaxFloat64, math.MaxFloat64)
}
func duplicatePoints(v any, min int) bool {
	points, ok := v.([]any)
	if !ok || len(points) < min {
		return false
	}
	for _, point := range points {
		if !duplicatePoint(point) {
			return false
		}
	}
	return true
}
func duplicateNullablePoint(v any) bool { return v == nil || duplicatePoint(v) }
func duplicateEnum(v any, choices ...string) bool {
	value, ok := v.(string)
	return ok && slices.Contains(choices, value)
}
func duplicateNullableEnum(v any, choices ...string) bool {
	return v == nil || duplicateEnum(v, choices...)
}

func validDuplicateAppStateField(key string, raw json.RawMessage) bool {
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return false
	}
	switch key {
	case "theme":
		return duplicateEnum(value, "light", "dark")
	case "gridModeEnabled":
		_, ok := value.(bool)
		return ok
	case "viewBackgroundColor":
		return duplicateColorValue(value)
	case "gridSize":
		return value == nil || duplicateNumber(value, math.SmallestNonzeroFloat64, math.MaxFloat64)
	case "gridStep":
		return duplicateNumber(value, math.SmallestNonzeroFloat64, math.MaxFloat64)
	default:
		return false
	}
}

func validDuplicateFileDescriptor(id string, raw json.RawMessage, requireAssetID bool) (libraryFileJSON, bool) {
	var file libraryFileJSON
	if !duplicateID.MatchString(id) {
		return file, false
	}
	var fields map[string]any
	if json.Unmarshal(raw, &fields) != nil || fields == nil {
		return file, false
	}
	for key := range fields {
		if !slices.Contains([]string{"id", "mimeType", "created", "assetId", "lastRetrieved", "version"}, key) {
			return file, false
		}
	}
	if fields["id"] != id || !duplicateEnum(fields["mimeType"], "image/png", "image/jpeg", "image/svg+xml", "image/gif", "image/webp", "image/avif") || !duplicateInteger(fields["created"]) {
		return file, false
	}
	for _, key := range []string{"lastRetrieved", "version"} {
		if value, ok := fields[key]; ok && !duplicateInteger(value) {
			return file, false
		}
	}
	if value, ok := fields["assetId"]; ok {
		assetID, valid := value.(string)
		if !valid || assetID == "" {
			return file, false
		}
		if _, err := uuid.Parse(assetID); err != nil {
			return file, false
		}
	} else if requireAssetID {
		return file, false
	}
	if json.Unmarshal(raw, &file) != nil {
		return file, false
	}
	return file, true
}

func validDuplicateNativeElement(raw json.RawMessage) bool {
	var element map[string]any
	if json.Unmarshal(raw, &element) != nil || element == nil || !duplicateIDValue(element["id"]) {
		return false
	}
	for _, key := range []string{"x", "y", "angle"} {
		if !duplicateNumber(element[key], -math.MaxFloat64, math.MaxFloat64) {
			return false
		}
	}
	for _, key := range []string{"width", "height", "strokeWidth", "roughness"} {
		if !duplicateNumber(element[key], 0, math.MaxFloat64) {
			return false
		}
	}
	if !duplicateNumber(element["opacity"], 0, 100) {
		return false
	}
	for _, key := range []string{"seed", "version", "versionNonce", "updated"} {
		if !duplicateInteger(element[key]) {
			return false
		}
	}
	if !duplicateColorValue(element["strokeColor"]) || !duplicateColorValue(element["backgroundColor"]) || !duplicateEnum(element["fillStyle"], "hachure", "cross-hatch", "solid", "zigzag") || !duplicateEnum(element["strokeStyle"], "solid", "dashed", "dotted") {
		return false
	}
	if _, ok := element["isDeleted"].(bool); !ok {
		return false
	}
	if _, ok := element["locked"].(bool); !ok {
		return false
	}
	if !duplicateOptionalID(element["frameId"]) {
		return false
	}
	if index := element["index"]; index != nil {
		value, ok := index.(string)
		if !ok || !duplicateIndex.MatchString(value) {
			return false
		}
	}
	groups, ok := element["groupIds"].([]any)
	if !ok {
		return false
	}
	seen := map[string]bool{}
	for _, entry := range groups {
		if !duplicateIDValue(entry) || seen[entry.(string)] {
			return false
		}
		seen[entry.(string)] = true
	}
	if refs := element["boundElements"]; refs != nil {
		entries, ok := refs.([]any)
		if !ok {
			return false
		}
		for _, raw := range entries {
			entry, ok := raw.(map[string]any)
			if !ok || len(entry) != 2 || !duplicateIDValue(entry["id"]) || !duplicateEnum(entry["type"], "arrow", "text") {
				return false
			}
		}
	}
	if metadata, exists := element["customData"]; exists {
		if _, ok := metadata.(map[string]any); !ok || !validDuplicateMetadata(metadata, 0) {
			return false
		}
	}
	if roundness := element["roundness"]; roundness != nil {
		value, ok := roundness.(map[string]any)
		if !ok || !duplicateInteger(value["type"]) || !slices.Contains([]float64{1, 2, 3}, value["type"].(float64)) {
			return false
		}
		for key, field := range value {
			if key != "type" && key != "value" || key == "value" && !duplicateNumber(field, -math.MaxFloat64, math.MaxFloat64) {
				return false
			}
		}
	}
	if link := element["link"]; link != nil {
		value, ok := link.(string)
		if !ok || !validDuplicateLink(value) {
			return false
		}
	}
	switch element["type"] {
	case "rectangle", "diamond", "ellipse":
		return true
	case "image":
		return validDuplicateImageFields(element)
	case "frame", "magicframe":
		return element["name"] == nil || duplicateString(element["name"])
	case "text":
		return duplicateNumber(element["fontSize"], math.SmallestNonzeroFloat64, math.MaxFloat64) && duplicateInteger(element["fontFamily"]) && duplicateString(element["text"]) && duplicateString(element["originalText"]) && duplicateEnum(element["textAlign"], "left", "center", "right") && duplicateEnum(element["verticalAlign"], "top", "middle", "bottom") && duplicateOptionalID(element["containerId"]) && duplicateBool(element["autoResize"]) && duplicateNumber(element["lineHeight"], math.SmallestNonzeroFloat64, math.MaxFloat64)
	case "line", "arrow":
		return validDuplicateLinearFields(element)
	case "freedraw":
		if !duplicatePoints(element["points"], 1) || !duplicateNullablePoint(element["lastCommittedPoint"]) || !duplicateBool(element["simulatePressure"]) {
			return false
		}
		pressures, ok := element["pressures"].([]any)
		if !ok {
			return false
		}
		for _, pressure := range pressures {
			if !duplicateNumber(pressure, 0, 1) {
				return false
			}
		}
		return len(pressures) == len(element["points"].([]any)) || element["simulatePressure"] == true && len(pressures) == 0
	default:
		return false
	}
}

func validDuplicateLink(value string) bool {
	if len(value) > 4096 || strings.IndexFunc(value, func(r rune) bool { return r < 32 || unicode.IsSpace(r) || r == '\\' }) >= 0 {
		return false
	}
	if strings.HasPrefix(value, "/") {
		if strings.HasPrefix(value, "//") {
			return false
		}
		decoded, err := url.PathUnescape(value)
		if err != nil {
			return false
		}
		for _, part := range strings.FieldsFunc(decoded, func(r rune) bool { return r == '/' || r == '?' || r == '#' }) {
			if part == ".." {
				return false
			}
		}
		return true
	}
	parsed, err := url.Parse(value)
	return err == nil && parsed.User == nil && slices.Contains([]string{"http", "https", "mailto"}, parsed.Scheme) && (parsed.Scheme == "mailto" || parsed.Host != "")
}

func duplicateBool(v any) bool { _, ok := v.(bool); return ok }
func validDuplicateMetadata(value any, depth int) bool {
	if depth > 32 {
		return false
	}
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			if key == "__proto__" || key == "prototype" || key == "constructor" || duplicateUnsafeMetadataKey.MatchString(key) || !validDuplicateMetadata(item, depth+1) {
				return false
			}
		}
	case []any:
		for _, item := range typed {
			if !validDuplicateMetadata(item, depth+1) {
				return false
			}
		}
	case nil, string, bool:
		return true
	case float64:
		return !math.IsNaN(typed) && !math.IsInf(typed, 0)
	default:
		return false
	}
	return true
}
func validDuplicateImageFields(element map[string]any) bool {
	if !duplicateOptionalID(element["fileId"]) || !duplicateEnum(element["status"], "pending", "saved", "error") {
		return false
	}
	scale, ok := element["scale"].([]any)
	if !ok || len(scale) != 2 || !(duplicateNumber(scale[0], -1, -1) || duplicateNumber(scale[0], 1, 1)) || !(duplicateNumber(scale[1], -1, -1) || duplicateNumber(scale[1], 1, 1)) {
		return false
	}
	if crop := element["crop"]; crop != nil {
		value, ok := crop.(map[string]any)
		if !ok || len(value) != 6 {
			return false
		}
		for _, key := range []string{"x", "y", "width", "height"} {
			if !duplicateNumber(value[key], 0, math.MaxFloat64) {
				return false
			}
		}
		for _, key := range []string{"naturalWidth", "naturalHeight"} {
			if !duplicateNumber(value[key], math.SmallestNonzeroFloat64, math.MaxFloat64) {
				return false
			}
		}
	}
	return true
}
func validDuplicateBinding(v any) bool {
	if v == nil {
		return true
	}
	value, ok := v.(map[string]any)
	if !ok || !duplicateIDValue(value["elementId"]) || !duplicateNumber(value["focus"], -1, 1) || !duplicateNumber(value["gap"], 0, math.MaxFloat64) {
		return false
	}
	for key, field := range value {
		switch key {
		case "elementId", "focus", "gap":
		case "fixedPoint":
			if !duplicatePoint(field) {
				return false
			}
		default:
			return false
		}
	}
	return len(value) == 3 || len(value) == 4
}
func validDuplicateLinearFields(element map[string]any) bool {
	if !duplicatePoints(element["points"], 2) || !duplicateNullablePoint(element["lastCommittedPoint"]) || !validDuplicateBinding(element["startBinding"]) || !validDuplicateBinding(element["endBinding"]) {
		return false
	}
	arrows := []string{"arrow", "bar", "dot", "circle", "circle_outline", "triangle", "triangle_outline", "diamond", "diamond_outline", "crowfoot_one", "crowfoot_many", "crowfoot_one_or_many"}
	if !duplicateNullableEnum(element["startArrowhead"], arrows...) || !duplicateNullableEnum(element["endArrowhead"], arrows...) {
		return false
	}
	if element["type"] == "arrow" {
		if elbowed, ok := element["elbowed"]; ok && !duplicateBool(elbowed) {
			return false
		}
		for _, key := range []string{"startIsSpecial", "endIsSpecial"} {
			if v, ok := element[key]; ok && v != nil && !duplicateBool(v) {
				return false
			}
		}
		if segments, ok := element["fixedSegments"]; ok && segments != nil {
			list, valid := segments.([]any)
			if !valid {
				return false
			}
			for _, raw := range list {
				segment, valid := raw.(map[string]any)
				if !valid || len(segment) != 3 || !duplicatePoint(segment["start"]) || !duplicatePoint(segment["end"]) || !duplicateInteger(segment["index"]) {
					return false
				}
			}
		}
	}
	return true
}
