package confluence_test

import (
	"errors"
	"testing"

	"github.com/mholtzscher/atlas/internal/atlaserr"
	confluenceops "github.com/mholtzscher/atlas/internal/confluence"
	"github.com/mholtzscher/atlas/internal/ops"
)

func TestExtractPageViewHTML(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		payload         string
		expectedHTML    string
		expectedMessage string
	}{
		{
			name:         "extracts body view value",
			payload:      `{"body":{"view":{"value":"<h1>Hello</h1>"}}}`,
			expectedHTML: "<h1>Hello</h1>",
		},
		{
			name:            "missing body returns structured error",
			payload:         `{}`,
			expectedMessage: "page body view missing",
		},
		{
			name:            "missing view returns structured error",
			payload:         `{"body":{}}`,
			expectedMessage: "page body view missing",
		},
		{
			name:            "non string value returns structured error",
			payload:         `{"body":{"view":{"value":123}}}`,
			expectedMessage: "page body view missing",
		},
		{
			name:            "invalid page json returns structured error",
			payload:         `{`,
			expectedMessage: "invalid Confluence page JSON",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			html, err := confluenceops.ExtractPageViewHTML([]byte(testCase.payload))

			if testCase.expectedMessage != "" {
				assertPageViewError(t, err, testCase.expectedMessage)
				return
			}

			if err != nil {
				t.Fatalf("ExtractPageViewHTML returned error: %v", err)
			}

			if html != testCase.expectedHTML {
				t.Fatalf("expected html %q, got %q", testCase.expectedHTML, html)
			}
		})
	}
}

func assertPageViewError(t *testing.T, err error, expectedMessage string) {
	t.Helper()

	if err == nil {
		t.Fatal("expected error")
	}

	var atlasError *atlaserr.Error
	if !errors.As(err, &atlasError) {
		t.Fatalf("expected *atlaserr.Error, got %T", err)
	}

	if atlasError.Code != atlaserr.CodeUpstreamError {
		t.Fatalf("expected code %s, got %s", atlaserr.CodeUpstreamError, atlasError.Code)
	}

	if atlasError.Op != ops.OpConfluencePageView {
		t.Fatalf("expected op %s, got %s", ops.OpConfluencePageView, atlasError.Op)
	}

	if atlasError.Message != expectedMessage {
		t.Fatalf("expected message %q, got %q", expectedMessage, atlasError.Message)
	}
}
