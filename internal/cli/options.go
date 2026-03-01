// Package cli provides shared CLI options and utilities.
package cli

import (
	"time"

	ufcli "github.com/urfave/cli/v3"

	"github.com/mholtzscher/atlas/internal/output"
)

const (
	// FlagOutput controls output formatting.
	FlagOutput = "output"
	// FlagSite is the Atlassian site URL flag.
	FlagSite = "site"
	// FlagAuth selects authentication mode.
	FlagAuth = "auth"
	// FlagEmail is the PAT username flag.
	FlagEmail = "email"
	// FlagAPIToken is the PAT token flag.
	FlagAPIToken = "api-token"
	// FlagTimeout controls outbound request timeout.
	FlagTimeout = "timeout"
	// FlagVerbose enables diagnostic logs.
	FlagVerbose = "verbose"
)

const (
	// EnvSite stores the Atlassian base site URL.
	EnvSite = "ATLAS_SITE"
	// EnvEmail stores the PAT username.
	EnvEmail = "ATLAS_EMAIL"
	// EnvAPIToken stores the PAT token.
	//nolint:gosec // env var name only, not a credential literal.
	EnvAPIToken = "ATLAS_API_TOKEN"
)

const (
	// AuthPAT uses email + API token basic auth.
	AuthPAT = "pat"
	// AuthOAuth is reserved for a future OAuth flow.
	AuthOAuth = "oauth"
)

// GlobalOptions holds CLI flags shared across commands.
type GlobalOptions struct {
	Output   output.Format
	Site     string
	Auth     string
	Email    string
	APIToken string
	Timeout  time.Duration
	Verbose  bool
}

// GlobalOptionsFromCommand extracts global options from command.
func GlobalOptionsFromCommand(cmd *ufcli.Command) GlobalOptions {
	out, _ := output.ParseFormat(cmd.String(FlagOutput))

	return GlobalOptions{
		Output:   out,
		Site:     cmd.String(FlagSite),
		Auth:     cmd.String(FlagAuth),
		Email:    cmd.String(FlagEmail),
		APIToken: cmd.String(FlagAPIToken),
		Timeout:  cmd.Duration(FlagTimeout),
		Verbose:  cmd.Bool(FlagVerbose),
	}
}
