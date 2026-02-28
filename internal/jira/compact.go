package jira

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type issueEnvelope struct {
	ID     string                     `json:"id"`
	Key    string                     `json:"key"`
	Fields map[string]json.RawMessage `json:"fields"`
}

// CompactIssue projects a Jira issue into a token-efficient shape.
func CompactIssue(issue json.RawMessage) (json.RawMessage, error) {
	root := map[string]json.RawMessage{}
	if err := json.Unmarshal(issue, &root); err != nil {
		return nil, fmt.Errorf("decode issue: %w", err)
	}

	envelope := issueEnvelope{}
	if err := json.Unmarshal(issue, &envelope); err != nil {
		return nil, fmt.Errorf("decode issue: %w", err)
	}

	compactFields := make(map[string]json.RawMessage, len(envelope.Fields))
	for fieldName, fieldValue := range envelope.Fields {
		compactFields[fieldName] = compactFieldValue(fieldName, fieldValue)
	}

	compact := map[string]json.RawMessage{}

	idBytes, idErr := json.Marshal(envelope.ID)
	if idErr != nil {
		return nil, fmt.Errorf("encode compact issue id: %w", idErr)
	}

	keyBytes, keyErr := json.Marshal(envelope.Key)
	if keyErr != nil {
		return nil, fmt.Errorf("encode compact issue key: %w", keyErr)
	}

	fieldsBytes, fieldsErr := json.Marshal(compactFields)
	if fieldsErr != nil {
		return nil, fmt.Errorf("encode compact issue fields: %w", fieldsErr)
	}

	compact["id"] = json.RawMessage(idBytes)
	compact["key"] = json.RawMessage(keyBytes)
	compact["fields"] = json.RawMessage(fieldsBytes)

	for rootKey, rootValue := range root {
		if rootKey == "id" || rootKey == "key" || rootKey == "fields" || rootKey == "self" || rootKey == "expand" {
			continue
		}

		compact[rootKey] = rootValue
	}

	body, err := json.Marshal(compact)
	if err != nil {
		return nil, fmt.Errorf("encode compact issue: %w", err)
	}

	return json.RawMessage(body), nil
}

func compactFieldValue(fieldName string, value json.RawMessage) json.RawMessage {
	trimmed := json.RawMessage(bytes.TrimSpace(value))
	if len(trimmed) == 0 {
		return trimmed
	}

	switch fieldName {
	case defaultFieldStatus, defaultFieldIssueType, defaultFieldPriority:
		return firstObjectValue(trimmed, "name", "id")
	case defaultFieldAssignee, defaultFieldReporter:
		return firstObjectValue(trimmed, "displayName", "accountId", "id")
	case defaultFieldProject:
		return firstObjectValue(trimmed, "key", "name", "id")
	default:
		return trimmed
	}
}

func firstObjectValue(value json.RawMessage, keys ...string) json.RawMessage {
	object := map[string]json.RawMessage{}
	if err := json.Unmarshal(value, &object); err != nil {
		return value
	}

	for _, key := range keys {
		if candidate, exists := object[key]; exists {
			return json.RawMessage(bytes.TrimSpace(candidate))
		}
	}

	return value
}
