package jira_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mholtzscher/atlas/internal/atlassian"
	jiraops "github.com/mholtzscher/atlas/internal/jira"
)

func TestGetIssueUsesAdditiveFieldsQuery(t *testing.T) {
	t.Parallel()

	defaultFields := strings.Join(jiraops.DefaultFields(), ",")

	testCases := []struct {
		name   string
		fields []string
		raw    bool
		want   string
	}{
		{
			name:   "no user fields uses defaults only",
			fields: nil,
			want:   defaultFields,
		},
		{
			name:   "user adds new and existing default",
			fields: []string{"labels", "status"},
			want:   defaultFields + ",labels",
		},
		{
			name:   "repeated flags are deduped",
			fields: []string{"labels", "labels", "components", "components"},
			want:   defaultFields + ",labels,components",
		},
		{
			name:   "special jira field is allowed",
			fields: []string{"*all"},
			want:   defaultFields + ",*all",
		},
		{
			name:   "whitespace and empty values are dropped",
			fields: []string{"  ", " labels ", "", "\tcomponents\t"},
			want:   defaultFields + ",labels,components",
		},
		{
			name: "raw requests all fields",
			raw:  true,
			want: "*all",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if request.URL.Path != "/rest/api/3/issue/ATLAS-1" {
					t.Fatalf("unexpected path: %s", request.URL.Path)
				}

				if got := request.URL.Query().Get("fieldsByKeys"); got != "true" {
					t.Fatalf("expected fieldsByKeys=true, got %q", got)
				}

				if got := request.URL.Query().Get("fields"); got != testCase.want {
					t.Fatalf("unexpected fields value: got %q want %q", got, testCase.want)
				}

				writeJSON(writer, `{"id":"10000"}`)
			}))
			defer server.Close()

			client := newTestClient(t, server.URL)

			_, err := jiraops.GetIssue(context.Background(), client, jiraops.GetIssueRequest{
				IssueKey: "ATLAS-1",
				Fields:   testCase.fields,
				Raw:      testCase.raw,
			})
			if err != nil {
				t.Fatalf("GetIssue returned error: %v", err)
			}
		})
	}
}

func TestSearchIssuesRawRequestsAllFields(t *testing.T) {
	t.Parallel()

	emitted := 0

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/rest/api/3/search/jql" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}

		if got := request.URL.Query().Get("fields"); got != "*all" {
			t.Fatalf("unexpected fields value: %q", got)
		}

		writeJSON(writer, `{"issues":[{"id":"10001"}],"nextPageToken":""}`)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)

	err := jiraops.SearchIssues(context.Background(), client, jiraops.SearchIssuesRequest{
		JQL:      "project = ATLAS",
		Raw:      true,
		Limit:    1,
		PageSize: 1,
	}, func(_ json.RawMessage) error {
		emitted++
		return nil
	})
	if err != nil {
		t.Fatalf("SearchIssues returned error: %v", err)
	}

	if emitted != 1 {
		t.Fatalf("expected 1 emitted issue, got %d", emitted)
	}
}

func TestSearchIssuesSetsFieldsByKeysTrue(t *testing.T) {
	t.Parallel()

	defaultFields := strings.Join(jiraops.DefaultFields(), ",")
	emitted := 0

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/rest/api/3/search/jql" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}

		if got := request.URL.Query().Get("fieldsByKeys"); got != "true" {
			t.Fatalf("expected fieldsByKeys=true, got %q", got)
		}

		if got := request.URL.Query().Get("fields"); got != defaultFields+",labels" {
			t.Fatalf("unexpected fields value: %q", got)
		}

		writeJSON(writer, `{"issues":[{"id":"10001"}],"nextPageToken":""}`)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)

	err := jiraops.SearchIssues(context.Background(), client, jiraops.SearchIssuesRequest{
		JQL:      "project = ATLAS",
		Fields:   []string{"labels"},
		Limit:    1,
		PageSize: 1,
	}, func(_ json.RawMessage) error {
		emitted++
		return nil
	})
	if err != nil {
		t.Fatalf("SearchIssues returned error: %v", err)
	}

	if emitted != 1 {
		t.Fatalf("expected 1 emitted issue, got %d", emitted)
	}
}

func newTestClient(t *testing.T, siteURL string) *atlassian.Client {
	t.Helper()

	authenticator, authErr := atlassian.NewPATAuthenticator("test@example.com", "token")
	if authErr != nil {
		t.Fatalf("build authenticator: %v", authErr)
	}

	client, clientErr := atlassian.NewClient(atlassian.ClientConfig{
		SiteURL:       siteURL,
		Authenticator: authenticator,
	})
	if clientErr != nil {
		t.Fatalf("build client: %v", clientErr)
	}

	return client
}

func writeJSON(writer http.ResponseWriter, body string) {
	writer.Header().Set("Content-Type", "application/json")
	_, _ = writer.Write([]byte(body))
}
