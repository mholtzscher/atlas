package confluence

import (
	"encoding/json"

	"github.com/go-xmlfmt/xmlfmt"

	"github.com/mholtzscher/atlas/internal/atlaserr"
	"github.com/mholtzscher/atlas/internal/ops"
)

// ExtractPageViewHTML extracts body.storage.value HTML from a Confluence page payload.
// Using storage format which is cleaner than view format and lacks editor artifacts.
func ExtractPageViewHTML(page json.RawMessage) (string, error) {
	fields := map[string]json.RawMessage{}
	if err := json.Unmarshal(page, &fields); err != nil {
		return "", atlaserr.New(
			atlaserr.CodeUpstreamError,
			"invalid Confluence page JSON",
			ops.OpConfluencePageView,
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

	return html, true
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

	return html, nil
}

func pageBodyViewMissingError() error {
	return atlaserr.New(
		atlaserr.CodeUpstreamError,
		"page body view missing",
		ops.OpConfluencePageView,
		false,
		nil,
	)
}
