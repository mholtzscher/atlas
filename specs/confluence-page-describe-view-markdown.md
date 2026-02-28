# Confluence Page Describe/View Markdown Spec

**Status:** Ready for task breakdown  
**Date:** 2026-02-28  
**Type:** Feature plan  
**Effort:** L (1-2 days)

## Problem Statement

**Who:** atlas CLI users and agent workflows consuming Confluence pages.  
**What:** users need clean Markdown page content from CLI, while retaining a metadata-first page inspection command.  
**Why it matters:** current `confluence page get` is metadata-oriented (compact output drops `body`) and does not provide a direct Markdown content path.  
**Evidence:** `internal/confluence/compact.go` drops `body`; `cmd/confluence/confluence.go` compacts Confluence records by default unless `--raw`.

## Discovery Summary

- CLI routing and output emission are in `cmd/confluence/confluence.go`; current `page get` uses `confluenceops.GetPage` then `maybeCompactConfluenceRecord` then `Emitter.EmitRecord`.
- Confluence request/query behavior is in `internal/confluence/operations.go` (`GetPage`, `buildQuery`, body-format constants).
- Compact projection is in `internal/confluence/compact.go` and intentionally drops `body`.
- Operation allowlisting and machine-visible op IDs are in `internal/ops/registry.go`.
- Confluence Cloud REST v2 supports `body.view.value` for `GET /wiki/api/v2/pages/{id}` and this is suitable for HTML -> Markdown conversion.

## Decision

### Command model

- Replace `atlas confluence page get <PAGE_ID>` with `atlas confluence page describe <PAGE_ID>` (breaking CLI rename).
- Add `atlas confluence page view <PAGE_ID>` for Markdown body content.

### Output model

- `page describe` emits compact metadata record via existing emitter (`jsonl`/`text`/`toon`).
- `page view` emits Markdown only to stdout (raw text stream, no `{op,data}` envelope).

### Conversion model

- Fetch page with `body-format=view`.
- Extract `body.view.value` (HTML string).
- Convert HTML -> Markdown using `github.com/JohannesKaufmann/html-to-markdown/v2` (best-effort fidelity).
- Unknown/complex constructs (macros, smart links, merged table edge cases) are tolerated as lossy output; command does not fail solely for unsupported semantics.

## Scope & Deliverables

| Deliverable | Effort | Depends On |
|-------------|--------|------------|
| D1. Rename `page get` command to `page describe` and update help/usage | S | - |
| D2. Add `page view` command that prints Markdown only | M | D1 |
| D3. Implement Confluence page HTML extraction + HTML->MD conversion helper | M | D2 |
| D4. Add operation IDs/registry entries for new command shape | S | D1, D2 |
| D5. Add tests for extraction, conversion, and command surface changes | M | D1-D4 |

## Non-Goals

- No markdown rendering for `page search` in this iteration.
- No markdown rendering for comments in this iteration.
- No global `--output=markdown` format.
- No high-fidelity macro/smart-link semantic preservation beyond best-effort text rendering.

## Detailed Design

### 1) CLI changes

In `cmd/confluence/confluence.go`:

- Replace `newPageGetCommand()` registration with:
  - `newPageDescribeCommand()` (metadata)
  - `newPageViewCommand()` (content)
- `page describe`:
  - Positional arg: `<PAGE_ID>`.
  - Keeps metadata include flags (`--include-labels`, `--include-properties`, `--include-operations`, `--include-versions`).
  - Forces request `BodyFormatNone` so content is not fetched.
  - Uses existing compact flow (`maybeCompactConfluenceRecord`) and emitter.
- `page view`:
  - Positional arg: `<PAGE_ID>`.
  - Forces `BodyFormatView`.
  - Bypasses emitter for success output and writes markdown to `cmd.Writer`.

### 2) Operation IDs and allowlist

In `internal/ops/registry.go`:

- Replace `OpConfluencePageGet` with `OpConfluencePageDescribe`.
- Add `OpConfluencePageView`.
- Update confluence definitions:
  - `confluence.page.describe` with metadata flags.
  - `confluence.page.view` with positional `pageID` and no content-format flag.

### 3) Markdown conversion helper

Add `internal/confluence/markdown.go` (or equivalent):

- `func ExtractPageViewHTML(page json.RawMessage) (string, error)`
  - Parse `body.view.value` as required string.
  - Return upstream-style error when missing/invalid.
- `func RenderHTMLAsMarkdown(html string) (string, error)`
  - Use html-to-markdown converter with base/commonmark/table plugins.
  - Normalize output to end with trailing newline.

Suggested error behavior:

- Missing `body.view.value` -> `atlaserr.New(CodeUpstreamError, "page body view missing", ops.OpConfluencePageView, false, nil)`.
- Conversion failure -> wrapped as upstream error with op `confluence.page.view`.

### 4) Command execution flow for `page view`

1. Validate one page ID arg.
2. Build runtime with op `confluence.page.view`.
3. Call `confluenceops.GetPage` with `SearchOptions{BodyFormat: BodyFormatView}`.
4. Extract HTML from `body.view.value`.
5. Convert to markdown.
6. Write markdown to stdout only.

## Data Model

Input (Confluence page JSON subset):

```json
{
  "id": "123",
  "title": "Example",
  "body": {
    "view": {
      "representation": "view",
      "value": "<h1>Heading</h1><p>...</p>"
    }
  }
}
```

Output for `page view`:

- Plain markdown string (not JSON-encoded), body only.

Output for `page describe`:

- Existing compact JSON record via emitter (operation envelope depends on global `--output`).

## API/Interface Contract

### CLI

- `atlas confluence page describe <PAGE_ID> [--include-labels --include-properties --include-operations --include-versions]`
- `atlas confluence page view <PAGE_ID>`

### Breaking change

- `atlas confluence page get` removed in favor of `atlas confluence page describe`.

## Acceptance Criteria

- [ ] `atlas confluence page get` is not present in help/subcommands.
- [ ] `atlas confluence page describe <id>` returns compact metadata and excludes body content by default.
- [ ] `atlas confluence page view <id>` returns markdown body only (no envelope, no metadata prelude).
- [ ] `page view` uses Confluence `body.view.value` and converts common HTML blocks (headings, paragraphs, links, emphasis, lists, code blocks, tables).
- [ ] Missing or invalid `body.view.value` returns structured atlas error tagged with `confluence.page.view`.
- [ ] `internal/ops/registry.go` reflects new op IDs and args.
- [ ] `just check` passes.

## Test Strategy

| Layer | What | How |
|-------|------|-----|
| Unit | HTML extraction from page payload | table-driven tests in new `internal/confluence/markdown_test.go` |
| Unit | HTML->MD conversion baseline | stable fixtures/snippets for headings/lists/code/tables |
| Unit/Integration-lite | command surface rename | verify `page describe` exists and `page get` removed (testscript help assertions) |

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| HTML conversion loses macro semantics | Medium | Medium | document best-effort behavior; keep converter isolated for future tuning |
| CLI rename breaks existing scripts | Medium | Medium | explicit breaking-change note in release docs/changelog |
| Converter output variability across versions | Low | Medium | pin dependency version; test core constructs only |

## Trade-offs Made

| Chose | Over | Because |
|-------|------|---------|
| Separate `describe` + `view` commands | single overloaded `get` | clearer UX separation between metadata vs content |
| markdown-only stdout for `view` | emitter envelope output | direct piping and file redirection with no post-processing |
| HTML(view)->Markdown | ADF->Markdown in v1 | simpler single-page implementation; closer to rendered content |

## Implementation Order

1. D1 CLI rename and command tree updates.
2. D4 op registry updates.
3. D3 markdown helper + dependency.
4. D2 wire `page view` flow.
5. D5 tests and final `just check`.

## Files Likely Touched

- `cmd/confluence/confluence.go`
- `internal/confluence/operations.go` (if helper extraction kept here) or new helper file in same package
- `internal/confluence/markdown.go` (new)
- `internal/confluence/markdown_test.go` (new)
- `internal/ops/registry.go`
- `go.mod`, `go.sum`
- `test/testscript/scripts/basic.txtar` (if command help assertions added)

## Completeness Check

- Scope bounded: pass
- Ambiguity resolved: pass
- Acceptance testable: pass
- Dependencies ordered: pass
- Types/data shapes specified: pass
- Effort estimated: pass
- Risks identified: pass
- Open questions resolved: pass

## Open Questions

- None
