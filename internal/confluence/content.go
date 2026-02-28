package confluence

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/go-xmlfmt/xmlfmt"

	"github.com/mholtzscher/atlas/internal/atlaserr"
)

var (
	// localIDRegex matches local-id="..." attributes in HTML.
	localIDRegex = regexp.MustCompile(`\s+local-id="[^"]*"`)
	// acLocalIDRegex matches ac:local-id="..." attributes.
	acLocalIDRegex = regexp.MustCompile(`\s+ac:local-id="[^"]*"`)
	// acNameRegex matches ac:name="..." attributes (often noise in storage format).
	acNameRegex = regexp.MustCompile(`\s+ac:name="[^"]*"`)
)

// ExtractPageViewHTML extracts body.storage.value HTML from a Confluence page payload.
// Using storage format which is cleaner than view format and lacks editor artifacts.
// Returns cleaned HTML with Confluence-specific attributes removed.
func ExtractPageViewHTML(page json.RawMessage) (string, error) {
	fields := map[string]json.RawMessage{}
	if err := json.Unmarshal(page, &fields); err != nil {
		return "", atlaserr.New(
			atlaserr.CodeUpstreamError,
			"invalid Confluence page JSON",
			false,
			nil,
		)
	}

	body, bodyExists := fields["body"]
	if !bodyExists {
		return "", pageBodyViewMissingError()
	}

	bodyFields := map[string]json.RawMessage{}
	if err := json.Unmarshal(body, &bodyFields); err != nil {
		return "", pageBodyViewMissingError()
	}

	// Try storage format first (cleaner, no editor artifacts)
	if storageHTML, found := extractFromStorage(bodyFields); found {
		return storageHTML, nil
	}

	// Fallback to view format
	return extractFromView(bodyFields)
}

// PrettyPrintHTML formats HTML with proper indentation for readability.
func PrettyPrintHTML(html string) string {
	return xmlfmt.FormatXML(html, "", "  ")
}

// CleanHTML removes Confluence-specific attributes that add noise to output.
// Strips local-id, ac:local-id, and ac:name attributes.
func CleanHTML(html string) string {
	// Remove attributes in order to avoid interference
	cleaned := localIDRegex.ReplaceAllString(html, "")
	cleaned = acLocalIDRegex.ReplaceAllString(cleaned, "")
	cleaned = acNameRegex.ReplaceAllString(cleaned, "")
	return cleaned
}

// ConvertToMarkdown converts HTML content to Markdown format.
// Uses the html-to-markdown library for robust conversion.
func ConvertToMarkdown(html string) (string, error) {
	markdown, err := htmltomarkdown.ConvertString(html)
	if err != nil {
		return "", fmt.Errorf("convert HTML to markdown: %w", err)
	}
	return markdown, nil
}

func extractFromStorage(bodyFields map[string]json.RawMessage) (string, bool) {
	storage, storageExists := bodyFields["storage"]
	if !storageExists {
		return "", false
	}

	storageFields := map[string]json.RawMessage{}
	if err := json.Unmarshal(storage, &storageFields); err != nil {
		return "", false
	}

	value, valueExists := storageFields["value"]
	if !valueExists {
		return "", false
	}

	html := ""
	if err := json.Unmarshal(value, &html); err != nil {
		return "", false
	}

	return CleanHTML(html), true
}

func extractFromView(bodyFields map[string]json.RawMessage) (string, error) {
	view, viewExists := bodyFields["view"]
	if !viewExists {
		return "", pageBodyViewMissingError()
	}

	viewFields := map[string]json.RawMessage{}
	if err := json.Unmarshal(view, &viewFields); err != nil {
		return "", pageBodyViewMissingError()
	}

	value, valueExists := viewFields["value"]
	if !valueExists {
		return "", pageBodyViewMissingError()
	}

	html := ""
	if err := json.Unmarshal(value, &html); err != nil {
		return "", pageBodyViewMissingError()
	}

	return CleanHTML(html), nil
}

func pageBodyViewMissingError() error {
	return atlaserr.New(
		atlaserr.CodeUpstreamError,
		"page body view missing",
		false,
		nil,
	)
}
