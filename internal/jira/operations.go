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
