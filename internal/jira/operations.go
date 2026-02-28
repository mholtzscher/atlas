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
	"github.com/mholtzscher/atlas/internal/ops"
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

// GetIssueRequest defines Jira issue get inputs.
type GetIssueRequest struct {
	IssueKey     string
	Fields       []string
	Expand       []string
	FieldsByKeys bool
}

// SearchIssuesRequest defines Jira issue search inputs.
type SearchIssuesRequest struct {
	JQL          string
	Fields       []string
	Expand       []string
	FieldsByKeys bool
	Limit        int
	PageSize     int
	PageToken    string
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
		return nil, atlaserr.InvalidArgument("missing issue key", ops.OpJiraIssueGet)
	}

	query := buildIssueQuery(request.Fields, request.Expand, request.FieldsByKeys)
	body, err := client.Get(ctx, issueGetPathPrefix+url.PathEscape(request.IssueKey), query, ops.OpJiraIssueGet)
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
		query := buildIssueQuery(request.Fields, request.Expand, request.FieldsByKeys)
		query.Set("jql", request.JQL)
		query.Set("maxResults", strconv.Itoa(pageSize))
		if nextPageToken != "" {
			query.Set("nextPageToken", nextPageToken)
		}

		body, err := client.Get(ctx, issueSearchPath, query, ops.OpJiraIssueSearch)
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
		return atlaserr.InvalidArgument("missing required --jql", ops.OpJiraIssueSearch)
	}

	if request.Limit < 0 {
		return atlaserr.InvalidArgument("--limit must be >= 0", ops.OpJiraIssueSearch)
	}

	if request.PageSize <= 0 {
		return atlaserr.InvalidArgument("--page-size must be > 0", ops.OpJiraIssueSearch)
	}

	return nil
}

func emitIssues(
	issues []json.RawMessage,
	remaining int,
	emit func(item json.RawMessage) error,
) (int, error) {
	for _, issue := range issues {
		if emitErr := emit(issue); emitErr != nil {
			return remaining, fmt.Errorf("emit issue: %w", emitErr)
		}

		remaining--
		if remaining == 0 {
			return 0, nil
		}
	}

	return remaining, nil
}

func buildIssueQuery(fields []string, expand []string, fieldsByKeys bool) url.Values {
	query := url.Values{}

	query.Set("fieldsByKeys", strconv.FormatBool(fieldsByKeys))

	effectiveFields := fields
	if len(effectiveFields) == 0 {
		effectiveFields = DefaultFields()
	}

	query.Set("fields", strings.Join(effectiveFields, ","))

	if len(expand) > 0 {
		query.Set("expand", strings.Join(expand, ","))
	}

	return query
}

func decodeSearchResponse(body []byte) (issueSearchResponse, error) {
	response := issueSearchResponse{}
	if err := json.Unmarshal(body, &response); err != nil {
		return issueSearchResponse{}, atlaserr.New(
			atlaserr.CodeUpstreamError,
			"invalid Jira search response JSON",
			ops.OpJiraIssueSearch,
			false,
			nil,
		)
	}

	return response, nil
}

const (
	projectSearchPath = "/rest/api/3/project/search"
	issueCommentsPath = "/rest/api/3/issue/"
	issueTypesPath    = "/rest/api/3/issuetype"
	myselfPath        = "/rest/api/3/myself"
)

// ListProjectsRequest defines Jira project list inputs.
type ListProjectsRequest struct{}

// ListProjects fetches all projects accessible to the user.
func ListProjects(
	ctx context.Context,
	client *atlassian.Client,
	_ ListProjectsRequest,
	emit func(item json.RawMessage) error,
) error {
	query := url.Values{}
	query.Set("maxResults", "1000")

	body, err := client.Get(ctx, projectSearchPath, query, ops.OpJiraProjectList)
	if err != nil {
		return err
	}

	var response struct {
		Values []json.RawMessage `json:"values"`
	}
	if unmarshalErr := json.Unmarshal(body, &response); unmarshalErr != nil {
		return atlaserr.New(
			atlaserr.CodeUpstreamError,
			"invalid Jira project list response JSON",
			ops.OpJiraProjectList,
			false,
			nil,
		)
	}

	for _, project := range response.Values {
		if emitErr := emit(project); emitErr != nil {
			return fmt.Errorf("emit project: %w", emitErr)
		}
	}

	return nil
}

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
		return atlaserr.InvalidArgument("missing issue key", ops.OpJiraIssueComments)
	}

	path := issueCommentsPath + url.PathEscape(request.IssueKey) + "/comment"
	query := url.Values{}
	query.Set("maxResults", "1000")

	body, err := client.Get(ctx, path, query, ops.OpJiraIssueComments)
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
			ops.OpJiraIssueComments,
			false,
			nil,
		)
	}

	for _, comment := range response.Comments {
		if emitErr := emit(comment); emitErr != nil {
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
	body, err := client.Get(ctx, issueTypesPath, nil, ops.OpJiraIssueTypes)
	if err != nil {
		return err
	}

	var issueTypes []json.RawMessage
	if unmarshalErr := json.Unmarshal(body, &issueTypes); unmarshalErr != nil {
		return atlaserr.New(
			atlaserr.CodeUpstreamError,
			"invalid Jira issue types response JSON",
			ops.OpJiraIssueTypes,
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
	body, err := client.Get(ctx, myselfPath, nil, ops.OpJiraMyself)
	if err != nil {
		return nil, err
	}

	return json.RawMessage(body), nil
}
