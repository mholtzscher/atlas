// Package confluence provides Confluence command tree.
package confluence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

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
	flagRaw               = "raw"
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
			newSpaceDescribeCommand(),
		},
	}
}

func newPageCommand() *ufcli.Command {
	return &ufcli.Command{
		Name:  "page",
		Usage: "Page operations",
		Commands: []*ufcli.Command{
			newPageDescribeCommand(),
			newPageViewCommand(),
			newPageSearchCommand(),
			newPageCommentsCommand(),
		},
	}
}

func newSpaceListCommand() *ufcli.Command {
	flags := paginationFlags("Max spaces to emit")
	flags = append(flags, &ufcli.BoolFlag{Name: flagRaw, Usage: "Emit full Confluence payload"})

	return &ufcli.Command{
		Name:  "list",
		Usage: "List accessible spaces",
		Flags: flags,
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
				compactSpace, compactErr := maybeCompactConfluenceRecord(cmd.Bool(flagRaw), space)
				if compactErr != nil {
					return compactErr
				}

				space = compactSpace
				return deps.Emitter.EmitRecord(ops.OpConfluenceSpaceList, space)
			})
		},
	}
}

func newSpaceDescribeCommand() *ufcli.Command {
	return &ufcli.Command{
		Name:      "describe",
		Usage:     "Describe space by key",
		ArgsUsage: "<SPACE_KEY>",
		Flags:     []ufcli.Flag{&ufcli.BoolFlag{Name: flagRaw, Usage: "Emit full Confluence payload"}},
		Action: func(ctx context.Context, cmd *ufcli.Command) error {
			deps, err := runtime.New(cmd, ops.OpConfluenceSpaceDescribe, true)
			if err != nil {
				return err
			}

			if cmd.Args().Len() != 1 {
				return errors.New("expected exactly one argument: <SPACE_KEY>")
			}

			space, describeErr := confluenceops.GetSpaceByKey(ctx, deps.Client, confluenceops.GetSpaceByKeyRequest{
				SpaceKey: cmd.Args().First(),
			})
			if describeErr != nil {
				return describeErr
			}

			compactSpace, compactErr := maybeCompactConfluenceRecord(cmd.Bool(flagRaw), space)
			if compactErr != nil {
				return compactErr
			}

			space = compactSpace

			return deps.Emitter.EmitRecord(ops.OpConfluenceSpaceDescribe, space)
		},
	}
}

func newPageDescribeCommand() *ufcli.Command {
	return &ufcli.Command{
		Name:      "describe",
		Usage:     "Describe page metadata by ID",
		ArgsUsage: "<PAGE_ID>",
		Flags:     pageDescribeFlags(),
		Action: func(ctx context.Context, cmd *ufcli.Command) error {
			deps, err := runtime.New(cmd, ops.OpConfluencePageDescribe, true)
			if err != nil {
				return err
			}

			if cmd.Args().Len() != 1 {
				return errors.New("expected exactly one argument: <PAGE_ID>")
			}

			page, getErr := confluenceops.GetPage(ctx, deps.Client, confluenceops.GetPageRequest{
				PageID:        cmd.Args().First(),
				SearchOptions: buildPageDescribeOptions(cmd),
			})
			if getErr != nil {
				return getErr
			}

			compactPage, compactErr := maybeCompactConfluenceRecord(false, page)
			if compactErr != nil {
				return compactErr
			}

			page = compactPage

			return deps.Emitter.EmitRecord(ops.OpConfluencePageDescribe, page)
		},
	}
}

func newPageViewCommand() *ufcli.Command {
	return &ufcli.Command{
		Name:      "view",
		Usage:     "Show page body content (formatted HTML)",
		ArgsUsage: "<PAGE_ID>",
		Action: func(ctx context.Context, cmd *ufcli.Command) error {
			deps, err := runtime.New(cmd, ops.OpConfluencePageView, true)
			if err != nil {
				return err
			}

			if cmd.Args().Len() != 1 {
				return errors.New("expected exactly one argument: <PAGE_ID>")
			}

			page, getErr := confluenceops.GetPage(ctx, deps.Client, confluenceops.GetPageRequest{
				PageID: cmd.Args().First(),
				SearchOptions: confluenceops.SearchOptions{
					BodyFormat: confluenceops.BodyFormatStorage,
				},
				Operation: ops.OpConfluencePageView,
			})
			if getErr != nil {
				return getErr
			}

			html, extractErr := confluenceops.ExtractPageViewHTML(page)
			if extractErr != nil {
				return extractErr
			}

			formatted := confluenceops.PrettyPrintHTML(html)
			if _, writeErr := fmt.Fprint(cmd.Writer, formatted); writeErr != nil {
				return fmt.Errorf("write page content: %w", writeErr)
			}

			return nil
		},
	}
}

func newPageSearchCommand() *ufcli.Command {
	flags := pageSearchFlags()
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
				compactPage, compactErr := maybeCompactConfluenceRecord(cmd.Bool(flagRaw), page)
				if compactErr != nil {
					return compactErr
				}

				page = compactPage
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
				Raw:        cmd.Bool(flagRaw),
			}, func(comment json.RawMessage) error {
				compactComment, compactErr := maybeCompactConfluenceRecord(cmd.Bool(flagRaw), comment)
				if compactErr != nil {
					return compactErr
				}

				comment = compactComment
				return deps.Emitter.EmitRecord(ops.OpConfluencePageComments, comment)
			})
		},
	}
}

func pageDescribeFlags() []ufcli.Flag {
	return []ufcli.Flag{
		&ufcli.BoolFlag{Name: flagIncludeLabels, Usage: "Include labels"},
		&ufcli.BoolFlag{Name: flagIncludeProperties, Usage: "Include properties"},
		&ufcli.BoolFlag{Name: flagIncludeOperations, Usage: "Include operations"},
		&ufcli.BoolFlag{Name: flagIncludeVersions, Usage: "Include versions"},
	}
}

func pageSearchFlags() []ufcli.Flag {
	return []ufcli.Flag{
		&ufcli.StringFlag{
			Name:  flagBodyFormat,
			Value: confluenceops.BodyFormatNone,
			Usage: "Body format (none, storage, editor, export_view, view, atlas_doc_format)",
		},
		&ufcli.BoolFlag{Name: flagIncludeLabels, Usage: "Include labels"},
		&ufcli.BoolFlag{Name: flagIncludeProperties, Usage: "Include properties"},
		&ufcli.BoolFlag{Name: flagIncludeOperations, Usage: "Include operations"},
		&ufcli.BoolFlag{Name: flagIncludeVersions, Usage: "Include versions"},
		&ufcli.BoolFlag{Name: flagRaw, Usage: "Emit full Confluence payload"},
	}
}

func commentsFlags() []ufcli.Flag {
	return []ufcli.Flag{
		&ufcli.StringFlag{
			Name:  flagBodyFormat,
			Value: confluenceops.BodyFormatNone,
			Usage: "Body format (none, storage, editor, export_view, view, atlas_doc_format)",
		},
		&ufcli.BoolFlag{Name: flagRaw, Usage: "Emit full Confluence payload"},
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
		Raw:               cmd.Bool(flagRaw),
	}
}

func buildPageDescribeOptions(cmd *ufcli.Command) confluenceops.SearchOptions {
	return confluenceops.SearchOptions{
		BodyFormat:        confluenceops.BodyFormatNone,
		IncludeLabels:     cmd.Bool(flagIncludeLabels),
		IncludeProperties: cmd.Bool(flagIncludeProperties),
		IncludeOperations: cmd.Bool(flagIncludeOperations),
		IncludeVersions:   cmd.Bool(flagIncludeVersions),
	}
}

func maybeCompactConfluenceRecord(raw bool, record json.RawMessage) (json.RawMessage, error) {
	if raw {
		return record, nil
	}

	compactRecord, compactErr := confluenceops.CompactRecord(record)
	if compactErr != nil {
		return nil, fmt.Errorf("compact confluence output: %w", compactErr)
	}

	return compactRecord, nil
}
