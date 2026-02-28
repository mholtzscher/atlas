# TOON Output Format Support

**Status:** Ready for Implementation  
**Type:** Feature Plan  
**Effort:** L (1-2 days)  
**Priority:** Medium  

## Summary

Add "toon" (Token-Oriented Object Notation) output format using the [gotoon](https://github.com/alpkeskin/gotoon) library. TOON provides 30-60% token reduction compared to JSON when feeding data to LLMs, while remaining human-readable.

## Motivation

Atlas outputs structured data from Jira and Confluence. Users currently pipe JSONL through external tools to optimize for LLM consumption. Native toon support enables:

1. **Direct LLM pipelines:** `atlas jira issue search --output=toon | llm-process`
2. **Compact terminal viewing:** More readable than JSONL for complex objects
3. **Reduced token costs:** Significant savings when sending Atlassian data to LLM APIs

## Current State

- Supported formats: `jsonl` (default), `text`
- Output system: `internal/output/output.go` with `Format` type and `Emitter` struct
- Configuration: Global `--output` flag in `cmd/root.go`
- Runtime: `runtime.New()` creates preconfigured `Emitter`

## Requirements

### Functional Requirements

1. **FR1:** Add `toon` as valid `--output` format option
2. **FR2:** Encode operation records using gotoon format when `--output=toon`
3. **FR3:** Support gotoon configuration options via CLI flags:
   - Indentation spaces (default: 2)
   - Delimiter type: comma, tab, pipe (default: comma)
   - Array length marker prefix (default: false)
4. **FR4:** Keep error output in simple text format (consistent with `text` format)
5. **FR5:** Support toon format in config file (`atlas.json`)

### Non-Goals

- Per-command format override (format remains global)
- Custom toon schemas or field selection
- Streaming partial toon output (emit complete records only)

## Design

### Architecture

```
┌─────────────┐     ┌──────────────┐     ┌─────────────────┐
│   root.go   │────▶│  options.go  │────▶│   output.go     │
│   --output  │     │ GlobalOptions│     │     Emitter     │
│ --toon-*    │     │  ToonConfig  │     │  EmitRecord()   │
└─────────────┘     └──────────────┘     └─────────────────┘
                                                   │
                                                   ▼
                                          ┌─────────────────┐
                                          │  gotoon.Encode  │
                                          │  json → toon    │
                                          └─────────────────┘
```

### Data Flow

1. User runs: `atlas jira issue get PROJ-123 --output=toon`
2. `cmd/root.go` validates flags, sets `GlobalOptions.Output = FormatToon`
3. `runtime.New()` creates `Emitter` with format and toon options
4. Operation returns `json.RawMessage` with issue data
5. `Emitter.EmitRecord()` detects `FormatToon`, unmarshals JSON to `interface{}`
6. Calls `gotoon.Encode()` with configured options
7. Writes toon string to stdout

### Key Decisions

**Error Format:** Keep simple text errors (`CODE (op): message` on stderr) rather than encoding in toon. This matches existing `text` format behavior and ensures errors are always human-readable even when piping output.

**Flag Scope:** Global flags only. Format is a global concern in atlas; per-command override would require significant Emitter refactoring.

**JSON Unmarshaling:** gotoon requires `interface{}` not `json.RawMessage`. Emitter will need to unmarshal JSON before encoding. This adds minimal overhead and keeps the gotoon integration clean.

## Implementation

### Deliverables (Ordered)

| # | Deliverable | Effort | Depends On | Files Changed |
|---|-------------|--------|------------|---------------|
| D1 | Add gotoon dependency | S | - | `go.mod`, `go.sum`, `gomod2nix.toml` |
| D2 | Extend Format type | S | - | `internal/output/output.go` |
| D3 | Add toon-specific flags | M | D1 | `cmd/root.go`, `internal/cli/options.go` |
| D4 | Extend GlobalOptions | S | D3 | `internal/cli/options.go` |
| D5 | Implement toon encoding | M | D2, D4 | `internal/output/output.go` |
| D6 | Update flag validation | S | D2 | `cmd/root.go` |
| D7 | Run tests and checks | M | D5 | All |

### File Changes

#### D1: Add gotoon dependency

```bash
go get github.com/alpkeskin/gotoon
just update-deps  # Updates gomod2nix.toml
```

#### D2: Extend Format type

**File:** `internal/output/output.go`

Add constant and update ParseFormat:

```go
const (
    FormatJSONL Format = "jsonl"
    FormatText  Format = "text"
    FormatToon  Format = "toon"  // NEW
)

func ParseFormat(raw string) (Format, error) {
    if raw == "" {
        return FormatJSONL, nil
    }

    format := Format(raw)
    switch format {
    case FormatJSONL, FormatText, FormatToon:  // ADD FormatToon
        return format, nil
    default:
        return "", fmt.Errorf("invalid --output: %q", raw)
    }
}
```

#### D3: Add toon-specific flags

**File:** `internal/cli/options.go`

Add flag constants:

```go
const (
    // ... existing constants
    
    // Toon-specific flags
    FlagToonIndent       = "toon-indent"
    FlagToonDelimiter  = "toon-delimiter"
    FlagToonLengthMarker = "toon-length-marker"
)
```

Add ToonConfig struct:

```go
// ToonConfig holds gotoon encoding options.
type ToonConfig struct {
    Indent       int
    Delimiter    string
    LengthMarker bool
}

// Extend GlobalOptions:
type GlobalOptions struct {
    Output    output.Format
    Toon      ToonConfig  // NEW
    Site      string
    // ... rest
}
```

**File:** `cmd/root.go`

Add toon flags to `globalFlags()`:

```go
&ufcli.IntFlag{
    Name:    cli.FlagToonIndent,
    Value:   2,
    Usage:   "TOON indentation spaces",
    Sources: configSources(cli.FlagToonIndent),
},
&ufcli.StringFlag{
    Name:    cli.FlagToonDelimiter,
    Value:   "comma",
    Usage:   "TOON delimiter: comma, tab, pipe",
    Sources: configSources(cli.FlagToonDelimiter),
    Action: func(_ context.Context, _ *ufcli.Command, v string) error {
        if v != "comma" && v != "tab" && v != "pipe" {
            return fmt.Errorf("invalid --%s: %q (must be 'comma', 'tab', or 'pipe')", cli.FlagToonDelimiter, v)
        }
        return nil
    },
},
&ufcli.BoolFlag{
    Name:    cli.FlagToonLengthMarker,
    Value:   false,
    Usage:   "Add # prefix to TOON array lengths",
    Sources: configSources(cli.FlagToonLengthMarker),
},
```

Update GlobalOptionsFromCommand to extract toon options.

#### D4: Extend GlobalOptions

**File:** `internal/cli/options.go`

Update extraction function:

```go
func GlobalOptionsFromCommand(cmd *ufcli.Command) GlobalOptions {
    out, _ := output.ParseFormat(cmd.String(FlagOutput))
    
    // Parse delimiter
    delimiter := ","
    switch cmd.String(FlagToonDelimiter) {
    case "tab":
        delimiter = "\t"
    case "pipe":
        delimiter = "|"
    }

    return GlobalOptions{
        Output: out,
        Toon: ToonConfig{
            Indent:       cmd.Int(FlagToonIndent),
            Delimiter:    delimiter,
            LengthMarker: cmd.Bool(FlagToonLengthMarker),
        },
        // ... rest
    }
}
```

#### D5: Implement toon encoding

**File:** `internal/output/output.go`

Add gotoon import and extend Emitter:

```go
import (
    "encoding/json"
    "fmt"
    "io"
    
    "github.com/alpkeskin/gotoon"  // NEW
    "github.com/mholtzscher/atlas/internal/atlaserr"
)

// ToonOptions holds gotoon encoding configuration.
type ToonOptions struct {
    Indent       int
    Delimiter    string
    LengthMarker bool
}

// Extend Emitter to include toon options:
type Emitter struct {
    format    Format
    toonOpts  ToonOptions  // NEW
    stdout    io.Writer
    stderr    io.Writer
}

// NewEmitter updated signature:
func NewEmitter(format Format, toonOpts ToonOptions, stdout, stderr io.Writer) Emitter {
    return Emitter{
        format:   format,
        toonOpts: toonOpts,
        stdout:   stdout,
        stderr:   stderr,
    }
}

// Extend EmitRecord:
func (e Emitter) EmitRecord(op string, data json.RawMessage) error {
    switch e.format {
    case FormatText:
        _, err := fmt.Fprintf(e.stdout, "%s\t%s\n", op, string(data))
        return err
        
    case FormatToon:  // NEW CASE
        var v interface{}
        if err := json.Unmarshal(data, &v); err != nil {
            return fmt.Errorf("unmarshal for toon encoding: %w", err)
        }
        
        var opts []gotoon.EncodeOption
        if e.toonOpts.Indent != 2 {
            opts = append(opts, gotoon.WithIndent(e.toonOpts.Indent))
        }
        if e.toonOpts.Delimiter != "," {
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
        
    default: // FormatJSONL
        recordBytes, err := json.Marshal(successRecord{Op: op, Data: data})
        if err != nil {
            return fmt.Errorf("marshal success record: %w", err)
        }
        _, err = fmt.Fprintln(e.stdout, string(recordBytes))
        return err
    }
}
```

**File:** `internal/runtime/runtime.go`

Update runtime to pass toon options:

```go
// In runtime.New(), when creating emitter:
emitter := output.NewEmitter(
    opts.Output,
    output.ToonOptions{
        Indent:       opts.Toon.Indent,
        Delimiter:    opts.Toon.Delimiter,
        LengthMarker: opts.Toon.LengthMarker,
    },
    stdout,
    stderr,
)
```

#### D6: Update flag validation

**File:** `cmd/root.go`

Update `--output` validation:

```go
Action: func(_ context.Context, _ *ufcli.Command, v string) error {
    if v != "jsonl" && v != "text" && v != "toon" {  // ADD "toon"
        return fmt.Errorf("invalid --%s: %q (must be 'jsonl', 'text', or 'toon')", cli.FlagOutput, v)
    }
    return nil
},
```

#### D7: Run tests and checks

```bash
just check
```

Verify:
- All tests pass
- No linting errors
- `atlas --output=toon jira issue get PROJ-123` produces valid toon
- Config file settings work: `echo '{"atlas": {"output": "toon", "toon-indent": 4}}' > atlas.json`

## Acceptance Criteria

- [ ] `atlas --output=toon jira issue get KEY` outputs valid toon format
- [ ] `atlas --output=toon jira issue search QUERY` streams multiple toon records
- [ ] `atlas --toon-indent=4 --output=toon ...` uses 4-space indentation
- [ ] `atlas --toon-delimiter=tab --output=toon ...` uses tab delimiter
- [ ] `atlas --toon-length-marker --output=toon ...` adds `#` prefix to arrays
- [ ] Config file `atlas.json` supports all toon options under `atlas.*`
- [ ] Errors print as text on stderr regardless of output format
- [ ] Invalid `--output` value shows helpful error with valid options
- [ ] All existing tests pass

## Testing Strategy

### Unit Tests

Add tests in `internal/output/output_test.go`:

```go
func TestEmitterEmitRecord_Toon(t *testing.T) {
    tests := []struct {
        name     string
        format   Format
        toonOpts ToonOptions
        data     json.RawMessage
        want     string
    }{
        {
            name:   "simple object",
            format: FormatToon,
            data:   json.RawMessage(`{"id":123,"name":"Test"}`),
            want:   "id: 123\nname: Test\n",
        },
        {
            name:     "array with custom indent",
            format:   FormatToon,
            toonOpts: ToonOptions{Indent: 4},
            data:     json.RawMessage(`{"items":[{"a":1},{"a":2}]}`),
            want:     "items[2]{a}:\n    1\n    2\n",
        },
    }
    // ... test implementation
}
```

### Integration Tests

- Test with real Jira/Confluence API responses
- Verify token count reduction vs JSONL using tokenizer
- Test streaming with search operations (multiple records)

### Manual Testing

```bash
# Basic toon output
atlas --output=toon jira issue get PROJ-123

# With custom options
atlas --output=toon --toon-indent=4 --toon-delimiter=pipe jira issue get PROJ-123

# Config file
cat > atlas.json << 'EOF'
{
  "atlas": {
    "output": "toon",
    "toon-indent": 2,
    "toon-length-marker": true
  }
}
EOF
atlas jira issue get PROJ-123
```

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| gotoon doesn't handle complex nested structures | Medium | Test with real API responses; gotoon normalizes structs/maps/slices automatically |
| Adding 3 new global flags clutters help | Low | Flags are logically grouped under "toon" prefix; only appear in global help |
| Performance: JSON unmarshal + toon encode overhead | Low | Acceptable for the benefit; both operations are fast for typical API responses |
| Config file validation for delimiter values | Low | Validation happens at flag parsing, config file bypasses this - document expected values |

## Open Questions

- [ ] Should we expose `--toon-*` flags only when `--output=toon`? (urfave/cli/v3 doesn't support conditional flags easily - skip for now)

## References

- gotoon library: https://github.com/alpkeskin/gotoon
- TOON format spec: See gotoon README
- Existing output system: `internal/output/output.go`
