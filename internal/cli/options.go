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
	// FlagNoColor disables color output.
	FlagNoColor = "no-color"
	// FlagToonIndent sets TOON indentation spaces.
	FlagToonIndent = "toon-indent"
	// FlagToonDelimiter sets TOON delimiter (comma, tab, pipe).
	FlagToonDelimiter = "toon-delimiter"
	// FlagToonLengthMarker adds # prefix to TOON array lengths.
	FlagToonLengthMarker = "toon-length-marker"
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
	Toon     ToonConfig
	Site     string
	Auth     string
	Email    string
	APIToken string
	Timeout  time.Duration
	Verbose  bool
	NoColor  bool
}

// ToonConfig holds gotoon encoding options.
type ToonConfig struct {
	Indent       int
	Delimiter    string
	LengthMarker bool
}

// GlobalOptionsFromCommand extracts global options from command.
func GlobalOptionsFromCommand(cmd *ufcli.Command) GlobalOptions {
	out, _ := output.ParseFormat(cmd.String(FlagOutput))

	// Parse delimiter string to actual delimiter character
	delimiter := ","
	switch cmd.String(FlagToonDelimiter) {
	case "tab":
		delimiter = "\t"
	case "pipe":
		delimiter = "|"
	}

	return GlobalOptions{
		Output: out,
		Toon: ToonConfig{
			Indent:       cmd.Int(FlagToonIndent),
			Delimiter:    delimiter,
			LengthMarker: cmd.Bool(FlagToonLengthMarker),
		},
		Site:     cmd.String(FlagSite),
		Auth:     cmd.String(FlagAuth),
		Email:    cmd.String(FlagEmail),
		APIToken: cmd.String(FlagAPIToken),
		Timeout:  cmd.Duration(FlagTimeout),
		Verbose:  cmd.Bool(FlagVerbose),
		NoColor:  cmd.Bool(FlagNoColor),
	}
}
