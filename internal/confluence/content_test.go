package confluence_test

import (
	"errors"
	"testing"

	"github.com/mholtzscher/atlas/internal/atlaserr"
	confluenceops "github.com/mholtzscher/atlas/internal/confluence"
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

func TestConvertToMarkdown(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		html        string
		expectedMD  string
		expectError bool
	}{
		{
			name:       "converts heading",
			html:       "<h1>Hello World</h1>",
			expectedMD: "# Hello World",
		},
		{
			name:       "converts paragraph",
			html:       "<p>This is a paragraph.</p>",
			expectedMD: "This is a paragraph.",
		},
		{
			name:       "converts strong text",
			html:       "<p>This is <strong>bold</strong> text.</p>",
			expectedMD: "This is **bold** text.",
		},
		{
			name:       "converts emphasis",
			html:       "<p>This is <em>italic</em> text.</p>",
			expectedMD: "This is *italic* text.",
		},
		{
			name:       "converts link",
			html:       `<a href="https://example.com">Link text</a>`,
			expectedMD: "[Link text](https://example.com)",
		},
		{
			name:       "converts unordered list",
			html:       "<ul><li>Item 1</li><li>Item 2</li></ul>",
			expectedMD: "- Item 1\n- Item 2",
		},
		{
			name:       "converts ordered list",
			html:       "<ol><li>First</li><li>Second</li></ol>",
			expectedMD: "1. First\n2. Second",
		},
		{
			name:       "converts code block",
			html:       "<pre><code>func main() {}</code></pre>",
			expectedMD: "```\nfunc main() {}\n```",
		},
		{
			name:       "converts inline code",
			html:       "<p>Use <code>print()</code> function.</p>",
			expectedMD: "Use `print()` function.",
		},
		{
			name:       "handles empty string",
			html:       "",
			expectedMD: "",
		},
		{
			name:       "handles plain text",
			html:       "Just plain text.",
			expectedMD: "Just plain text.",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			markdown, err := confluenceops.ConvertToMarkdown(testCase.html)

			if testCase.expectError {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}

			if err != nil {
				t.Fatalf("ConvertToMarkdown returned error: %v", err)
			}

			if markdown != testCase.expectedMD {
				t.Fatalf("expected markdown %q, got %q", testCase.expectedMD, markdown)
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

	if atlasError.Message != expectedMessage {
		t.Fatalf("expected message %q, got %q", expectedMessage, atlasError.Message)
	}
}
