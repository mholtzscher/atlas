// atlas is a CLI tool.
package main

import (
	"context"
	"errors"
	"os"

	"github.com/mholtzscher/atlas/cmd"
	"github.com/mholtzscher/atlas/internal/atlaserr"
	"github.com/mholtzscher/atlas/internal/output"
)

func main() {
	// Use default text format for pre-command error output.
	// Command-specific output format is determined by the --output flag.
	emitter := output.NewEmitter(output.FormatText, os.Stdout, os.Stderr)

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		atlasError := normalizeError(err)
		if emitErr := emitter.EmitError(atlasError); emitErr != nil {
			_, _ = os.Stderr.WriteString(emitErr.Error())
			_, _ = os.Stderr.WriteString("\n")
		}

		os.Exit(1)
	}
}

func normalizeError(err error) *atlaserr.Error {
	var structuredError *atlaserr.Error
	if errors.As(err, &structuredError) {
		return structuredError
	}

	return atlaserr.InvalidArgument(err.Error(), "")
}
