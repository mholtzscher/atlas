// Package output renders command data and errors.
package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/alpkeskin/gotoon"

	"github.com/mholtzscher/atlas/internal/atlaserr"
)

const (
	// FormatJSONL prints newline-delimited JSON records.
	FormatJSONL Format = "jsonl"
	// FormatText prints plain text output.
	FormatText Format = "text"
	// FormatToon prints TOON (Token-Oriented Object Notation) output.
	FormatToon Format = "toon"

	// DefaultToonIndent is the default indentation for TOON output.
	DefaultToonIndent = 2
	// DefaultToonDelimiter is the default delimiter for TOON output.
	DefaultToonDelimiter = ","
)

// Format controls output encoding.
type Format string

// ToonOptions holds gotoon encoding configuration.
type ToonOptions struct {
	Indent       int
	Delimiter    string
	LengthMarker bool
}

type successRecord struct {
	Data json.RawMessage `json:"data"`
}

// Emitter writes success and error output.
type Emitter struct {
	format   Format
	toonOpts ToonOptions
	stdout   io.Writer
	stderr   io.Writer
}

// NewEmitter creates an output emitter for a format.
func NewEmitter(format Format, toonOpts ToonOptions, stdout io.Writer, stderr io.Writer) Emitter {
	return Emitter{
		format:   format,
		toonOpts: toonOpts,
		stdout:   stdout,
		stderr:   stderr,
	}
}

// ParseFormat validates output format input.
func ParseFormat(raw string) (Format, error) {
	if raw == "" {
		return FormatJSONL, nil
	}

	format := Format(raw)
	switch format {
	case FormatJSONL, FormatText, FormatToon:
		return format, nil
	default:
		return "", fmt.Errorf("invalid --output: %q", raw)
	}
}

// EmitRecord writes one operation record.
func (e Emitter) EmitRecord(data json.RawMessage) error {
	switch e.format {
	case FormatText:
		_, err := fmt.Fprintf(e.stdout, "%s\n", string(data))
		return err

	case FormatToon:
		var v any
		if err := json.Unmarshal(data, &v); err != nil {
			return fmt.Errorf("unmarshal for toon encoding: %w", err)
		}

		var opts []gotoon.EncodeOption
		if e.toonOpts.Indent != DefaultToonIndent {
			opts = append(opts, gotoon.WithIndent(e.toonOpts.Indent))
		}
		if e.toonOpts.Delimiter != DefaultToonDelimiter {
			opts = append(opts, gotoon.WithDelimiter(e.toonOpts.Delimiter))
		}
		if e.toonOpts.LengthMarker {
			opts = append(opts, gotoon.WithLengthMarker())
		}

		encoded, err := gotoon.Encode(v, opts...)
		if err != nil {
			return fmt.Errorf("toon encoding: %w", err)
		}

		_, err = fmt.Fprintln(e.stdout, encoded)
		return err

	case FormatJSONL:
		recordBytes, err := json.Marshal(successRecord{Data: data})
		if err != nil {
			return fmt.Errorf("marshal success record: %w", err)
		}
		_, err = fmt.Fprintln(e.stdout, string(recordBytes))
		return err
	}

	return fmt.Errorf("unknown format: %s", e.format)
}

// EmitError writes one error object.
func (e Emitter) EmitError(err *atlaserr.Error) error {
	if e.format == FormatText || e.format == FormatToon {
		_, writeErr := fmt.Fprintf(e.stderr, "%s: %s\n", err.Code, err.Message)
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
