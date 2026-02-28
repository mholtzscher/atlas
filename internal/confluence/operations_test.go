package confluence_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/mholtzscher/atlas/internal/atlaserr"
	"github.com/mholtzscher/atlas/internal/atlassian"
	confluenceops "github.com/mholtzscher/atlas/internal/confluence"
	"github.com/mholtzscher/atlas/internal/ops"
)

func TestListSpacesFollowsPagination(t *testing.T) {
	t.Parallel()

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestCount++

		if request.URL.Path != "/wiki/api/v2/spaces" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}

		switch request.URL.Query().Get("cursor") {
		case "":
			if request.URL.Query().Get("limit") != "1" {
				t.Fatalf("expected initial limit=1, got %q", request.URL.Query().Get("limit"))
			}

			writeJSON(writer, `{"results":[{"id":"100"}],"_links":{"next":"/wiki/api/v2/spaces?cursor=next&limit=1"}}`)
		case "next":
			writeJSON(writer, `{"results":[{"id":"200"}],"_links":{"next":""}}`)
		default:
			t.Fatalf("unexpected cursor: %q", request.URL.Query().Get("cursor"))
		}
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	ids := make([]string, 0, 2)

	err := confluenceops.ListSpaces(context.Background(), client, confluenceops.ListSpacesRequest{
		Limit:    2,
		PageSize: 1,
	}, func(item json.RawMessage) error {
		var payload struct {
			ID string `json:"id"`
		}

		idErr := json.Unmarshal(item, &payload)
		if idErr != nil {
			return idErr
		}

		ids = append(ids, payload.ID)
		return nil
	})
	if err != nil {
		t.Fatalf("ListSpaces returned error: %v", err)
	}

	if requestCount != 2 {
		t.Fatalf("expected 2 requests, got %d", requestCount)
	}

	if !reflect.DeepEqual(ids, []string{"100", "200"}) {
		t.Fatalf("unexpected ids: %#v", ids)
	}
}

func TestGetSpaceByKeyNotFound(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/wiki/api/v2/spaces" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}

		if request.URL.Query().Get("keys") != "DEV" {
			t.Fatalf("expected keys=DEV, got %q", request.URL.Query().Get("keys"))
		}

		writeJSON(writer, `{"results":[],"_links":{"next":""}}`)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)

	_, err := confluenceops.GetSpaceByKey(
		context.Background(),
		client,
		confluenceops.GetSpaceByKeyRequest{SpaceKey: "DEV"},
	)
	if err == nil {
		t.Fatal("expected error")
	}

	var atlasError *atlaserr.Error
	if !errors.As(err, &atlasError) {
		t.Fatalf("expected *atlaserr.Error, got %T", err)
	}

	if atlasError.Code != atlaserr.CodeNotFound {
		t.Fatalf("expected code %s, got %s", atlaserr.CodeNotFound, atlasError.Code)
	}

	if atlasError.Op != ops.OpConfluenceSpaceGet {
		t.Fatalf("expected op %s, got %s", ops.OpConfluenceSpaceGet, atlasError.Op)
	}
}

func TestListPageCommentsTraversesThreadsAndHonorsLimit(t *testing.T) {
	t.Parallel()

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestCount++

		switch request.URL.Path {
		case "/wiki/api/v2/pages/123/footer-comments":
			if request.URL.Query().Get("limit") != "2" {
				t.Fatalf("expected limit=2, got %q", request.URL.Query().Get("limit"))
			}

			writeJSON(writer, `{"results":[{"id":"c1"},{"id":"c2"}],"_links":{"next":""}}`)
		case "/wiki/api/v2/footer-comments/c1/children":
			switch request.URL.Query().Get("cursor") {
			case "":
				writeJSON(
					writer,
					`{"results":[{"id":"c1a"}],`+
						`"_links":{"next":"/wiki/api/v2/footer-comments/c1/children?cursor=next&limit=2"}}`,
				)
			case "next":
				writeJSON(writer, `{"results":[{"id":"c1b"}],"_links":{"next":""}}`)
			default:
				t.Fatalf("unexpected cursor: %q", request.URL.Query().Get("cursor"))
			}
		default:
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	emittedIDs := make([]string, 0, 3)

	err := confluenceops.ListPageComments(context.Background(), client, confluenceops.ListPageCommentsRequest{
		PageID:     "123",
		Limit:      3,
		PageSize:   2,
		BodyFormat: confluenceops.BodyFormatView,
	}, func(item json.RawMessage) error {
		var payload struct {
			ID string `json:"id"`
		}

		idErr := json.Unmarshal(item, &payload)
		if idErr != nil {
			return idErr
		}

		emittedIDs = append(emittedIDs, payload.ID)
		return nil
	})
	if err != nil {
		t.Fatalf("ListPageComments returned error: %v", err)
	}

	if !reflect.DeepEqual(emittedIDs, []string{"c1", "c1a", "c1b"}) {
		t.Fatalf("unexpected emitted ids: %#v", emittedIDs)
	}

	if requestCount != 3 {
		t.Fatalf("expected 3 requests, got %d", requestCount)
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
