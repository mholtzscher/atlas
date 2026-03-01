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
	Data json.RawMessage `json:"data"`
}

// Pagination contains metadata about paginated results.
type Pagination struct {
	HasMore    bool   `json:"hasMore"`
	NextCursor string `json:"nextCursor,omitempty"`
	Total      int    `json:"total,omitempty"`
	Returned   int    `json:"returned"`
}

// paginationRecord wraps pagination metadata for JSONL output.
type paginationRecord struct {
	Pagination *Pagination `json:"pagination"`
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
func (e Emitter) EmitRecord(data json.RawMessage) error {
	switch e.format {
	case FormatText:
		_, err := fmt.Fprintf(e.stdout, "%s\n", string(data))
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
	if e.format == FormatText {
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

// EmitPagination writes pagination metadata if applicable.
// When hasMore is false, the record is omitted entirely for JSONL format.
func (e Emitter) EmitPagination(pagination *Pagination) error {
	if pagination == nil || !pagination.HasMore {
		return nil
	}

	switch e.format {
	case FormatText:
		// Text format doesn't include pagination metadata
		return nil

	case FormatJSONL:
		recordBytes, err := json.Marshal(paginationRecord{Pagination: pagination})
		if err != nil {
			return fmt.Errorf("marshal pagination record: %w", err)
		}
		_, err = fmt.Fprintln(e.stdout, string(recordBytes))
		return err
	}

	return fmt.Errorf("unknown format: %s", e.format)
}

// Format returns emitter output format.
func (e Emitter) Format() Format {
	return e.format
}
