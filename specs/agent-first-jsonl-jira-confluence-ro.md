# Feature: Atlas v1 agent-first JSONL + Jira/Confluence read-only

### Problem Statement
**Who:** an agent harness that shells out to `atlas` to query Jira/Confluence Cloud
**What:** needs stable, machine-friendly, token-efficient output and a way to restrict what ops the agent can execute
**Why it matters:** agents need predictable parsing + low tokens; harness needs a clean allowlist boundary (no “do anything” command)
**Evidence:** requirement from prompt; repo currently has only a demo command and no output/auth/policy conventions

### Proposed Solution
Implement `atlas` as an explicit-operation CLI (no generic HTTP passthrough), defaulting to JSON Lines on stdout for success data and JSON Lines on stderr for structured errors. Read-only Jira + Confluence Cloud commands stream results (one object per item) with minimal default fields/expansions and opt-in “bigger” payload via flags.

Exception: CLI help output is always human text (even when `--output=jsonl`).

Permissions model is harness-owned/enforced: the harness restricts which `atlas <op>` invocations are permitted. `atlas` supports this by:
- keeping operations fine-grained (so harness can allowlist by command path)
- shipping `atlas meta ops` (machine output) so the harness can discover op IDs + mutation classification
- avoiding an escape hatch command that can hit arbitrary endpoints

Authentication in v1 is PAT (email+API token) via flags/env only; OAuth is a follow-on.

### Scope & Deliverables
| Deliverable | Effort | Depends On |
|-------------|--------|------------|
| [D1] Global output contract: `--output jsonl|text`, JSONL default, structured stderr errors | M | - |
| [D2] Auth + HTTP client seam (PAT now, OAuth-ready interface) | M | D1 |
| [D3] Jira read-only ops: issue get + JQL search (Cloud v3) with streaming + pagination | L | D2 |
| [D4] Confluence read-only ops: page get + CQL search (Cloud v2) with streaming + pagination | L | D2 |
| [D5] `atlas meta ops` (ops registry for harness allowlisting) | S | D1 |
| [D6] Blackbox tests (testscript) for output shapes, pagination, and error records | M | D1-D5 |

### Non-Goals (Explicit Exclusions)
- Write/mutate operations (create/update/transition) in Jira/Confluence
- Local response caching
- Storing secrets to disk (no credential config file in v1)
- Generic “raw request” / “proxy” command that can call arbitrary Atlassian endpoints
- Full OAuth2 (3LO) flow in v1 (only interface seam + future shape)

### Data Model

#### Operation IDs (for harness policy)
`atlas meta ops` emits these op IDs (JSONL, one per op):
- `jira.issue.get` (read)
- `jira.issue.search` (read)
- `confluence.page.get` (read)
- `confluence.page.search` (read)

Each `meta ops` line:
```json
{"op":"jira.issue.get","mutates":false,"auth":["pat","oauth"],"args":{"positional":["issueKey"],"flags":["fields","expand"]}}
```

#### Output records (stdout, JSONL)
All successful JSONL records are objects with:
- `op` (string) operation ID
- `data` (object) op-specific payload

Example (jira issue get):
```json
{"op":"jira.issue.get","data":{"key":"ABC-123","fields":{"summary":"...","status":{"name":"In Progress"}}}}
```

For streaming list/search ops, stdout is a sequence of records (one per item). No summary record by default; EOF indicates completion.

#### Help output (always text)
`--help` (and command-specific help) always prints text usage to stdout; `--output` does not change this. This is an explicit exception to the “stdout is JSONL” contract.

`--version` is also always text.

#### Error records (stderr, JSONL)
On any failure, emit exactly one JSON object to stderr (in `--output=jsonl`) and exit non-zero.

```json
{"error":{"code":"AUTH_FAILED","message":"...","op":"jira.issue.search","retryable":false,"details":{}}}
```

Error codes (initial set):
- `INVALID_ARGUMENT`
- `AUTH_FAILED`
- `FORBIDDEN`
- `NOT_FOUND`
- `RATE_LIMITED` (include `retryAfterSeconds` when known)
- `UPSTREAM_ERROR` (non-2xx not mapped above)
- `NETWORK_ERROR`

### API/Interface Contract

#### Global flags / env
Global flags (apply to all ops):
- `--output` (`jsonl` default, `text` optional)
- `--site` (Atlassian site base URL, e.g. `https://acme.atlassian.net`) (env `ATLAS_SITE`)
- `--auth` (`pat` default; `oauth` reserved)
- `--email` (PAT username) (env `ATLAS_EMAIL`)
- `--api-token` (PAT secret) (env `ATLAS_API_TOKEN`)
- `--timeout` (default 30s)
- existing: `--verbose`, `--no-color` (keep; verbose must log to stderr only)

Precedence: flags > env. If required auth inputs missing, return `INVALID_ARGUMENT`.

PAT request auth header (v1):
```text
Authorization: Basic base64(<email>:<api_token>)
```
Never print secrets (including in `--verbose`).

User-Agent: `atlas/<version>`.

#### Jira ops (Cloud REST v3)
Base: `{site}/rest/api/3` (PAT mode).

1) `atlas jira issue get <ISSUE_KEY>`
- Endpoint: `GET /issue/{issueIdOrKey}`
- Flags:
  - `--fields` (additional fields added to compact defaults; repeat flag to add multiple)
  - `--expand` (comma list; default empty)
  - `--raw` (emit full Jira issue payload; requests `fields=*all`)
- Default `fields` (token-efficient): `summary,status,issuetype,priority,assignee,reporter,project,created,updated`
- Requests always send `fieldsByKeys=true`
- Output defaults to compact issue projection

2) `atlas jira issue search --jql <JQL>`
- Endpoint: `GET /search/jql`
- Pagination: token-based via `nextPageToken` (loop until empty or `--limit` reached)
- Flags:
  - `--fields`, `--expand`, `--raw` (same defaults as get)
  - `--limit` (max items; default 50)
  - `--page-size` (API `maxResults`; default 50)
  - `--page-token` (start token; default empty)

Stdout: one JSONL record per issue with `op:"jira.issue.search"`.

#### Confluence ops (Cloud REST v2)
Base: `{site}/wiki/api/v2` (PAT mode).

1) `atlas confluence page get <PAGE_ID>`
- Endpoint: `GET /pages/{id}`
- Flags:
  - `--body-format` (`none` default; `storage|atlas_doc_format|view|...`)
  - `--include-labels` (bool; default false)
  - `--include-properties` (bool; default false)
  - `--include-operations` (bool; default false)
  - `--include-versions` (bool; default false)
  - `--raw` (emit full Confluence payload; enables include/body options)

Default `body-format=none` means do not request/emit page body.
Output defaults to compact projection.

2) `atlas confluence page search --cql <CQL>`
- Endpoint: `GET /content/search?cql=...` (under v2 base)
- Pagination: cursor-based; follow `_links.next` until absent or `--limit` reached
- Flags:
  - `--limit` (default 25)
  - `--page-size` (API `limit`; default 25)
  - `--cursor` (start cursor; default empty)
  - same include/body flags as `page get` (defaults token-efficient)
  - `--raw` (emit full Confluence payload; enables include/body options)

Stdout: one JSONL record per page with `op:"confluence.page.search"`.

#### OAuth (future)
Define an auth abstraction that can later switch base URLs to Atlassian “ex” domains:
- Jira: `https://api.atlassian.com/ex/jira/{cloudId}/rest/api/3/...`
- Confluence: `https://api.atlassian.com/ex/confluence/{cloudId}/wiki/...`

No OAuth commands in v1.

#### Exit codes
- `0` success
- `1` any failure (v1). (Future: reserved codes for auth vs rate-limit.)

### Acceptance Criteria
- [ ] Default output is JSONL; successful runs write only JSONL to stdout
- [ ] `--help` always prints text usage to stdout (even when `--output=jsonl`)
- [ ] `--version` always prints text to stdout
- [ ] In JSONL mode, failures emit exactly one structured error object to stderr and exit non-zero
- [ ] Jira search uses `/rest/api/3/search/jql` and correctly follows `nextPageToken`
- [ ] Confluence search follows cursor pagination via `_links.next`
- [ ] Default payloads are token-efficient (minimal fields; no Confluence body by default)
- [ ] No command exists that can call arbitrary endpoints (keeps harness allowlist meaningful)
- [ ] `atlas meta ops` lists all ops with stable `op` IDs and `mutates=false`

### Test Strategy
| Layer | What | How |
|------|------|-----|
| Unit | pagination loops, flag/env precedence, error mapping | table-driven tests on internal packages |
| Integration (blackbox) | CLI output shape + stderr behavior | `test/testscript` scripts asserting JSONL lines |

Mock upstream:
- Use `httptest.Server` for unit/integration boundary tests; inject base URL via `--site`.

### Risks & Mitigations
| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Atlassian API churn (Jira search already changed) | Med | High | pin to `/search/jql`; isolate API client per product |
| Token bloat from default expansions | Med | Med | conservative defaults; explicit flags to opt in |
| OAuth base URL + cloudId complexity | High | Med | keep PAT-only in v1; auth interface seam |
| Rate limit behavior surprises | Med | Med | map 429 to `RATE_LIMITED` + retry-after; document |

### Trade-offs Made
| Chose | Over | Because |
|------|------|---------|
| harness-only permissions | CLI-enforced policy | avoids duplicating policy logic; keep atlas simple; rely on fine-grained ops + no escape hatch |
| JSONL default | text default | agent-first + streaming; testability |
| conservative defaults | full raw payloads | token efficiency, predictable parsing |

### Open Questions
- None (v1 bounded to Cloud + PAT; permissions enforced by harness)

### Success Metrics
- Harness can allowlist by command path/op ID without needing deep parsing
- Typical `search` output size reduced vs raw upstream by default field selection
- Stable JSONL parsing across ops (no incidental stdout noise)
