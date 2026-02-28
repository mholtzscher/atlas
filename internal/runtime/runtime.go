// Package runtime builds shared command dependencies.
package runtime

import (
	"fmt"

	ufcli "github.com/urfave/cli/v3"

	"github.com/mholtzscher/atlas/internal/atlaserr"
	"github.com/mholtzscher/atlas/internal/atlassian"
	"github.com/mholtzscher/atlas/internal/cli"
	"github.com/mholtzscher/atlas/internal/output"
)

// Dependencies contains command runtime dependencies.
type Dependencies struct {
	Options cli.GlobalOptions
	Emitter output.Emitter
	Client  *atlassian.Client
}

// New creates runtime dependencies for a command operation.
func New(cmd *ufcli.Command, needsNetwork bool) (Dependencies, error) {
	options := cli.GlobalOptionsFromCommand(cmd)

	emitter := output.NewEmitter(
		options.Output,
		output.ToonOptions{
			Indent:       options.Toon.Indent,
			Delimiter:    options.Toon.Delimiter,
			LengthMarker: options.Toon.LengthMarker,
		},
		cmd.Writer,
		cmd.ErrWriter,
	)
	deps := Dependencies{
		Options: options,
		Emitter: emitter,
		Client:  nil,
	}

	if !needsNetwork {
		return deps, nil
	}

	if options.Site == "" {
		return Dependencies{}, atlaserr.InvalidArgument(
			fmt.Sprintf("missing required --%s or %s", cli.FlagSite, cli.EnvSite),
		)
	}

	authenticator, authErr := buildAuthenticator(options)
	if authErr != nil {
		return Dependencies{}, atlaserr.InvalidArgument(authErr.Error())
	}

	client, clientErr := atlassian.NewClient(atlassian.ClientConfig{
		SiteURL:       options.Site,
		Timeout:       options.Timeout,
		Authenticator: authenticator,
		Verbose:       options.Verbose,
		ErrWriter:     cmd.ErrWriter,
		UserAgent:     fmt.Sprintf("atlas/%s", cmd.Root().Version),
	})
	if clientErr != nil {
		return Dependencies{}, atlaserr.InvalidArgument(clientErr.Error())
	}

	deps.Client = client
	return deps, nil
}

func buildAuthenticator(options cli.GlobalOptions) (atlassian.Authenticator, error) {
	switch options.Auth {
	case atlassian.AuthModePAT:
		return atlassian.NewPATAuthenticator(options.Email, options.APIToken)
	case atlassian.AuthModeOAuth:
		return atlassian.NewOAuthAuthenticator(), nil
	default:
		return nil, fmt.Errorf("unsupported auth mode: %s", options.Auth)
	}
}
