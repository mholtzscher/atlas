// Package confluence implements Confluence read operations.
package confluence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"

	"github.com/mholtzscher/atlas/internal/atlaserr"
	"github.com/mholtzscher/atlas/internal/atlassian"
	"github.com/mholtzscher/atlas/internal/output"
)

const (
	pageGetPathPrefix              = "/wiki/api/v2/pages/"
	pageSearchPath                 = "/wiki/rest/api/content/search"
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
	// defaultCommentsPageSize is the page size for fetching all comments.
	defaultCommentsPageSize = 100
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
) (*output.Pagination, error) {
	if validateErr := validateSearchRequest(request); validateErr != nil {
		return nil, validateErr
	}

	if request.Limit <= 0 {
		return nil, nil //nolint:nilnil // nil pagination means no more data
	}

	remaining := request.Limit
	nextURL := buildInitialSearchURL(request)
	stripBody := !request.Raw
	returnedCount := 0
	var lastNextURL string

	for remaining > 0 && nextURL != "" {
		response, pageErr := getSearchPage(ctx, client, nextURL)
		if pageErr != nil {
			return nil, pageErr
		}

		var emitErr error
		remaining, returnedCount, emitErr = emitSearchResults(
			response.Results,
			stripBody,
			remaining,
			returnedCount,
			emit,
		)
		if emitErr != nil {
			return nil, emitErr
		}

		if remaining == 0 {
			// Reached limit - check if there are more results
			if response.Links.Next != "" {
				return &output.Pagination{
					HasMore:    true,
					NextCursor: extractCursorFromURL(response.Links.Next),
					Returned:   returnedCount,
				}, nil
			}
			// No more results available
			return nil, nil //nolint:nilnil // nil pagination means no more data
		}

		lastNextURL = response.Links.Next
		nextURL = response.Links.Next
	}

	if lastNextURL != "" {
		// There might be more results available
		return &output.Pagination{
			HasMore:    true,
			NextCursor: extractCursorFromURL(lastNextURL),
			Returned:   returnedCount,
		}, nil
	}

	return nil, nil //nolint:nilnil // nil pagination means no more data
}

// ListSpaces streams accessible spaces.
func ListSpaces(
	ctx context.Context,
	client *atlassian.Client,
	request ListSpacesRequest,
	emit func(item json.RawMessage) error,
) (*output.Pagination, error) {
	if request.Limit < 0 {
		return nil, atlaserr.InvalidArgument("--limit must be >= 0")
	}

	if request.PageSize <= 0 {
		return nil, atlaserr.InvalidArgument("--page-size must be > 0")
	}

	if request.Limit == 0 {
		return nil, nil //nolint:nilnil // nil pagination means no more data
	}

	remaining := request.Limit
	nextURL := buildInitialSpacesURL(request)
	returnedCount := 0
	var lastNextURL string

	for remaining > 0 && nextURL != "" {
		response, pageErr := getResultsPage(
			ctx,
			client,
			nextURL,
			"invalid Confluence spaces response JSON",
		)
		if pageErr != nil {
			return nil, pageErr
		}

		for _, space := range response.Results {
			if emitErr := emit(space); emitErr != nil {
				return nil, fmt.Errorf("emit space: %w", emitErr)
			}

			remaining--
			returnedCount++
			if remaining == 0 {
				// Reached limit - check if there are more results
				if response.Links.Next != "" {
					return &output.Pagination{
						HasMore:    true,
						NextCursor: extractCursorFromURL(response.Links.Next),
						Returned:   returnedCount,
					}, nil
				}
				// No more results available
				return nil, nil //nolint:nilnil // nil pagination means no more data
			}
		}

		lastNextURL = response.Links.Next
		nextURL = response.Links.Next
	}

	if lastNextURL != "" {
		// There might be more results available
		return &output.Pagination{
			HasMore:    true,
			NextCursor: extractCursorFromURL(lastNextURL),
			Returned:   returnedCount,
		}, nil
	}

	return nil, nil //nolint:nilnil // nil pagination means no more data
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

// ListPageComments streams all page footer comments and their threaded replies.
// This function returns all comments without pagination support.
func ListPageComments(
	ctx context.Context,
	client *atlassian.Client,
	request ListPageCommentsRequest,
	emit func(item json.RawMessage) error,
) error {
	if request.PageID == "" {
		return atlaserr.InvalidArgument("missing page ID")
	}

	nextURL := buildInitialPageCommentsURL(request)
	visitedCommentIDs := map[string]struct{}{}

	for nextURL != "" {
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
			if processErr := emitCommentTree(
				ctx,
				client,
				comment,
				request,
				visitedCommentIDs,
				emit,
			); processErr != nil {
				return processErr
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

func getSearchPage(
	ctx context.Context,
	client *atlassian.Client,
	requestURL string,
) (pageSearchResponse, error) {
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
	returnedCount int,
	emit func(item json.RawMessage) error,
) (int, int, error) {
	for _, page := range pages {
		cleanPage, cleanErr := maybeStripBody(page, stripBody)
		if cleanErr != nil {
			return remaining, returnedCount, cleanErr
		}

		if emitErr := emit(cleanPage); emitErr != nil {
			return remaining, returnedCount, fmt.Errorf("emit page: %w", emitErr)
		}

		remaining--
		returnedCount++
		if remaining == 0 {
			return 0, returnedCount, nil
		}
	}

	return remaining, returnedCount, nil
}

func maybeStripBody(page json.RawMessage, stripBody bool) (json.RawMessage, error) {
	if !stripBody {
		return page, nil
	}

	return removeBody(page)
}

func buildInitialSearchURL(request SearchPagesRequest) string {
	query := buildSearchQuery(request.SearchOptions)
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
	query := buildCommentsQuery(request.BodyFormat, request.Raw, defaultCommentsPageSize, "")
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
	visitedCommentIDs map[string]struct{},
	emit func(item json.RawMessage) error,
) error {
	rootID, shouldTraverse, emitErr := emitUniqueComment(
		rootComment,
		visitedCommentIDs,
		emit,
	)
	if emitErr != nil {
		return emitErr
	}

	if !shouldTraverse {
		return nil
	}

	stack := []string{rootID}

	for len(stack) > 0 {
		last := len(stack) - 1
		parentID := stack[last]
		stack = stack[:last]

		childIDs, childrenErr := emitChildrenForParent(
			ctx,
			client,
			parentID,
			request,
			visitedCommentIDs,
			emit,
		)
		if childrenErr != nil {
			return childrenErr
		}

		stack = appendReversed(stack, childIDs)
	}

	return nil
}

func emitChildrenForParent(
	ctx context.Context,
	client *atlassian.Client,
	parentID string,
	request ListPageCommentsRequest,
	visitedCommentIDs map[string]struct{},
	emit func(item json.RawMessage) error,
) ([]string, error) {
	nextURL := buildCommentChildrenURL(
		parentID,
		defaultCommentsPageSize,
		request.BodyFormat,
		request.Raw,
	)
	childIDs := make([]string, 0, defaultCommentsPageSize)

	for nextURL != "" {
		response, pageErr := getResultsPage(
			ctx,
			client,
			nextURL,
			"invalid Confluence comment children response JSON",
		)
		if pageErr != nil {
			return nil, pageErr
		}

		for _, child := range response.Results {
			childID, shouldTraverse, emitErr := emitUniqueComment(
				child,
				visitedCommentIDs,
				emit,
			)
			if emitErr != nil {
				return nil, emitErr
			}

			if shouldTraverse {
				childIDs = append(childIDs, childID)
			}
		}

		nextURL = response.Links.Next
	}

	return childIDs, nil
}

func emitUniqueComment(
	comment json.RawMessage,
	visitedCommentIDs map[string]struct{},
	emit func(item json.RawMessage) error,
) (string, bool, error) {
	commentID, commentIDErr := extractID(comment)
	if commentIDErr != nil {
		return "", false, atlaserr.New(
			atlaserr.CodeUpstreamError,
			"comment result missing valid id",
			false,
			nil,
		)
	}

	if _, seen := visitedCommentIDs[commentID]; seen {
		return commentID, false, nil
	}

	// Simplify comment body before emitting
	simplifiedComment, simplifyErr := simplifyCommentBody(comment)
	if simplifyErr != nil {
		// If simplification fails, use original comment
		simplifiedComment = comment
	}

	visitedCommentIDs[commentID] = struct{}{}
	if emitErr := emit(simplifiedComment); emitErr != nil {
		return "", false, fmt.Errorf("emit comment: %w", emitErr)
	}

	return commentID, true, nil
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

func buildSearchQuery(options SearchOptions) url.Values {
	query := url.Values{}

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

// simplifyCommentBody extracts the body content from nested structure and converts HTML to Markdown.
// Input: {"body":{"storage":{"representation":"storage","value":"<p>text</p>"}},...}
// Output: {"body":"text",...}.
func simplifyCommentBody(comment json.RawMessage) (json.RawMessage, error) {
	var data map[string]any
	if err := json.Unmarshal(comment, &data); err != nil {
		return nil, fmt.Errorf("parse comment: %w", err)
	}

	body, hasBody := data["body"]
	if !hasBody {
		return comment, nil
	}

	bodyObj, isMap := body.(map[string]any)
	if !isMap {
		return comment, nil
	}

	// Try to extract content from storage format.
	plainText := extractPlainTextFromStorage(bodyObj)
	if plainText == "" {
		// Try to extract from atlas_doc_format.
		plainText = extractPlainTextFromADF(bodyObj)
	}

	if plainText != "" {
		data["body"] = plainText
		return json.Marshal(data)
	}

	return comment, nil
}

// extractPlainTextFromStorage extracts plain text from storage format body.
func extractPlainTextFromStorage(bodyObj map[string]any) string {
	storage, hasStorage := bodyObj["storage"]
	if !hasStorage {
		return ""
	}

	storageObj, isMap := storage.(map[string]any)
	if !isMap {
		return ""
	}

	value, hasValue := storageObj["value"]
	if !hasValue {
		return ""
	}

	strValue, isString := value.(string)
	if !isString || strValue == "" {
		return ""
	}

	return stripHTMLTags(strValue)
}

// extractPlainTextFromADF extracts plain text from ADF format body.
func extractPlainTextFromADF(bodyObj map[string]any) string {
	adf, hasAdf := bodyObj["atlas_doc_format"]
	if !hasAdf {
		return ""
	}

	adfObj, isMap := adf.(map[string]any)
	if !isMap {
		return ""
	}

	value, hasValue := adfObj["value"]
	if !hasValue {
		return ""
	}

	strValue, isString := value.(string)
	if !isString || strValue == "" {
		return ""
	}

	return extractTextFromADFJSON(strValue)
}

// stripHTMLTags removes HTML tags and decodes entities.
func stripHTMLTags(html string) string {
	markdown, err := htmltomarkdown.ConvertString(html)
	if err != nil {
		return simpleStripTags(html)
	}
	return cleanMarkdown(markdown)
}

// simpleStripTags is a fallback HTML tag stripper.
func simpleStripTags(html string) string {
	inTag := false
	var result []rune
	for _, r := range html {
		switch r {
		case '<':
			inTag = true
		case '>':
			inTag = false
		default:
			if !inTag {
				result = append(result, r)
			}
		}
	}
	return string(result)
}

// cleanMarkdown removes markdown formatting for plain text output.
func cleanMarkdown(md string) string {
	replacements := []struct {
		old string
		new string
	}{
		{"**", ""},
		{"__", ""},
		{"*", ""},
		{"_", ""},
		{"`", ""},
		{"## ", ""},
		{"# ", ""},
		{"> ", ""},
		{"- ", ""},
		{"* ", ""},
	}
	result := md
	for _, r := range replacements {
		result = strings.ReplaceAll(result, r.old, r.new)
	}
	return strings.TrimSpace(result)
}

// extractTextFromADFJSON extracts plain text from Atlas Document Format JSON.
func extractTextFromADFJSON(adfJSON string) string {
	var adf struct {
		Content []struct {
			Type    string `json:"type"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"content"`
	}
	if err := json.Unmarshal([]byte(adfJSON), &adf); err != nil {
		return adfJSON
	}

	var texts []string
	for _, node := range adf.Content {
		for _, child := range node.Content {
			if child.Text != "" {
				texts = append(texts, child.Text)
			}
		}
	}

	return strings.Join(texts, " ")
}

// extractCursorFromURL extracts the cursor query parameter from a URL string.
func extractCursorFromURL(urlStr string) string {
	if urlStr == "" {
		return ""
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return ""
	}

	return parsedURL.Query().Get("cursor")
}
