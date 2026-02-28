package confluence_test

import (
	"bytes"
	"encoding/json"
	"testing"

	confluenceops "github.com/mholtzscher/atlas/internal/confluence"
)

func TestCompactRecordDropsNoisyFieldsAndCollapsesNestedObjects(t *testing.T) {
	t.Parallel()

	record := json.RawMessage(`{
		"id":"123",
		"title":"Page title",
		"status":"current",
		"space":{"id":"99","key":"ENG","name":"Engineering","_links":{"webui":"/spaces/ENG"}},
		"author":{"accountId":"abc123","displayName":"Jane Doe","email":"jane@example.com"},
		"body":{"view":{"value":"<p>big payload</p>"}},
		"_links":{"webui":"/wiki/spaces/ENG/pages/123"}
	}`)

	compact, err := confluenceops.CompactRecord(record)
	if err != nil {
		t.Fatalf("CompactRecord returned error: %v", err)
	}

	decoded := map[string]json.RawMessage{}
	decodeErr := json.Unmarshal(compact, &decoded)
	if decodeErr != nil {
		t.Fatalf("decode compact record: %v", decodeErr)
	}

	if _, exists := decoded["_links"]; exists {
		t.Fatalf("compact output unexpectedly contains _links")
	}

	if _, exists := decoded["body"]; exists {
		t.Fatalf("compact output unexpectedly contains body")
	}

	assertRawValue(t, decoded, "id", `"123"`)
	assertRawValue(t, decoded, "title", `"Page title"`)
	assertRawValue(t, decoded, "status", `"current"`)
	assertRawValue(t, decoded, "space", `"ENG"`)
	assertRawValue(t, decoded, "author", `"Jane Doe"`)
}

func TestCompactRecordInvalidJSON(t *testing.T) {
	t.Parallel()

	_, err := confluenceops.CompactRecord(json.RawMessage(`{"id":`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func assertRawValue(t *testing.T, object map[string]json.RawMessage, key string, want string) {
	t.Helper()

	got, exists := object[key]
	if !exists {
		t.Fatalf("missing key %q", key)
	}

	if !bytes.Equal(bytes.TrimSpace(got), []byte(want)) {
		t.Fatalf("unexpected value for %q: got %s want %s", key, string(got), want)
	}
}
