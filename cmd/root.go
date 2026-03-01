// Package cmd implements the CLI commands for atlas.
package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	altsrc "github.com/urfave/cli-altsrc/v3"
	altsrcjson "github.com/urfave/cli-altsrc/v3/json"
	ufcli "github.com/urfave/cli/v3"

	"github.com/mholtzscher/atlas/cmd/confluence"
	"github.com/mholtzscher/atlas/cmd/jira"
	"github.com/mholtzscher/atlas/internal/cli"
	"github.com/mholtzscher/atlas/internal/output"
)

const defaultTimeout = 30 * time.Second

// Version is set at build time.
//
//nolint:gochecknoglobals // version set at build time
var Version = "0.1.1" // x-release-please-version

// Run is the entry point for the CLI.
func Run(ctx context.Context, args []string) error {
	app := &ufcli.Command{
		Name:    "atlas",
		Usage:   "Agent first CLI for Atlassian products",
		Version: Version,
		Flags:   globalFlags(),
		Commands: []*ufcli.Command{
			jira.NewCommand(),
			confluence.NewCommand(),
		},
	}

	return app.Run(ctx, args)
}

func xdgConfigPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(configDir, "atlas", "atlas.json")
}

func configSources(flagName string, envKeys ...string) ufcli.ValueSourceChain {
	sources := ufcli.ValueSourceChain{}

	if len(envKeys) > 0 {
		sources.Append(ufcli.EnvVars(envKeys...))
	}

	// Local config takes precedence over XDG config
	localConfig := "./atlas.json"
	sources.Append(ufcli.NewValueSourceChain(
		altsrcjson.JSON("atlas."+flagName, altsrc.StringSourcer(localConfig)),
	))

	// XDG config as fallback
	xdgConfig := xdgConfigPath()
	if xdgConfig != "" {
		sources.Append(ufcli.NewValueSourceChain(
			altsrcjson.JSON("atlas."+flagName, altsrc.StringSourcer(xdgConfig)),
		))
	}

	return sources
}

func globalFlags() []ufcli.Flag {
	return []ufcli.Flag{
		&ufcli.StringFlag{
			Name:    cli.FlagOutput,
			Value:   string(output.FormatJSONL),
			Usage:   "Output format (jsonl or text)",
			Sources: configSources(cli.FlagOutput),
			Action: func(_ context.Context, _ *ufcli.Command, v string) error {
				if v != "jsonl" && v != "text" {
					return fmt.Errorf("invalid --%s: %q (must be 'jsonl' or 'text')", cli.FlagOutput, v)
				}
				return nil
			},
		},
		&ufcli.StringFlag{
			Name:    cli.FlagSite,
			Usage:   "Atlassian site URL",
			Sources: configSources(cli.FlagSite, cli.EnvSite),
		},
		&ufcli.StringFlag{
			Name:    cli.FlagAuth,
			Value:   cli.AuthPAT,
			Usage:   "Authentication mode (pat or oauth)",
			Sources: configSources(cli.FlagAuth),
			Action: func(_ context.Context, _ *ufcli.Command, v string) error {
				if v != cli.AuthPAT && v != cli.AuthOAuth {
					return fmt.Errorf("invalid --%s: %q (must be 'pat' or 'oauth')", cli.FlagAuth, v)
				}
				return nil
			},
		},
		&ufcli.StringFlag{
			Name:    cli.FlagEmail,
			Usage:   "PAT username/email",
			Sources: configSources(cli.FlagEmail, cli.EnvEmail),
		},
		&ufcli.StringFlag{
			Name:    cli.FlagAPIToken,
			Usage:   "PAT API token",
			Sources: configSources(cli.FlagAPIToken, cli.EnvAPIToken),
		},
		&ufcli.DurationFlag{
			Name:    cli.FlagTimeout,
			Usage:   "HTTP timeout",
			Value:   defaultTimeout,
			Sources: configSources(cli.FlagTimeout),
		},
		&ufcli.BoolFlag{
			Name:    cli.FlagVerbose,
			Usage:   "Print verbose output",
			Sources: configSources(cli.FlagVerbose),
		},
	}
}
