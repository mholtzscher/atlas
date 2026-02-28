// Package confluence provides Confluence command tree.
package confluence

import (
	"context"
	"encoding/json"
	"errors"

	ufcli "github.com/urfave/cli/v3"

	confluenceops "github.com/mholtzscher/atlas/internal/confluence"
	"github.com/mholtzscher/atlas/internal/ops"
	"github.com/mholtzscher/atlas/internal/runtime"
)

const (
	flagBodyFormat        = "body-format"
	flagIncludeLabels     = "include-labels"
	flagIncludeProperties = "include-properties"
	flagIncludeOperations = "include-operations"
	flagIncludeVersions   = "include-versions"
	flagCQL               = "cql"
	flagLimit             = "limit"
	flagPageSize          = "page-size"
	flagCursor            = "cursor"
)

const (
	defaultLimit    = 25
	defaultPageSize = 25
)

// NewCommand creates the Confluence command tree.
func NewCommand() *ufcli.Command {
	return &ufcli.Command{
		Name:  "confluence",
		Usage: "Confluence Cloud operations",
		Commands: []*ufcli.Command{
			newSpaceCommand(),
			newPageCommand(),
		},
	}
}

func newSpaceCommand() *ufcli.Command {
	return &ufcli.Command{
		Name:  "space",
		Usage: "Space operations",
		Commands: []*ufcli.Command{
			newSpaceListCommand(),
			newSpaceGetCommand(),
		},
	}
}

func newPageCommand() *ufcli.Command {
	return &ufcli.Command{
		Name:  "page",
		Usage: "Page operations",
		Commands: []*ufcli.Command{
			newPageGetCommand(),
			newPageSearchCommand(),
			newPageCommentsCommand(),
		},
	}
}

func newSpaceListCommand() *ufcli.Command {
	return &ufcli.Command{
		Name:  "list",
		Usage: "List accessible spaces",
		Flags: paginationFlags("Max spaces to emit"),
		Action: func(ctx context.Context, cmd *ufcli.Command) error {
			deps, err := runtime.New(cmd, ops.OpConfluenceSpaceList, true)
			if err != nil {
				return err
			}

			return confluenceops.ListSpaces(ctx, deps.Client, confluenceops.ListSpacesRequest{
				Limit:    cmd.Int(flagLimit),
				PageSize: cmd.Int(flagPageSize),
				Cursor:   cmd.String(flagCursor),
			}, func(space json.RawMessage) error {
				return deps.Emitter.EmitRecord(ops.OpConfluenceSpaceList, space)
			})
		},
	}
}

func newSpaceGetCommand() *ufcli.Command {
	return &ufcli.Command{
		Name:      "get",
		Usage:     "Get space by key",
		ArgsUsage: "<SPACE_KEY>",
		Action: func(ctx context.Context, cmd *ufcli.Command) error {
			deps, err := runtime.New(cmd, ops.OpConfluenceSpaceGet, true)
			if err != nil {
				return err
			}

			if cmd.Args().Len() != 1 {
				return errors.New("expected exactly one argument: <SPACE_KEY>")
			}

			space, getErr := confluenceops.GetSpaceByKey(ctx, deps.Client, confluenceops.GetSpaceByKeyRequest{
				SpaceKey: cmd.Args().First(),
			})
			if getErr != nil {
				return getErr
			}

			return deps.Emitter.EmitRecord(ops.OpConfluenceSpaceGet, space)
		},
	}
}

func newPageGetCommand() *ufcli.Command {
	return &ufcli.Command{
		Name:      "get",
		Usage:     "Get page by ID",
		ArgsUsage: "<PAGE_ID>",
		Flags:     pageFlags(),
		Action: func(ctx context.Context, cmd *ufcli.Command) error {
			deps, err := runtime.New(cmd, ops.OpConfluencePageGet, true)
			if err != nil {
				return err
			}

			if cmd.Args().Len() != 1 {
				return errors.New("expected exactly one argument: <PAGE_ID>")
			}

			page, getErr := confluenceops.GetPage(ctx, deps.Client, confluenceops.GetPageRequest{
				PageID:        cmd.Args().First(),
				SearchOptions: buildSearchOptions(cmd),
			})
			if getErr != nil {
				return getErr
			}

			return deps.Emitter.EmitRecord(ops.OpConfluencePageGet, page)
		},
	}
}

func newPageSearchCommand() *ufcli.Command {
	flags := pageFlags()
	flags = append(flags, &ufcli.StringFlag{Name: flagCQL, Usage: "CQL query", Required: true})
	flags = append(flags, paginationFlags("Max pages to emit")...)

	return &ufcli.Command{
		Name:  "search",
		Usage: "Search pages with CQL",
		Flags: flags,
		Action: func(ctx context.Context, cmd *ufcli.Command) error {
			deps, err := runtime.New(cmd, ops.OpConfluencePageSearch, true)
			if err != nil {
				return err
			}

			return confluenceops.SearchPages(ctx, deps.Client, confluenceops.SearchPagesRequest{
				CQL:           cmd.String(flagCQL),
				Limit:         cmd.Int(flagLimit),
				PageSize:      cmd.Int(flagPageSize),
				Cursor:        cmd.String(flagCursor),
				SearchOptions: buildSearchOptions(cmd),
			}, func(page json.RawMessage) error {
				return deps.Emitter.EmitRecord(ops.OpConfluencePageSearch, page)
			})
		},
	}
}

func newPageCommentsCommand() *ufcli.Command {
	flags := commentsFlags()
	flags = append(flags, paginationFlags("Max comments to emit")...)

	return &ufcli.Command{
		Name:      "comments",
		Usage:     "Get footer comments on a page",
		ArgsUsage: "<PAGE_ID>",
		Flags:     flags,
		Action: func(ctx context.Context, cmd *ufcli.Command) error {
			deps, err := runtime.New(cmd, ops.OpConfluencePageComments, true)
			if err != nil {
				return err
			}

			if cmd.Args().Len() != 1 {
				return errors.New("expected exactly one argument: <PAGE_ID>")
			}

			return confluenceops.ListPageComments(ctx, deps.Client, confluenceops.ListPageCommentsRequest{
				PageID:     cmd.Args().First(),
				Limit:      cmd.Int(flagLimit),
				PageSize:   cmd.Int(flagPageSize),
				Cursor:     cmd.String(flagCursor),
				BodyFormat: cmd.String(flagBodyFormat),
			}, func(comment json.RawMessage) error {
				return deps.Emitter.EmitRecord(ops.OpConfluencePageComments, comment)
			})
		},
	}
}

func pageFlags() []ufcli.Flag {
	return []ufcli.Flag{
		&ufcli.StringFlag{
			Name:  flagBodyFormat,
			Value: confluenceops.BodyFormatView,
			Usage: "Body format (none, storage, editor, export_view, view, atlas_doc_format)",
		},
		&ufcli.BoolFlag{Name: flagIncludeLabels, Usage: "Include labels"},
		&ufcli.BoolFlag{Name: flagIncludeProperties, Usage: "Include properties"},
		&ufcli.BoolFlag{Name: flagIncludeOperations, Usage: "Include operations"},
		&ufcli.BoolFlag{Name: flagIncludeVersions, Usage: "Include versions"},
	}
}

func commentsFlags() []ufcli.Flag {
	return []ufcli.Flag{
		&ufcli.StringFlag{
			Name:  flagBodyFormat,
			Value: confluenceops.BodyFormatView,
			Usage: "Body format (none, storage, editor, export_view, view, atlas_doc_format)",
		},
	}
}

func paginationFlags(limitUsage string) []ufcli.Flag {
	return []ufcli.Flag{
		&ufcli.IntFlag{Name: flagLimit, Value: defaultLimit, Usage: limitUsage},
		&ufcli.IntFlag{Name: flagPageSize, Value: defaultPageSize, Usage: "Max results per request"},
		&ufcli.StringFlag{Name: flagCursor, Usage: "Initial cursor"},
	}
}

func buildSearchOptions(cmd *ufcli.Command) confluenceops.SearchOptions {
	return confluenceops.SearchOptions{
		BodyFormat:        cmd.String(flagBodyFormat),
		IncludeLabels:     cmd.Bool(flagIncludeLabels),
		IncludeProperties: cmd.Bool(flagIncludeProperties),
		IncludeOperations: cmd.Bool(flagIncludeOperations),
		IncludeVersions:   cmd.Bool(flagIncludeVersions),
	}
}
