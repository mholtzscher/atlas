// Package confluence implements Confluence read operations.
package confluence

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/mholtzscher/atlas/internal/atlaserr"
	"github.com/mholtzscher/atlas/internal/atlassian"
	"github.com/mholtzscher/atlas/internal/ops"
)

const (
	pageGetPathPrefix = "/wiki/api/v2/pages/"
	pageSearchPath    = "/wiki/api/v2/content/search"
)

const (
	// BodyFormatNone avoids requesting body content.
	BodyFormatNone = "none"
	defaultLimit   = 25
)

type pageSearchResponse struct {
	Results []json.RawMessage `json:"results"`
	Links   struct {
		Next string `json:"next"`
	} `json:"_links"`
}

// SearchOptions controls optional Confluence page fields.
type SearchOptions struct {
	BodyFormat        string
	IncludeLabels     bool
	IncludeProperties bool
	IncludeOperations bool
	IncludeVersions   bool
}

// GetPageRequest defines page get inputs.
type GetPageRequest struct {
	SearchOptions

	PageID string
}

// SearchPagesRequest defines page search inputs.
type SearchPagesRequest struct {
	SearchOptions

	CQL      string
	Limit    int
	PageSize int
	Cursor   string
}

// GetPage fetches one page by ID.
func GetPage(
	ctx context.Context,
	client *atlassian.Client,
	request GetPageRequest,
) (json.RawMessage, error) {
	if request.PageID == "" {
		return nil, atlaserr.InvalidArgument("missing page ID", ops.OpConfluencePageGet)
	}

	body, err := client.Get(
		ctx,
		pageGetPathPrefix+url.PathEscape(request.PageID),
		buildQuery(request.SearchOptions),
		ops.OpConfluencePageGet,
	)
	if err != nil {
		return nil, err
	}

	if request.BodyFormat == BodyFormatNone {
		return removeBody(json.RawMessage(body), ops.OpConfluencePageGet)
	}

	return json.RawMessage(body), nil
}

// SearchPages streams pages from CQL search.
func SearchPages(
	ctx context.Context,
	client *atlassian.Client,
	request SearchPagesRequest,
	emit func(item json.RawMessage) error,
) error {
	if validateErr := validateSearchRequest(request); validateErr != nil {
		return validateErr
	}

	if request.Limit <= 0 {
		return nil
	}

	remaining := request.Limit
	nextURL := buildInitialSearchURL(request)
	stripBody := request.BodyFormat == BodyFormatNone

	for remaining > 0 && nextURL != "" {
		response, pageErr := getSearchPage(ctx, client, nextURL)
		if pageErr != nil {
			return pageErr
		}

		var emitErr error
		remaining, emitErr = emitSearchResults(response.Results, stripBody, remaining, emit)
		if emitErr != nil {
			return emitErr
		}

		if remaining == 0 {
			return nil
		}

		nextURL = response.Links.Next
	}

	return nil
}

func validateSearchRequest(request SearchPagesRequest) error {
	if request.CQL == "" {
		return atlaserr.InvalidArgument("missing required --cql", ops.OpConfluencePageSearch)
	}

	if request.Limit < 0 {
		return atlaserr.InvalidArgument("--limit must be >= 0", ops.OpConfluencePageSearch)
	}

	if request.PageSize <= 0 {
		return atlaserr.InvalidArgument("--page-size must be > 0", ops.OpConfluencePageSearch)
	}

	return nil
}

func getSearchPage(ctx context.Context, client *atlassian.Client, requestURL string) (pageSearchResponse, error) {
	body, err := client.GetURL(ctx, requestURL, ops.OpConfluencePageSearch)
	if err != nil {
		return pageSearchResponse{}, err
	}

	return decodeSearchResponse(body)
}

func emitSearchResults(
	pages []json.RawMessage,
	stripBody bool,
	remaining int,
	emit func(item json.RawMessage) error,
) (int, error) {
	for _, page := range pages {
		cleanPage, cleanErr := maybeStripBody(page, stripBody)
		if cleanErr != nil {
			return remaining, cleanErr
		}

		if emitErr := emit(cleanPage); emitErr != nil {
			return remaining, fmt.Errorf("emit page: %w", emitErr)
		}

		remaining--
		if remaining == 0 {
			return 0, nil
		}
	}

	return remaining, nil
}

func maybeStripBody(page json.RawMessage, stripBody bool) (json.RawMessage, error) {
	if !stripBody {
		return page, nil
	}

	return removeBody(page, ops.OpConfluencePageSearch)
}

func buildInitialSearchURL(request SearchPagesRequest) string {
	query := buildQuery(request.SearchOptions)
	query.Set("cql", request.CQL)
	query.Set("limit", strconv.Itoa(min(request.Limit, request.PageSize)))
	if request.Cursor != "" {
		query.Set("cursor", request.Cursor)
	}

	resolvedURL := url.URL{Path: pageSearchPath, RawQuery: query.Encode()}
	return resolvedURL.String()
}

func buildQuery(options SearchOptions) url.Values {
	query := url.Values{}

	if options.BodyFormat != "" && options.BodyFormat != BodyFormatNone {
		query.Set("body-format", options.BodyFormat)
	}

	if options.IncludeLabels {
		query.Set("include-labels", strconv.FormatBool(true))
	}

	if options.IncludeProperties {
		query.Set("include-properties", strconv.FormatBool(true))
	}

	if options.IncludeOperations {
		query.Set("include-operations", strconv.FormatBool(true))
	}

	if options.IncludeVersions {
		query.Set("include-versions", strconv.FormatBool(true))
	}

	return query
}

func removeBody(page json.RawMessage, op string) (json.RawMessage, error) {
	fields := map[string]json.RawMessage{}
	if err := json.Unmarshal(page, &fields); err != nil {
		return nil, atlaserr.New(
			atlaserr.CodeUpstreamError,
			"invalid Confluence page JSON",
			op,
			false,
			nil,
		)
	}

	delete(fields, "body")

	cleaned, err := json.Marshal(fields)
	if err != nil {
		return nil, atlaserr.New(
			atlaserr.CodeUpstreamError,
			"marshal Confluence page JSON",
			op,
			false,
			nil,
		)
	}

	return cleaned, nil
}

func decodeSearchResponse(body []byte) (pageSearchResponse, error) {
	response := pageSearchResponse{}
	if err := json.Unmarshal(body, &response); err != nil {
		return pageSearchResponse{}, atlaserr.New(
			atlaserr.CodeUpstreamError,
			"invalid Confluence search response JSON",
			ops.OpConfluencePageSearch,
			false,
			nil,
		)
	}

	return response, nil
}
