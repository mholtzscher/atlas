// Package output renders command data and errors.
package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/mholtzscher/atlas/internal/atlaserr"
)

const (
	// FormatJSONL prints newline-delimited JSON records.
	FormatJSONL Format = "jsonl"
	// FormatText prints plain text output.
	FormatText Format = "text"
)

// Format controls output encoding.
type Format string

type successRecord struct {
	Op   string          `json:"op"`
	Data json.RawMessage `json:"data"`
}

// Emitter writes success and error output.
type Emitter struct {
	format Format
	stdout io.Writer
	stderr io.Writer
}

// NewEmitter creates an output emitter for a format.
func NewEmitter(format Format, stdout io.Writer, stderr io.Writer) Emitter {
	return Emitter{
		format: format,
		stdout: stdout,
		stderr: stderr,
	}
}

// ParseFormat validates output format input.
func ParseFormat(raw string) (Format, error) {
	if raw == "" {
		return FormatJSONL, nil
	}

	format := Format(raw)
	switch format {
	case FormatJSONL, FormatText:
		return format, nil
	default:
		return "", fmt.Errorf("invalid --output: %q", raw)
	}
}

// EmitRecord writes one operation record.
func (e Emitter) EmitRecord(op string, data json.RawMessage) error {
	if e.format == FormatText {
		_, err := fmt.Fprintf(e.stdout, "%s\t%s\n", op, string(data))
		return err
	}

	recordBytes, err := json.Marshal(successRecord{Op: op, Data: data})
	if err != nil {
		return fmt.Errorf("marshal success record: %w", err)
	}

	_, err = fmt.Fprintln(e.stdout, string(recordBytes))
	return err
}

// EmitError writes one error object.
func (e Emitter) EmitError(err *atlaserr.Error) error {
	if e.format == FormatText {
		if err.Op == "" {
			_, writeErr := fmt.Fprintf(e.stderr, "%s: %s\n", err.Code, err.Message)
			return writeErr
		}

		_, writeErr := fmt.Fprintf(e.stderr, "%s (%s): %s\n", err.Code, err.Op, err.Message)
		return writeErr
	}

	errorBytes, marshalErr := json.Marshal(err.Envelope())
	if marshalErr != nil {
		return fmt.Errorf("marshal error record: %w", marshalErr)
	}

	_, writeErr := fmt.Fprintln(e.stderr, string(errorBytes))
	return writeErr
}

// Format returns emitter output format.
func (e Emitter) Format() Format {
	return e.format
}
