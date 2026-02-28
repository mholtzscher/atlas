// Package atlassian contains shared Atlassian Cloud HTTP/auth logic.
package atlassian

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
)

const authorizationHeader = "Authorization"

const (
	// AuthModePAT uses basic auth with email + token.
	AuthModePAT = "pat"
	// AuthModeOAuth is reserved for a future OAuth implementation.
	AuthModeOAuth = "oauth"
)

// Authenticator applies credentials to outgoing requests.
type Authenticator interface {
	Apply(request *http.Request) error
}

// PATAuthenticator applies Atlassian PAT basic auth.
type PATAuthenticator struct {
	email    string
	apiToken string
}

// OAuthAuthenticator is a v1 placeholder.
type OAuthAuthenticator struct{}

// NewPATAuthenticator creates a PAT authenticator.
func NewPATAuthenticator(email string, apiToken string) (*PATAuthenticator, error) {
	if email == "" {
		return nil, errors.New("missing PAT email")
	}

	if apiToken == "" {
		return nil, errors.New("missing PAT API token")
	}

	return &PATAuthenticator{email: email, apiToken: apiToken}, nil
}

// NewOAuthAuthenticator creates an OAuth authenticator placeholder.
func NewOAuthAuthenticator() *OAuthAuthenticator {
	return &OAuthAuthenticator{}
}

// Apply attaches PAT basic auth credentials.
func (a *PATAuthenticator) Apply(request *http.Request) error {
	credentials := fmt.Sprintf("%s:%s", a.email, a.apiToken)
	encoded := base64.StdEncoding.EncodeToString([]byte(credentials))
	request.Header.Set(authorizationHeader, fmt.Sprintf("Basic %s", encoded))

	return nil
}

// Apply returns a placeholder error for OAuth mode.
func (*OAuthAuthenticator) Apply(_ *http.Request) error {
	return errors.New("oauth auth mode is not implemented yet")
}
