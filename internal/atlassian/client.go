package atlassian

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mholtzscher/atlas/internal/atlaserr"
)

const (
	acceptHeader       = "Accept"
	acceptJSON         = "application/json"
	userAgentHeader    = "User-Agent"
	defaultClientAgent = "atlas/dev"
)

// ClientConfig configures a shared Atlassian HTTP client.
type ClientConfig struct {
	SiteURL       string
	Timeout       time.Duration
	Authenticator Authenticator
	HTTPClient    *http.Client
	Verbose       bool
	ErrWriter     io.Writer
	UserAgent     string
}

// Client wraps a site-scoped HTTP client.
type Client struct {
	baseURL       *url.URL
	httpClient    *http.Client
	authenticator Authenticator
	verbose       bool
	errWriter     io.Writer
	userAgent     string
}

// NewClient creates a configured Atlassian client.
func NewClient(config ClientConfig) (*Client, error) {
	parsedURL, err := url.Parse(config.SiteURL)
	if err != nil {
		return nil, fmt.Errorf("parse site URL: %w", err)
	}

	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, errors.New("site URL must be absolute")
	}

	if config.Authenticator == nil {
		return nil, errors.New("missing authenticator")
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{}
	}

	if config.Timeout > 0 {
		httpClient.Timeout = config.Timeout
	}

	errWriter := config.ErrWriter
	if errWriter == nil {
		errWriter = io.Discard
	}

	userAgent := config.UserAgent
	if userAgent == "" {
		userAgent = defaultClientAgent
	}

	return &Client{
		baseURL:       parsedURL,
		httpClient:    httpClient,
		authenticator: config.Authenticator,
		verbose:       config.Verbose,
		errWriter:     errWriter,
		userAgent:     userAgent,
	}, nil
}

// Get sends a GET request to a site-relative path and query.
func (c *Client) Get(
	ctx context.Context,
	path string,
	query url.Values,
	op string,
) ([]byte, error) {
	resolvedURL, err := c.resolveURL(path)
	if err != nil {
		return nil, atlaserr.InvalidArgument(err.Error(), op)
	}

	resolvedURL.RawQuery = query.Encode()
	return c.getResolvedURL(ctx, resolvedURL, op)
}

// GetURL sends a GET request using an absolute or site-relative URL.
func (c *Client) GetURL(ctx context.Context, requestURL string, op string) ([]byte, error) {
	resolvedURL, err := c.resolveURL(requestURL)
	if err != nil {
		return nil, atlaserr.InvalidArgument(err.Error(), op)
	}

	return c.getResolvedURL(ctx, resolvedURL, op)
}

func (c *Client) getResolvedURL(ctx context.Context, resolvedURL *url.URL, op string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, resolvedURL.String(), nil)
	if err != nil {
		return nil, atlaserr.InvalidArgument(fmt.Sprintf("build request: %v", err), op)
	}

	request.Header.Set(acceptHeader, acceptJSON)
	request.Header.Set(userAgentHeader, c.userAgent)

	if authErr := c.authenticator.Apply(request); authErr != nil {
		return nil, atlaserr.InvalidArgument(authErr.Error(), op)
	}

	if c.verbose {
		_, _ = fmt.Fprintf(c.errWriter, "HTTP GET %s\n", resolvedURL.Redacted())
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, atlaserr.Network(op, err)
	}

	defer func() {
		_ = response.Body.Close()
	}()

	body, readErr := io.ReadAll(response.Body)
	if readErr != nil {
		return nil, atlaserr.Network(op, readErr)
	}

	if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices {
		return body, nil
	}

	return nil, atlaserr.FromHTTPStatus(op, response.StatusCode, response.Header)
}

func (c *Client) resolveURL(pathOrURL string) (*url.URL, error) {
	parsedURL, err := url.Parse(pathOrURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL/path: %w", err)
	}

	if parsedURL.IsAbs() {
		return parsedURL, nil
	}

	cleanBase := *c.baseURL
	cleanBase.Path = strings.TrimSuffix(cleanBase.Path, "/") + "/"
	return cleanBase.ResolveReference(parsedURL), nil
}
