// Package jira provides Jira command tree.
package jira

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	ufcli "github.com/urfave/cli/v3"

	jiraops "github.com/mholtzscher/atlas/internal/jira"
	"github.com/mholtzscher/atlas/internal/runtime"
)

const (
	flagFields    = "fields"
	flagExpand    = "expand"
	flagQuery     = "query"
	flagLimit     = "limit"
	flagPageSize  = "page-size"
	flagPageToken = "page-token"
	flagRaw       = "raw"
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
			newProjectCommand(),
			newMyselfCommand(),
		},
	}
}

func newIssueCommand() *ufcli.Command {
	return &ufcli.Command{
		Name:  "issue",
		Usage: "Issue operations",
		Commands: []*ufcli.Command{
			newIssueDescribeCommand(),
			newIssueSearchCommand(),
			newIssueCommentsCommand(),
			newIssueTypesCommand(),
		},
	}
}

func newIssueCommentsCommand() *ufcli.Command {
	return &ufcli.Command{
		Name:      "comments",
		Usage:     "Get comments on an issue",
		ArgsUsage: "<ISSUE_KEY>",
		Action: func(ctx context.Context, cmd *ufcli.Command) error {
			deps, err := runtime.New(cmd, true)
			if err != nil {
				return err
			}

			if cmd.Args().Len() != 1 {
				return errors.New("expected exactly one argument: <ISSUE_KEY>")
			}

			return jiraops.GetIssueComments(ctx, deps.Client, jiraops.GetIssueCommentsRequest{
				IssueKey: cmd.Args().First(),
			}, func(comment json.RawMessage) error {
				return deps.Emitter.EmitRecord(comment)
			})
		},
	}
}

func newIssueDescribeCommand() *ufcli.Command {
	return &ufcli.Command{
		Name:      "describe",
		Usage:     "Describe issue by key",
		ArgsUsage: "<ISSUE_KEY>",
		Flags: []ufcli.Flag{
			&ufcli.StringSliceFlag{Name: flagFields, Usage: "Additional issue fields (added to compact defaults)"},
			&ufcli.StringSliceFlag{Name: flagExpand, Usage: "Expand fields"},
			&ufcli.BoolFlag{Name: flagRaw, Usage: "Emit full Jira issue payload"},
		},
		Action: func(ctx context.Context, cmd *ufcli.Command) error {
			deps, err := runtime.New(cmd, true)
			if err != nil {
				return err
			}

			if cmd.Args().Len() != 1 {
				return errors.New("expected exactly one argument: <ISSUE_KEY>")
			}

			issue, describeErr := jiraops.GetIssue(ctx, deps.Client, jiraops.GetIssueRequest{
				IssueKey: cmd.Args().First(),
				Fields:   cmd.StringSlice(flagFields),
				Expand:   cmd.StringSlice(flagExpand),
				Raw:      cmd.Bool(flagRaw),
			})
			if describeErr != nil {
				return describeErr
			}

			if !cmd.Bool(flagRaw) {
				compactIssue, compactErr := jiraops.CompactIssue(issue)
				if compactErr != nil {
					return fmt.Errorf("compact issue output: %w", compactErr)
				}

				issue = compactIssue
			}

			return deps.Emitter.EmitRecord(issue)
		},
	}
}

func newIssueSearchCommand() *ufcli.Command {
	return &ufcli.Command{
		Name:  "search",
		Usage: "Search issues with JQL",
		Flags: []ufcli.Flag{
			&ufcli.StringFlag{Name: flagQuery, Usage: "JQL query", Required: true},
			&ufcli.StringSliceFlag{Name: flagFields, Usage: "Additional issue fields (added to compact defaults)"},
			&ufcli.StringSliceFlag{Name: flagExpand, Usage: "Expand fields"},
			&ufcli.BoolFlag{Name: flagRaw, Usage: "Emit full Jira issue payload"},
			&ufcli.IntFlag{Name: flagLimit, Value: defaultLimit, Usage: "Max issues to emit"},
			&ufcli.IntFlag{Name: flagPageSize, Value: defaultPageSize, Usage: "Max results per request", Hidden: true},
			&ufcli.StringFlag{Name: flagPageToken, Usage: "Initial page token", Hidden: true},
		},
		Action: func(ctx context.Context, cmd *ufcli.Command) error {
			deps, err := runtime.New(cmd, true)
			if err != nil {
				return err
			}

			return jiraops.SearchIssues(ctx, deps.Client, jiraops.SearchIssuesRequest{
				JQL:       cmd.String(flagQuery),
				Fields:    cmd.StringSlice(flagFields),
				Expand:    cmd.StringSlice(flagExpand),
				Raw:       cmd.Bool(flagRaw),
				Limit:     cmd.Int(flagLimit),
				PageSize:  cmd.Int(flagPageSize),
				PageToken: cmd.String(flagPageToken),
			}, func(issue json.RawMessage) error {
				if !cmd.Bool(flagRaw) {
					compactIssue, compactErr := jiraops.CompactIssue(issue)
					if compactErr != nil {
						return fmt.Errorf("compact issue output: %w", compactErr)
					}

					issue = compactIssue
				}

				return deps.Emitter.EmitRecord(issue)
			})
		},
	}
}

func newProjectCommand() *ufcli.Command {
	return &ufcli.Command{
		Name:  "project",
		Usage: "Project operations",
		Commands: []*ufcli.Command{
			newProjectListCommand(),
		},
	}
}

func newProjectListCommand() *ufcli.Command {
	return &ufcli.Command{
		Name:  "list",
		Usage: "List all accessible projects",
		Action: func(ctx context.Context, cmd *ufcli.Command) error {
			deps, err := runtime.New(cmd, true)
			if err != nil {
				return err
			}

			return jiraops.ListProjects(
				ctx,
				deps.Client,
				jiraops.ListProjectsRequest{},
				func(project json.RawMessage) error {
					return deps.Emitter.EmitRecord(project)
				},
			)
		},
	}
}

func newMyselfCommand() *ufcli.Command {
	return &ufcli.Command{
		Name:  "myself",
		Usage: "Get current user information",
		Action: func(ctx context.Context, cmd *ufcli.Command) error {
			deps, err := runtime.New(cmd, true)
			if err != nil {
				return err
			}

			user, getErr := jiraops.GetMyself(ctx, deps.Client, jiraops.GetMyselfRequest{})
			if getErr != nil {
				return getErr
			}

			return deps.Emitter.EmitRecord(user)
		},
	}
}

func newIssueTypesCommand() *ufcli.Command {
	return &ufcli.Command{
		Name:  "types",
		Usage: "List all issue types",
		Action: func(ctx context.Context, cmd *ufcli.Command) error {
			deps, err := runtime.New(cmd, true)
			if err != nil {
				return err
			}

			return jiraops.ListIssueTypes(
				ctx,
				deps.Client,
				jiraops.ListIssueTypesRequest{},
				func(issueType json.RawMessage) error {
					return deps.Emitter.EmitRecord(issueType)
				},
			)
		},
	}
}
