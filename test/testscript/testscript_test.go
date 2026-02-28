package testscript //nolint:testpackage // testscript tests must use same package

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/mholtzscher/atlas/cmd"
	"github.com/mholtzscher/atlas/internal/atlaserr"
	"github.com/mholtzscher/atlas/internal/output"

	"github.com/rogpeppe/go-internal/testscript"
)

func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){
		"atlas": func() {
			runAtlas(os.Args)
		},
	})
}

func TestScript(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "scripts",
	})
}

func runAtlas(args []string) {
	emitter := output.NewEmitter(output.FormatText, output.ToonOptions{}, os.Stdout, os.Stderr)

	if err := cmd.Run(context.Background(), args); err != nil {
		_ = emitter.EmitError(normalizeError(err))
		os.Exit(1)
	}

	os.Exit(0)
}
func normalizeError(err error) *atlaserr.Error {
	var structuredError *atlaserr.Error
	if errors.As(err, &structuredError) {
		return structuredError
	}

	return atlaserr.InvalidArgument(err.Error())
}
