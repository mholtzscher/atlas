# Spec: Confluence Simple Additions (Spaces + Page Comments)

**Status**: Ready for implementation  
**Type**: Feature Plan  
**Effort**: M (3-6 hours)  
**Target**: Confluence Cloud REST API v2

---

## Problem Statement

Add three Confluence read operations to match existing Jira parity and unblock basic automation:

1. `confluence space list` - list accessible spaces
2. `confluence space get <SPACE-KEY>` - fetch space details by key
3. `confluence page comments <PAGE-ID>` - fetch page footer comments including threads (replies)

---

## Constraints & Decisions

| Constraint | Decision |
|------------|----------|
| API target | Confluence Cloud only (`/wiki/api/v2/...`) |
| Auth | Reuse existing global auth (`pat` implemented; `oauth` allowlisted but unimplemented) |
| Output formats | Reuse existing JSONL/text/toon emitters; no bespoke tables |
| Output schema | Emit upstream JSON objects (pass-through) |
| Pagination | Cursor-based, matching `confluence page search` (`--limit`, `--page-size`, `--cursor`) |
| Comments scope | Footer comments only; include reply threads via children traversal |

---

## Solution

### New CLI Commands

```bash
# Spaces
atlas confluence space list [--limit N] [--page-size N] [--cursor CURSOR]
atlas confluence space get <SPACE_KEY>

# Page footer comments (incl replies)
atlas confluence page comments <PAGE_ID> [--limit N] [--page-size N] [--cursor CURSOR] [--body-format <fmt>]
```

Notes:
- `--body-format` reuses the existing page flag. For comments, it is forwarded as `body-format` when not `none`.
- `confluence page comments` emits a flat stream of comment JSON objects. Parent/child relationships depend on upstream fields; traversal order is parent then descendants.

---

## API Endpoints

| Command | Endpoint(s) | Notes |
|---------|-------------|------|
| `space list` | `GET /wiki/api/v2/spaces` | Cursor pagination via `cursor` and `_links.next` |
| `space get <KEY>` | `GET /wiki/api/v2/spaces?keys=<KEY>&limit=1` then `GET /wiki/api/v2/spaces/{id}` | v2 has no direct get-by-key; list-by-key to resolve ID |
| `page comments <ID>` | `GET /wiki/api/v2/pages/{id}/footer-comments` + `GET /wiki/api/v2/footer-comments/{commentId}/children` (recursive) | Fetch roots then traverse children pages until exhausted or `--limit` reached |

---

## Operations / Contracts

### Operation IDs

Add stable op IDs:

```go
OpConfluenceSpaceList   = "confluence.space.list"
OpConfluenceSpaceGet    = "confluence.space.get"
OpConfluencePageComments = "confluence.page.comments"
```

### Allowlist Registry

Update `internal/ops/registry.go`:
- `confluence.space.list`: flags `limit`, `page-size`, `cursor`
- `confluence.space.get`: positional `spaceKey`
- `confluence.page.comments`: positional `pageID`; flags `limit`, `page-size`, `cursor`, `body-format`

---

## Implementation Details

### Command Tree

Update `cmd/confluence/confluence.go`:
- Add `space` command group alongside `page`.
- Add `page comments` command alongside `page get` and `page search`.

Implementation pattern matches existing Confluence commands:
- `runtime.New(cmd, opID, true)`
- Validate positional args (`expected exactly one argument: <...>`)
- For list-like commands, stream items via `emit func(json.RawMessage) error` calling `deps.Emitter.EmitRecord(opID, item)`

### Confluence Operations

Update `internal/confluence/operations.go` with new operations using the existing conventions:

**Spaces**
- `ListSpaces(ctx, client, request, emit)`
  - Request fields: `Limit`, `PageSize`, `Cursor`
  - Loop: request initial URL (`/wiki/api/v2/spaces?limit=<min>&cursor=<cursor>`) then follow `_links.next` via `client.GetURL`.
  - Decode response envelope minimally: `results []json.RawMessage`, `_links.next string`.
- `GetSpaceByKey(ctx, client, spaceKey)`
  - Call list endpoint with `keys=<spaceKey>&limit=1`.
  - If no results: return `atlaserr.New(atlaserr.CodeNotFound, "space not found", OpConfluenceSpaceGet, false, {"spaceKey":...})`.
  - Extract `id` from the returned JSON; then call `GET /wiki/api/v2/spaces/{id}` and emit that JSON.

**Page footer comments (threads)**
- `ListPageFooterComments(ctx, client, request, emit)`
  - Request fields: `PageID`, `Limit`, `PageSize`, `Cursor`, `BodyFormat`.
  - Fetch roots from `/wiki/api/v2/pages/{pageId}/footer-comments` (cursor pagination).
  - For each emitted comment, extract `id` and fetch its children from `/wiki/api/v2/footer-comments/{commentId}/children`.
  - Children fetching is recursive (depth-first), paginated by `_links.next` until exhausted or `Limit` reached.
  - Implementation should be iterative (explicit stack/queue) to avoid unbounded recursion.
  - Guardrails: stop all further requests once `remaining == 0`; track visited comment IDs to avoid cycles.

**ID extraction**
- Comment/space IDs may be numeric or string in upstream JSON; implement a helper that tolerates both and returns a string suitable for URL path escaping.

---

## Acceptance Criteria

- `atlas confluence space list` emits up to `--limit` space records; follows `_links.next` until limit reached or exhausted.
- `atlas confluence space get <SPACE_KEY>`:
  - emits exactly 1 record when key exists
  - returns `NOT_FOUND` when key does not exist (200 + empty results)
  - returns `INVALID_ARGUMENT` when `<SPACE_KEY>` is missing/empty
- `atlas confluence page comments <PAGE_ID>` emits footer comments including replies; stops once `--limit` reached.
- New ops appear in `atlas ops` output (if such command exists) / allowlist registry updated.
- `just check` passes.

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| N+1 requests for comment threads | Slow / rate limits on heavily-threaded pages | Respect `--limit`; keep `--page-size`; stop traversal early; surface 429 via existing `RATE_LIMITED` |
| Thread reconstruction unclear in pass-through output | Consumers may not reliably build a tree | Non-goal for MVP; future enhancement: optional wrapper emitting `parentID`/`depth` derived from traversal context |
| Space get-by-key requires 2 calls | Slight latency | Cache-free; acceptable for single lookup |

---

## Testing

- Add Go unit tests with `httptest.Server` covering:
  - spaces list pagination (`_links.next` follow)
  - space get-by-key empty result -> `NOT_FOUND`
  - comments traversal: roots + children pagination + early stop at `--limit`
- Add/extend testscript coverage for argument validation (missing args) and missing `--site` behavior.

---

## Deliverables (Ordered)

| # | Deliverable | Effort | Location | Depends On |
|---|-------------|--------|----------|------------|
| 1 | Add op IDs + allowlist defs | S | `internal/ops/registry.go` | - |
| 2 | Implement Confluence space ops | M | `internal/confluence/operations.go` | 1 |
| 3 | Implement Confluence page comments op | M | `internal/confluence/operations.go` | 1 |
| 4 | Add CLI commands (`space list/get`, `page comments`) | S | `cmd/confluence/confluence.go` | 2-3 |
| 5 | Add tests | M | `internal/confluence/*_test.go`, `test/testscript/scripts/*` | 2-4 |
| 6 | Run `just check` | S | - | 5 |

---

## Open Questions

None remaining.

---

**Phase**: DONE  
**Status**: Ready for implementation
