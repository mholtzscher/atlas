package jira_test

import (
	"bytes"
	"encoding/json"
	"testing"

	jiraops "github.com/mholtzscher/atlas/internal/jira"
)

func TestCompactIssue(t *testing.T) {
	t.Parallel()

	issue := json.RawMessage(`{
		"id":"10000",
		"key":"KAN-1",
		"self":"https://example/rest/api/3/issue/10000",
		"expand":"renderedFields,names,schema",
		"changelog":{"histories":[{"id":"1"}]},
		"fields":{
			"summary":"First",
			"status":{"id":"10000","name":"To Do","self":"https://example/status/10000"},
			"issuetype":{"id":"10003","name":"Task","iconUrl":"https://example/icon"},
			"priority":{"id":"3","name":"Medium","iconUrl":"https://example/priority"},
			"assignee":null,
			"reporter":{"accountId":"abc","displayName":"Jane Doe","emailAddress":"jane@example.com"},
			"project":{"id":"10000","key":"KAN","name":"KANBAN"},
			"created":"2026-02-28T12:29:13.441-0600",
			"updated":"2026-02-28T12:31:23.428-0600",
			"labels":["one","two"]
		}
	}`)

	compact, err := jiraops.CompactIssue(issue)
	if err != nil {
		t.Fatalf("jiraops.CompactIssue returned error: %v", err)
	}

	decoded := map[string]json.RawMessage{}
	decodeErr := json.Unmarshal(compact, &decoded)
	if decodeErr != nil {
		t.Fatalf("decode compact issue: %v", decodeErr)
	}

	if _, exists := decoded["self"]; exists {
		t.Fatalf("compact output unexpectedly contains self")
	}

	if _, exists := decoded["expand"]; exists {
		t.Fatalf("compact output unexpectedly contains expand")
	}

	assertRawTopLevel(t, decoded, "changelog", `{"histories":[{"id":"1"}]}`)

	fields := map[string]json.RawMessage{}
	fieldsErr := json.Unmarshal(decoded["fields"], &fields)
	if fieldsErr != nil {
		t.Fatalf("decode compact fields: %v", fieldsErr)
	}

	assertRawField(t, fields, "summary", `"First"`)
	assertRawField(t, fields, "status", `"To Do"`)
	assertRawField(t, fields, "issuetype", `"Task"`)
	assertRawField(t, fields, "priority", `"Medium"`)
	assertRawField(t, fields, "assignee", `null`)
	assertRawField(t, fields, "reporter", `"Jane Doe"`)
	assertRawField(t, fields, "project", `"KAN"`)
	assertRawField(t, fields, "created", `"2026-02-28T12:29:13.441-0600"`)
	assertRawField(t, fields, "updated", `"2026-02-28T12:31:23.428-0600"`)
	assertRawField(t, fields, "labels", `["one","two"]`)
}

func assertRawTopLevel(t *testing.T, root map[string]json.RawMessage, key string, want string) {
	t.Helper()

	got, exists := root[key]
	if !exists {
		t.Fatalf("missing top-level key %q", key)
	}

	if !bytes.Equal(bytes.TrimSpace(got), []byte(want)) {
		t.Fatalf("unexpected value for %q: got %s want %s", key, string(got), want)
	}
}

func TestCompactIssueInvalidJSON(t *testing.T) {
	t.Parallel()

	_, err := jiraops.CompactIssue(json.RawMessage(`{"id":`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func assertRawField(t *testing.T, fields map[string]json.RawMessage, key string, want string) {
	t.Helper()

	got, exists := fields[key]
	if !exists {
		t.Fatalf("missing field %q", key)
	}

	if !bytes.Equal(bytes.TrimSpace(got), []byte(want)) {
		t.Fatalf("unexpected value for %q: got %s want %s", key, string(got), want)
	}
}
