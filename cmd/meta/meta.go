// Package meta provides machine-readable metadata commands.
package meta

import (
	"context"
	"encoding/json"
	"fmt"

	ufcli "github.com/urfave/cli/v3"

	"github.com/mholtzscher/atlas/internal/ops"
	"github.com/mholtzscher/atlas/internal/output"
	"github.com/mholtzscher/atlas/internal/runtime"
)

// NewCommand creates the `meta` command tree.
func NewCommand() *ufcli.Command {
	return &ufcli.Command{
		Name:  "meta",
		Usage: "Metadata for harnesses",
		Commands: []*ufcli.Command{
			newOpsCommand(),
		},
	}
}

func newOpsCommand() *ufcli.Command {
	return &ufcli.Command{
		Name:  "ops",
		Usage: "List operation registry",
		Action: func(_ context.Context, cmd *ufcli.Command) error {
			deps, err := runtime.New(cmd, "meta.ops", false)
			if err != nil {
				return err
			}

			for _, definition := range ops.All() {
				line, marshalErr := json.Marshal(definition)
				if marshalErr != nil {
					return marshalErr
				}

				if deps.Options.Output == output.FormatText {
					if _, writeErr := fmt.Fprintln(cmd.Writer, definition.Op); writeErr != nil {
						return writeErr
					}
					continue
				}

				if _, writeErr := fmt.Fprintln(cmd.Writer, string(line)); writeErr != nil {
					return writeErr
				}
			}

			return nil
		},
	}
}
