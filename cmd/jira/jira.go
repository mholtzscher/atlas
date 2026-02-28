// Package jira provides Jira command tree.
package jira

import (
	"context"
	"encoding/json"
	"errors"

	ufcli "github.com/urfave/cli/v3"

	jiraops "github.com/mholtzscher/atlas/internal/jira"
	"github.com/mholtzscher/atlas/internal/ops"
	"github.com/mholtzscher/atlas/internal/runtime"
)

const (
	flagFields       = "fields"
	flagExpand       = "expand"
	flagFieldsByKeys = "fields-by-keys"
	flagJQL          = "jql"
	flagLimit        = "limit"
	flagPageSize     = "page-size"
	flagPageToken    = "page-token"
)

const (
	defaultLimit    = 50
	defaultPageSize = 50
)

// NewCommand creates the Jira command tree.
func NewCommand() *ufcli.Command {
	return &ufcli.Command{
		Name:  "jira",
		Usage: "Jira Cloud operations",
		Commands: []*ufcli.Command{
			newIssueCommand(),
		},
	}
}

func newIssueCommand() *ufcli.Command {
	return &ufcli.Command{
		Name:  "issue",
		Usage: "Issue operations",
		Commands: []*ufcli.Command{
			newIssueGetCommand(),
			newIssueSearchCommand(),
		},
	}
}

func newIssueGetCommand() *ufcli.Command {
	return &ufcli.Command{
		Name:      "get",
		Usage:     "Get issue by key",
		ArgsUsage: "<ISSUE_KEY>",
		Flags: []ufcli.Flag{
			&ufcli.StringSliceFlag{Name: flagFields, Value: jiraops.DefaultFields(), Usage: "Issue fields"},
			&ufcli.StringSliceFlag{Name: flagExpand, Usage: "Expand fields"},
			&ufcli.BoolFlag{Name: flagFieldsByKeys, Value: true, Usage: "Interpret fields by key"},
		},
		Action: func(ctx context.Context, cmd *ufcli.Command) error {
			deps, err := runtime.New(cmd, ops.OpJiraIssueGet, true)
			if err != nil {
				return err
			}

			if cmd.Args().Len() != 1 {
				return errors.New("expected exactly one argument: <ISSUE_KEY>")
			}

			issue, getErr := jiraops.GetIssue(ctx, deps.Client, jiraops.GetIssueRequest{
				IssueKey:     cmd.Args().First(),
				Fields:       cmd.StringSlice(flagFields),
				Expand:       cmd.StringSlice(flagExpand),
				FieldsByKeys: cmd.Bool(flagFieldsByKeys),
			})
			if getErr != nil {
				return getErr
			}

			return deps.Emitter.EmitRecord(ops.OpJiraIssueGet, issue)
		},
	}
}

func newIssueSearchCommand() *ufcli.Command {
	return &ufcli.Command{
		Name:  "search",
		Usage: "Search issues with JQL",
		Flags: []ufcli.Flag{
			&ufcli.StringFlag{Name: flagJQL, Usage: "JQL query", Required: true},
			&ufcli.StringSliceFlag{Name: flagFields, Value: jiraops.DefaultFields(), Usage: "Issue fields"},
			&ufcli.StringSliceFlag{Name: flagExpand, Usage: "Expand fields"},
			&ufcli.BoolFlag{Name: flagFieldsByKeys, Value: true, Usage: "Interpret fields by key"},
			&ufcli.IntFlag{Name: flagLimit, Value: defaultLimit, Usage: "Max issues to emit"},
			&ufcli.IntFlag{Name: flagPageSize, Value: defaultPageSize, Usage: "Max results per request"},
			&ufcli.StringFlag{Name: flagPageToken, Usage: "Initial page token"},
		},
		Action: func(ctx context.Context, cmd *ufcli.Command) error {
			deps, err := runtime.New(cmd, ops.OpJiraIssueSearch, true)
			if err != nil {
				return err
			}

			return jiraops.SearchIssues(ctx, deps.Client, jiraops.SearchIssuesRequest{
				JQL:          cmd.String(flagJQL),
				Fields:       cmd.StringSlice(flagFields),
				Expand:       cmd.StringSlice(flagExpand),
				FieldsByKeys: cmd.Bool(flagFieldsByKeys),
				Limit:        cmd.Int(flagLimit),
				PageSize:     cmd.Int(flagPageSize),
				PageToken:    cmd.String(flagPageToken),
			}, func(issue json.RawMessage) error {
				return deps.Emitter.EmitRecord(ops.OpJiraIssueSearch, issue)
			})
		},
	}
}
