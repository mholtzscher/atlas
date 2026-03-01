# Spec: Jira `--fields` Additive + Remove `--fields-by-keys`

Type: Feature plan  
Effort: S (1-2h)  
Status: Draft

## Problem
We want compact, high-signal Jira defaults while allowing users to request extra fields without replacing defaults. Current behavior treats `--fields` as set/replace and exposes `--fields-by-keys`, which adds key-vs-ID ambiguity.

## Goals
- `--fields` is additive: defaults always included; user fields appended (deduped).
- Remove `--fields-by-keys` from CLI and op registry; always request by field key.
- Keep payload handling generic (`json.RawMessage`), no per-op response structs.

## Non-goals
- No escape hatch to replace/remove default fields.
- No new raw/full profile in this change.
- No change to `--expand` semantics.

## User-facing contract

### Affected commands
- `atlas jira issue get <ISSUE_KEY>`
- `atlas jira issue search --jql <JQL>`

### Default fields (always requested)
`summary,status,issuetype,priority,assignee,reporter,project,created,updated`

### `--fields` semantics (additive)
- Effective fields = `DefaultFields + user --fields` (stable order, deduped).
- Preferred usage is repeated flags:
  - `--fields labels --fields components`
- No guaranteed comma-splitting behavior inside one flag value.

### Special Jira values
- `--fields *all` and `--fields *navigable` are allowed and still additive.

### Remove `--fields-by-keys`
- Flag removed from commands.
- Implementation always sends `fieldsByKeys=true`.

## Design / Implementation

### Centralize additive merge
File: `internal/jira/operations.go`

- Remove `FieldsByKeys` from:
  - `GetIssueRequest`
  - `SearchIssuesRequest`
  - `buildIssueQuery` signature
- `buildIssueQuery(fields, expand)` should:
  - set `fieldsByKeys=true`
  - build effective fields as defaults + user values
  - trim whitespace, drop empty, stable dedupe
  - set `fields=<comma-joined effective list>`

### CLI changes
File: `cmd/jira/jira.go`

- Remove `--fields-by-keys` constant, flag registration, and request mapping.
- Update `--fields` help text to:
  - `Additional issue fields (added to compact defaults)`

### Docs/spec update
File: `specs/agent-first-jsonl-jira-confluence-ro.md`

- Remove `--fields-by-keys` mention.
- Update `--fields` description to additive semantics.

## Deliverables (ordered)
1. [D1] Remove `--fields-by-keys` from CLI (S)
2. [D2] Implement additive field merge in Jira ops (S)
3. [D3] Update docs/spec text (S)
4. [D4] Tests and `just check` (S)

## Testing
- Table-driven tests for effective field merge:
  - no user fields => defaults only
  - user adds new + existing default => no dup, new appended
  - repeated flags => deduped
  - `*all` allowed
- Validate `buildIssueQuery` sets `fieldsByKeys=true` and expected `fields` value.

## Risks / Trade-offs
- Breaking change for users/scripts relying on `--fields-by-keys`.
- Users cannot use `--fields` to reduce payload anymore (intentional).

## Acceptance criteria
- `atlas jira issue get KEY` requests defaults and sends `fieldsByKeys=true`.
- `atlas jira issue get KEY --fields labels` requests defaults + `labels`, no duplicates.
- CLI help no longer exposes `--fields-by-keys`.
- `just check` passes.

## Open questions
- None.
