// Package jira implements Jira read operations.
package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/mholtzscher/atlas/internal/atlaserr"
	"github.com/mholtzscher/atlas/internal/atlassian"
)

const (
	issueGetPathPrefix = "/rest/api/3/issue/"
	issueSearchPath    = "/rest/api/3/search/jql"
)

const (
	defaultFieldSummary   = "summary"
	defaultFieldStatus    = "status"
	defaultFieldIssueType = "issuetype"
	defaultFieldPriority  = "priority"
	defaultFieldAssignee  = "assignee"
	defaultFieldReporter  = "reporter"
	defaultFieldProject   = "project"
	defaultFieldCreated   = "created"
	defaultFieldUpdated   = "updated"
)

type issueSearchResponse struct {
	Issues        []json.RawMessage `json:"issues"`
	NextPageToken string            `json:"nextPageToken"`
}

type projectSearchResponse struct {
	Values []json.RawMessage `json:"values"`
}

// GetIssueRequest defines Jira issue get inputs.
type GetIssueRequest struct {
	IssueKey string
	Fields   []string
	Expand   []string
	Raw      bool
}

// SearchIssuesRequest defines Jira issue search inputs.
type SearchIssuesRequest struct {
	JQL       string
	Fields    []string
	Expand    []string
	Raw       bool
	Limit     int
	PageSize  int
	PageToken string
}

// DefaultFields returns token-efficient default field selection.
func DefaultFields() []string {
	return []string{
		defaultFieldSummary,
		defaultFieldStatus,
		defaultFieldIssueType,
		defaultFieldPriority,
		defaultFieldAssignee,
		defaultFieldReporter,
		defaultFieldProject,
		defaultFieldCreated,
		defaultFieldUpdated,
	}
}

// GetIssue fetches one issue.
func GetIssue(
	ctx context.Context,
	client *atlassian.Client,
	request GetIssueRequest,
) (json.RawMessage, error) {
	if request.IssueKey == "" {
		return nil, atlaserr.InvalidArgument("missing issue key")
	}

	query := buildIssueQuery(request.Fields, request.Expand, request.Raw)
	body, err := client.Get(ctx, issueGetPathPrefix+url.PathEscape(request.IssueKey), query)
	if err != nil {
		return nil, err
	}

	return json.RawMessage(body), nil
}

// SearchIssues streams issues from JQL search.
func SearchIssues(
	ctx context.Context,
	client *atlassian.Client,
	request SearchIssuesRequest,
	emit func(item json.RawMessage) error,
) error {
	if validateErr := validateSearchRequest(request); validateErr != nil {
		return validateErr
	}

	if request.Limit <= 0 {
		return nil
	}

	remaining := request.Limit
	nextPageToken := request.PageToken

	for remaining > 0 {
		pageSize := min(remaining, request.PageSize)
		query := buildIssueQuery(request.Fields, request.Expand, request.Raw)
		query.Set("jql", request.JQL)
		query.Set("maxResults", strconv.Itoa(pageSize))
		if nextPageToken != "" {
			query.Set("nextPageToken", nextPageToken)
		}

		body, err := client.Get(ctx, issueSearchPath, query)
		if err != nil {
			return err
		}

		response, decodeErr := decodeSearchResponse(body)
		if decodeErr != nil {
			return decodeErr
		}

		var emitErr error
		remaining, emitErr = emitIssues(response.Issues, remaining, emit)
		if emitErr != nil {
			return emitErr
		}

		if remaining == 0 {
			return nil
		}

		if response.NextPageToken == "" {
			return nil
		}

		nextPageToken = response.NextPageToken
	}

	return nil
}

func validateSearchRequest(request SearchIssuesRequest) error {
	if request.JQL == "" {
		return atlaserr.InvalidArgument("missing required --jql")
	}

	return nil
}

func emitIssues(issues []json.RawMessage, remaining int, emit func(item json.RawMessage) error) (int, error) {
	for _, issue := range issues {
		if remaining <= 0 {
			return 0, nil
		}

		if emitErr := emit(issue); emitErr != nil {
			return remaining, fmt.Errorf("emit issue: %w", emitErr)
		}

		remaining--
	}

	return remaining, nil
}

func buildIssueQuery(fields []string, expand []string, raw bool) url.Values {
	query := url.Values{}
	query.Set("fieldsByKeys", "true")

	effectiveFields := effectiveIssueFields(fields, raw)

	if len(effectiveFields) > 0 {
		query.Set("fields", strings.Join(effectiveFields, ","))
	}

	if len(expand) > 0 {
		query.Set("expand", strings.Join(expand, ","))
	}

	return query
}

func effectiveIssueFields(fields []string, raw bool) []string {
	if raw {
		return []string{"*all"}
	}

	effectiveFields := append([]string{}, DefaultFields()...)
	seen := make(map[string]struct{}, len(effectiveFields))
	for _, field := range effectiveFields {
		seen[field] = struct{}{}
	}

	for _, field := range fields {
		trimmedField := strings.TrimSpace(field)
		if trimmedField == "" {
			continue
		}

		if _, exists := seen[trimmedField]; exists {
			continue
		}

		effectiveFields = append(effectiveFields, trimmedField)
		seen[trimmedField] = struct{}{}
	}

	return effectiveFields
}

func decodeSearchResponse(body []byte) (issueSearchResponse, error) {
	var response issueSearchResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return issueSearchResponse{}, atlaserr.New(
			atlaserr.CodeUpstreamError,
			"invalid Jira issue search response JSON",
			false,
			nil,
		)
	}

	return response, nil
}

const (
	issueTypesPath    = "/rest/api/3/issuetype"
	issueCommentsPath = "/rest/api/3/issue/"
	myselfPath        = "/rest/api/3/myself"
	projectsPath      = "/rest/api/3/project"
	projectSearchPath = "/rest/api/3/project/search"
)

// GetIssueCommentsRequest defines Jira issue comments inputs.
type GetIssueCommentsRequest struct {
	IssueKey string
}

// GetIssueComments fetches comments for an issue.
func GetIssueComments(
	ctx context.Context,
	client *atlassian.Client,
	request GetIssueCommentsRequest,
	emit func(item json.RawMessage) error,
) error {
	if request.IssueKey == "" {
		return atlaserr.InvalidArgument("missing issue key")
	}

	path := issueCommentsPath + url.PathEscape(request.IssueKey) + "/comment"
	query := url.Values{}
	query.Set("maxResults", "1000")

	body, err := client.Get(ctx, path, query)
	if err != nil {
		return err
	}

	var response struct {
		Comments []json.RawMessage `json:"comments"`
	}
	if unmarshalErr := json.Unmarshal(body, &response); unmarshalErr != nil {
		return atlaserr.New(
			atlaserr.CodeUpstreamError,
			"invalid Jira issue comments response JSON",
			false,
			nil,
		)
	}

	for _, comment := range response.Comments {
		// Simplify comment body before emitting
		simplifiedComment, simplifyErr := simplifyCommentBody(comment)
		if simplifyErr != nil {
			// If simplification fails, use original comment
			simplifiedComment = comment
		}

		if emitErr := emit(simplifiedComment); emitErr != nil {
			return fmt.Errorf("emit comment: %w", emitErr)
		}
	}

	return nil
}

// ListIssueTypesRequest defines Jira issue types list inputs.
type ListIssueTypesRequest struct{}

// ListIssueTypes fetches all issue types.
func ListIssueTypes(
	ctx context.Context,
	client *atlassian.Client,
	_ ListIssueTypesRequest,
	emit func(item json.RawMessage) error,
) error {
	body, err := client.Get(ctx, issueTypesPath, nil)
	if err != nil {
		return err
	}

	var issueTypes []json.RawMessage
	if unmarshalErr := json.Unmarshal(body, &issueTypes); unmarshalErr != nil {
		return atlaserr.New(
			atlaserr.CodeUpstreamError,
			"invalid Jira issue types response JSON",
			false,
			nil,
		)
	}

	for _, issueType := range issueTypes {
		if emitErr := emit(issueType); emitErr != nil {
			return fmt.Errorf("emit issue type: %w", emitErr)
		}
	}

	return nil
}

// GetMyselfRequest defines Jira myself inputs.
type GetMyselfRequest struct{}

// GetMyself fetches the current user's information.
func GetMyself(
	ctx context.Context,
	client *atlassian.Client,
	_ GetMyselfRequest,
) (json.RawMessage, error) {
	body, err := client.Get(ctx, myselfPath, nil)
	if err != nil {
		return nil, err
	}

	return json.RawMessage(body), nil
}

// simplifyCommentBody extracts the body content from ADF structure and returns plain text.
// Input: {"body":{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"Hello"}]}]},...}
// Output: {"body":"Hello",...}.
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

	// Extract plain text from ADF structure.
	plainText := extractTextFromADF(bodyObj)
	if plainText != "" {
		data["body"] = plainText
		return json.Marshal(data)
	}

	return comment, nil
}

// extractTextFromADF extracts plain text from Atlassian Document Format structure.
func extractTextFromADF(adf map[string]any) string {
	// Check if it's an ADF document.
	if docType, hasType := adf["type"]; !hasType || docType != "doc" {
		return ""
	}

	content, hasContent := adf["content"]
	if !hasContent {
		return ""
	}

	contentArray, isArray := content.([]any)
	if !isArray {
		return ""
	}

	return extractTextFromContentNodes(contentArray)
}

// extractTextFromContentNodes extracts text from ADF content nodes.
func extractTextFromContentNodes(nodes []any) string {
	var texts []string

	for _, node := range nodes {
		nodeObj, isMap := node.(map[string]any)
		if !isMap {
			continue
		}

		texts = append(texts, extractTextFromNode(nodeObj)...)
	}

	return strings.Join(texts, " ")
}

// extractTextFromNode extracts text from a single ADF node.
func extractTextFromNode(node map[string]any) []string {
	var texts []string

	nodeContent, hasNodeContent := node["content"]
	if !hasNodeContent {
		return texts
	}

	nodeContentArray, isNodeArray := nodeContent.([]any)
	if !isNodeArray {
		return texts
	}

	for _, child := range nodeContentArray {
		childObj, isChildMap := child.(map[string]any)
		if !isChildMap {
			continue
		}

		if text := extractTextFromTextNode(childObj); text != "" {
			texts = append(texts, text)
		}
	}

	return texts
}

// extractTextFromTextNode extracts text from a text-type ADF node.
func extractTextFromTextNode(node map[string]any) string {
	if childType, hasChildType := node["type"]; !hasChildType || childType != "text" {
		return ""
	}

	text, hasText := node["text"]
	if !hasText {
		return ""
	}

	textStr, isString := text.(string)
	if !isString || textStr == "" {
		return ""
	}

	return textStr
}

// ListProjectsRequest defines Jira projects list inputs.
type ListProjectsRequest struct {
	Query    string
	Limit    int
	PageSize int
	Expand   []string
}

// ListProjects fetches all projects the user has access to.
func ListProjects(
	ctx context.Context,
	client *atlassian.Client,
	request ListProjectsRequest,
	emit func(item json.RawMessage) error,
) error {
	if request.Limit <= 0 {
		return nil
	}

	remaining := request.Limit
	isSearch := request.Query != ""
	path := projectsPath
	if isSearch {
		path = projectSearchPath
	}

	for remaining > 0 {
		pageSize := min(remaining, request.PageSize)
		query := buildProjectsQuery(request, pageSize)

		body, err := client.Get(ctx, path, query)
		if err != nil {
			return err
		}

		projects, decodeErr := decodeProjectsResponse(body, isSearch)
		if decodeErr != nil {
			return decodeErr
		}

		var emitErr error
		remaining, emitErr = emitProjects(projects, remaining, emit)
		if emitErr != nil {
			return emitErr
		}

		if remaining == 0 || len(projects) < pageSize {
			return nil
		}
	}

	return nil
}

func buildProjectsQuery(request ListProjectsRequest, pageSize int) url.Values {
	query := url.Values{}
	if request.Query != "" {
		query.Set("query", request.Query)
	}

	query.Set("maxResults", strconv.Itoa(pageSize))

	if len(request.Expand) > 0 {
		query.Set("expand", strings.Join(request.Expand, ","))
	}

	return query
}

func decodeProjectsResponse(body []byte, isSearch bool) ([]json.RawMessage, error) {
	if isSearch {
		var response projectSearchResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, atlaserr.New(
				atlaserr.CodeUpstreamError,
				"invalid Jira project search response JSON",
				false,
				nil,
			)
		}

		return response.Values, nil
	}

	var projects []json.RawMessage
	if err := json.Unmarshal(body, &projects); err != nil {
		return nil, atlaserr.New(
			atlaserr.CodeUpstreamError,
			"invalid Jira projects response JSON",
			false,
			nil,
		)
	}

	return projects, nil
}

func emitProjects(projects []json.RawMessage, remaining int, emit func(item json.RawMessage) error) (int, error) {
	for _, project := range projects {
		if remaining <= 0 {
			return 0, nil
		}

		if err := emit(project); err != nil {
			return remaining, fmt.Errorf("emit project: %w", err)
		}

		remaining--
	}

	return remaining, nil
}
