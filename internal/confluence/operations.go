// Package confluence implements Confluence read operations.
package confluence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"

	"github.com/mholtzscher/atlas/internal/atlaserr"
	"github.com/mholtzscher/atlas/internal/atlassian"
)

const (
	pageGetPathPrefix              = "/wiki/api/v2/pages/"
	pageSearchPath                 = "/wiki/api/v2/content/search"
	spacesPath                     = "/wiki/api/v2/spaces"
	spacesGetPathPrefix            = "/wiki/api/v2/spaces/"
	pageFooterCommentsPathTemplate = "/wiki/api/v2/pages/%s/footer-comments"
	commentChildrenPathTemplate    = "/wiki/api/v2/footer-comments/%s/children"
)

const (
	// BodyFormatNone avoids requesting body content.
	BodyFormatNone = "none"
	// BodyFormatStorage returns XHTML storage format.
	BodyFormatStorage = "storage"
	// BodyFormatEditor returns editor format.
	BodyFormatEditor = "editor"
	// BodyFormatExportView returns export view format.
	BodyFormatExportView = "export_view"
	// BodyFormatView returns view format.
	BodyFormatView = "view"
	// BodyFormatAtlasDocFormat returns Atlas Document Format (ADF).
	BodyFormatAtlasDocFormat = "atlas_doc_format"
	defaultLimit             = 25
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
	Raw               bool
}

// GetPageRequest defines page describe/view inputs.
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

// ListSpacesRequest defines space list inputs.
type ListSpacesRequest struct {
	Limit    int
	PageSize int
	Cursor   string
}

// GetSpaceByKeyRequest defines space get inputs.
type GetSpaceByKeyRequest struct {
	SpaceKey string
}

// ListPageCommentsRequest defines page comments inputs.
type ListPageCommentsRequest struct {
	PageID     string
	Limit      int
	PageSize   int
	Cursor     string
	BodyFormat string
	Raw        bool
}

// GetPage fetches one page by ID.
func GetPage(
	ctx context.Context,
	client *atlassian.Client,
	request GetPageRequest,
) (json.RawMessage, error) {
	if request.PageID == "" {
		return nil, atlaserr.InvalidArgument("missing page ID")
	}

	body, err := client.Get(
		ctx,
		pageGetPathPrefix+url.PathEscape(request.PageID),
		buildQuery(request.SearchOptions),
	)
	if err != nil {
		return nil, err
	}

	if request.BodyFormat == BodyFormatNone && !request.Raw {
		return removeBody(json.RawMessage(body))
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
	stripBody := request.BodyFormat == BodyFormatNone && !request.Raw

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

// ListSpaces streams accessible spaces.
func ListSpaces(
	ctx context.Context,
	client *atlassian.Client,
	request ListSpacesRequest,
	emit func(item json.RawMessage) error,
) error {
	if request.Limit < 0 {
		return atlaserr.InvalidArgument("--limit must be >= 0")
	}

	if request.PageSize <= 0 {
		return atlaserr.InvalidArgument("--page-size must be > 0")
	}

	if request.Limit == 0 {
		return nil
	}

	remaining := request.Limit
	nextURL := buildInitialSpacesURL(request)

	for remaining > 0 && nextURL != "" {
		response, pageErr := getResultsPage(
			ctx,
			client,
			nextURL,
			"invalid Confluence spaces response JSON",
		)
		if pageErr != nil {
			return pageErr
		}

		for _, space := range response.Results {
			if emitErr := emit(space); emitErr != nil {
				return fmt.Errorf("emit space: %w", emitErr)
			}

			remaining--
			if remaining == 0 {
				return nil
			}
		}

		nextURL = response.Links.Next
	}

	return nil
}

// GetSpaceByKey fetches one space by key.
func GetSpaceByKey(
	ctx context.Context,
	client *atlassian.Client,
	request GetSpaceByKeyRequest,
) (json.RawMessage, error) {
	if request.SpaceKey == "" {
		return nil, atlaserr.InvalidArgument("missing space key")
	}

	lookupQuery := url.Values{}
	lookupQuery.Set("keys", request.SpaceKey)
	lookupQuery.Set("limit", "1")

	lookupBody, lookupErr := client.Get(ctx, spacesPath, lookupQuery)
	if lookupErr != nil {
		return nil, lookupErr
	}

	lookupResponse, decodeErr := decodeResultsPage(
		lookupBody,
		"invalid Confluence space lookup response JSON",
	)
	if decodeErr != nil {
		return nil, decodeErr
	}

	if len(lookupResponse.Results) == 0 {
		details, detailsErr := json.Marshal(struct {
			SpaceKey string `json:"spaceKey"`
		}{
			SpaceKey: request.SpaceKey,
		})
		if detailsErr != nil {
			details = nil
		}

		return nil, atlaserr.New(
			atlaserr.CodeNotFound,
			"space not found",
			false,
			details,
		)
	}

	spaceID, spaceIDErr := extractID(lookupResponse.Results[0])
	if spaceIDErr != nil {
		return nil, atlaserr.New(
			atlaserr.CodeUpstreamError,
			"space lookup result missing valid id",
			false,
			nil,
		)
	}

	body, getErr := client.Get(
		ctx,
		spacesGetPathPrefix+url.PathEscape(spaceID),
		nil,
	)
	if getErr != nil {
		return nil, getErr
	}

	return json.RawMessage(body), nil
}

// ListPageComments streams page footer comments and their threaded replies.
func ListPageComments(
	ctx context.Context,
	client *atlassian.Client,
	request ListPageCommentsRequest,
	emit func(item json.RawMessage) error,
) error {
	if request.PageID == "" {
		return atlaserr.InvalidArgument("missing page ID")
	}

	if request.Limit < 0 {
		return atlaserr.InvalidArgument("--limit must be >= 0")
	}

	if request.PageSize <= 0 {
		return atlaserr.InvalidArgument("--page-size must be > 0")
	}

	if request.Limit == 0 {
		return nil
	}

	remaining := request.Limit
	visitedCommentIDs := map[string]struct{}{}
	nextURL := buildInitialPageCommentsURL(request)

	for remaining > 0 && nextURL != "" {
		response, pageErr := getResultsPage(
			ctx,
			client,
			nextURL,
			"invalid Confluence page comments response JSON",
		)
		if pageErr != nil {
			return pageErr
		}

		for _, comment := range response.Results {
			var processErr error
			remaining, processErr = emitCommentTree(
				ctx,
				client,
				comment,
				request,
				remaining,
				visitedCommentIDs,
				emit,
			)
			if processErr != nil {
				return processErr
			}

			if remaining == 0 {
				return nil
			}
		}

		nextURL = response.Links.Next
	}

	return nil
}

func validateSearchRequest(request SearchPagesRequest) error {
	if request.CQL == "" {
		return atlaserr.InvalidArgument("missing required --cql")
	}

	if request.Limit < 0 {
		return atlaserr.InvalidArgument("--limit must be >= 0")
	}

	if request.PageSize <= 0 {
		return atlaserr.InvalidArgument("--page-size must be > 0")
	}

	return nil
}

func getSearchPage(ctx context.Context, client *atlassian.Client, requestURL string) (pageSearchResponse, error) {
	body, err := client.GetURL(ctx, requestURL)
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

	return removeBody(page)
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

func buildInitialSpacesURL(request ListSpacesRequest) string {
	query := url.Values{}
	query.Set("limit", strconv.Itoa(min(request.Limit, request.PageSize)))
	if request.Cursor != "" {
		query.Set("cursor", request.Cursor)
	}

	resolvedURL := url.URL{Path: spacesPath, RawQuery: query.Encode()}
	return resolvedURL.String()
}

func buildInitialPageCommentsURL(request ListPageCommentsRequest) string {
	query := buildCommentsQuery(request.BodyFormat, request.Raw, min(request.Limit, request.PageSize), request.Cursor)
	resolvedURL := url.URL{
		Path:     fmt.Sprintf(pageFooterCommentsPathTemplate, url.PathEscape(request.PageID)),
		RawQuery: query.Encode(),
	}

	return resolvedURL.String()
}

func buildCommentChildrenURL(commentID string, pageSize int, bodyFormat string, raw bool) string {
	query := buildCommentsQuery(bodyFormat, raw, pageSize, "")
	resolvedURL := url.URL{
		Path:     fmt.Sprintf(commentChildrenPathTemplate, url.PathEscape(commentID)),
		RawQuery: query.Encode(),
	}

	return resolvedURL.String()
}

func buildCommentsQuery(bodyFormat string, raw bool, limit int, cursor string) url.Values {
	query := url.Values{}
	effectiveBodyFormat := bodyFormat
	if raw && (effectiveBodyFormat == "" || effectiveBodyFormat == BodyFormatNone) {
		effectiveBodyFormat = BodyFormatView
	}

	if effectiveBodyFormat != "" && effectiveBodyFormat != BodyFormatNone {
		query.Set("body-format", effectiveBodyFormat)
	}

	query.Set("limit", strconv.Itoa(limit))

	if cursor != "" {
		query.Set("cursor", cursor)
	}

	return query
}

func getResultsPage(
	ctx context.Context,
	client *atlassian.Client,
	requestURL string,
	invalidResponseMessage string,
) (pageSearchResponse, error) {
	body, err := client.GetURL(ctx, requestURL)
	if err != nil {
		return pageSearchResponse{}, err
	}

	return decodeResultsPage(body, invalidResponseMessage)
}

func decodeResultsPage(body []byte, invalidResponseMessage string) (pageSearchResponse, error) {
	response := pageSearchResponse{}
	if err := json.Unmarshal(body, &response); err != nil {
		return pageSearchResponse{}, atlaserr.New(
			atlaserr.CodeUpstreamError,
			invalidResponseMessage,
			false,
			nil,
		)
	}

	return response, nil
}

func extractID(raw json.RawMessage) (string, error) {
	var payload struct {
		ID json.RawMessage `json:"id"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", err
	}

	if len(payload.ID) == 0 {
		return "", errors.New("missing id")
	}

	var asString string
	if err := json.Unmarshal(payload.ID, &asString); err == nil && asString != "" {
		return asString, nil
	}

	var asInt int64
	if err := json.Unmarshal(payload.ID, &asInt); err == nil {
		return strconv.FormatInt(asInt, 10), nil
	}

	var asFloat float64
	if err := json.Unmarshal(payload.ID, &asFloat); err == nil {
		return strconv.FormatFloat(asFloat, 'f', -1, 64), nil
	}

	return "", errors.New("unsupported id format")
}

func emitCommentTree(
	ctx context.Context,
	client *atlassian.Client,
	rootComment json.RawMessage,
	request ListPageCommentsRequest,
	remaining int,
	visitedCommentIDs map[string]struct{},
	emit func(item json.RawMessage) error,
) (int, error) {
	rootID, nextRemaining, shouldTraverse, emitErr := emitUniqueComment(rootComment, remaining, visitedCommentIDs, emit)
	if emitErr != nil {
		return remaining, emitErr
	}

	if !shouldTraverse {
		return nextRemaining, nil
	}

	remaining = nextRemaining

	stack := []string{rootID}

	for remaining > 0 && len(stack) > 0 {
		last := len(stack) - 1
		parentID := stack[last]
		stack = stack[:last]

		childIDs, childRemaining, childrenErr := emitChildrenForParent(
			ctx,
			client,
			parentID,
			request,
			remaining,
			visitedCommentIDs,
			emit,
		)
		if childrenErr != nil {
			return remaining, childrenErr
		}

		remaining = childRemaining
		stack = appendReversed(stack, childIDs)
	}

	return remaining, nil
}

func emitChildrenForParent(
	ctx context.Context,
	client *atlassian.Client,
	parentID string,
	request ListPageCommentsRequest,
	remaining int,
	visitedCommentIDs map[string]struct{},
	emit func(item json.RawMessage) error,
) ([]string, int, error) {
	nextURL := buildCommentChildrenURL(parentID, request.PageSize, request.BodyFormat, request.Raw)
	childIDs := make([]string, 0, request.PageSize)

	for remaining > 0 && nextURL != "" {
		response, pageErr := getResultsPage(
			ctx,
			client,
			nextURL,
			"invalid Confluence comment children response JSON",
		)
		if pageErr != nil {
			return nil, remaining, pageErr
		}

		for _, child := range response.Results {
			childID, nextRemaining, shouldTraverse, emitErr := emitUniqueComment(
				child,
				remaining,
				visitedCommentIDs,
				emit,
			)
			if emitErr != nil {
				return nil, remaining, emitErr
			}

			remaining = nextRemaining
			if shouldTraverse {
				childIDs = append(childIDs, childID)
			}

			if remaining == 0 {
				return childIDs, 0, nil
			}
		}

		nextURL = response.Links.Next
	}

	return childIDs, remaining, nil
}

func emitUniqueComment(
	comment json.RawMessage,
	remaining int,
	visitedCommentIDs map[string]struct{},
	emit func(item json.RawMessage) error,
) (string, int, bool, error) {
	commentID, commentIDErr := extractID(comment)
	if commentIDErr != nil {
		return "", remaining, false, atlaserr.New(
			atlaserr.CodeUpstreamError,
			"comment result missing valid id",
			false,
			nil,
		)
	}

	if _, seen := visitedCommentIDs[commentID]; seen {
		return commentID, remaining, false, nil
	}

	visitedCommentIDs[commentID] = struct{}{}
	if emitErr := emit(comment); emitErr != nil {
		return "", remaining, false, fmt.Errorf("emit comment: %w", emitErr)
	}

	nextRemaining := remaining - 1
	if nextRemaining <= 0 {
		return commentID, 0, false, nil
	}

	return commentID, nextRemaining, true, nil
}

func appendReversed(base []string, items []string) []string {
	for idx := len(items) - 1; idx >= 0; idx-- {
		base = append(base, items[idx])
	}

	return base
}

func buildQuery(options SearchOptions) url.Values {
	query := url.Values{}
	effectiveBodyFormat := options.BodyFormat
	if options.Raw && (effectiveBodyFormat == "" || effectiveBodyFormat == BodyFormatNone) {
		effectiveBodyFormat = BodyFormatView
	}

	if effectiveBodyFormat != "" && effectiveBodyFormat != BodyFormatNone {
		query.Set("body-format", effectiveBodyFormat)
	}

	if options.IncludeLabels || options.Raw {
		query.Set("include-labels", strconv.FormatBool(true))
	}

	if options.IncludeProperties || options.Raw {
		query.Set("include-properties", strconv.FormatBool(true))
	}

	if options.IncludeOperations || options.Raw {
		query.Set("include-operations", strconv.FormatBool(true))
	}

	if options.IncludeVersions || options.Raw {
		query.Set("include-versions", strconv.FormatBool(true))
	}

	return query
}

func removeBody(page json.RawMessage) (json.RawMessage, error) {
	fields := map[string]json.RawMessage{}
	if err := json.Unmarshal(page, &fields); err != nil {
		return nil, atlaserr.New(
			atlaserr.CodeUpstreamError,
			"invalid Confluence page JSON",
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
			false,
			nil,
		)
	}

	return cleaned, nil
}

func decodeSearchResponse(body []byte) (pageSearchResponse, error) {
	return decodeResultsPage(body, "invalid Confluence search response JSON")
}
