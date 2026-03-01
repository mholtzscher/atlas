package confluence

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// CompactRecord projects Confluence payloads into a token-efficient shape.
func CompactRecord(record json.RawMessage) (json.RawMessage, error) {
	return compactValue(record, 0)
}

func compactValue(value json.RawMessage, depth int) (json.RawMessage, error) {
	trimmed := json.RawMessage(bytes.TrimSpace(value))
	if len(trimmed) == 0 {
		return trimmed, nil
	}

	switch trimmed[0] {
	case '{':
		return compactObject(trimmed, depth)
	case '[':
		return compactArray(trimmed, depth)
	default:
		return trimmed, nil
	}
}

func compactObject(value json.RawMessage, depth int) (json.RawMessage, error) {
	object := map[string]json.RawMessage{}
	if err := json.Unmarshal(value, &object); err != nil {
		return nil, fmt.Errorf("decode object: %w", err)
	}

	compactObject := make(map[string]json.RawMessage, len(object))
	for key, child := range object {
		if shouldDropCompactKey(key) {
			continue
		}

		compactChild, compactErr := compactValue(child, depth+1)
		if compactErr != nil {
			return nil, compactErr
		}

		compactObject[key] = compactChild
	}

	if depth > 0 {
		if scalar, collapse := tryCollapseObject(compactObject); collapse {
			return scalar, nil
		}
	}

	body, err := json.Marshal(compactObject)
	if err != nil {
		return nil, fmt.Errorf("encode compact object: %w", err)
	}

	return body, nil
}

func compactArray(value json.RawMessage, depth int) (json.RawMessage, error) {
	array := []json.RawMessage{}
	if err := json.Unmarshal(value, &array); err != nil {
		return nil, fmt.Errorf("decode array: %w", err)
	}

	compactArray := make([]json.RawMessage, 0, len(array))
	for _, child := range array {
		compactChild, compactErr := compactValue(child, depth+1)
		if compactErr != nil {
			return nil, compactErr
		}

		compactArray = append(compactArray, compactChild)
	}

	body, err := json.Marshal(compactArray)
	if err != nil {
		return nil, fmt.Errorf("encode compact array: %w", err)
	}

	return body, nil
}

func tryCollapseObject(object map[string]json.RawMessage) (json.RawMessage, bool) {
	if len(object) == 0 {
		return nil, false
	}

	if scalar, ok := firstScalarValue(object); ok {
		return scalar, true
	}

	return nil, false
}

func firstScalarValue(object map[string]json.RawMessage) (json.RawMessage, bool) {
	if scalar, ok := trimmedScalar(object, "key"); ok {
		return scalar, true
	}

	if scalar, ok := trimmedScalar(object, "name"); ok {
		return scalar, true
	}

	if scalar, ok := trimmedScalar(object, "title"); ok {
		return scalar, true
	}

	if scalar, ok := trimmedScalar(object, "displayName"); ok {
		return scalar, true
	}

	if scalar, ok := trimmedScalar(object, "accountId"); ok {
		return scalar, true
	}

	if scalar, ok := trimmedScalar(object, "id"); ok {
		return scalar, true
	}

	if scalar, ok := trimmedScalar(object, "number"); ok {
		return scalar, true
	}

	return nil, false
}

func trimmedScalar(object map[string]json.RawMessage, key string) (json.RawMessage, bool) {
	scalar, ok := object[key]
	if !ok {
		return nil, false
	}

	trimmed := json.RawMessage(bytes.TrimSpace(scalar))
	if len(trimmed) == 0 {
		return nil, false
	}

	return trimmed, true
}

func shouldDropCompactKey(key string) bool {
	switch key {
	case "_expandable", "_links", "avatarUrls", "icon", "iconUrl", "links", "self", "tinyui", "webui":
		return true
	default:
		return false
	}
}
