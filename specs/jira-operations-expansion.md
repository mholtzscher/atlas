# Spec: Jira Operations Expansion

**Status**: Ready for implementation  
**Type**: Feature Plan  
**Effort**: M (2-3 hours)  
**Target**: Jira Cloud REST API v3

---

## Problem Statement

Users need to:
1. **Discover projects** before searching issues (need project keys)
2. **View comments** on specific issues
3. **List issue types** for creating new issues
4. **Verify authentication** (who am I logged in as?)

Currently, the CLI only supports `jira issue get` and `jira issue search`.

---

## Constraints & Decisions

| Constraint | Decision |
|------------|----------|
| **Output formats** | Use existing JSONL/text only |
| **API target** | Jira Cloud only (v3 REST API) |
| **Pagination** | None for MVP (return all results) |
| **Scope** | 4 Jira operations, no Confluence yet |

---

## Solution: Minimal Viable Implementation

All 4 operations are simple GET requests following existing patterns in `internal/jira/operations.go`.

### API Endpoints

| Operation | Endpoint | Auth Required |
|-----------|----------|---------------|
| `jira project list` | `GET /rest/api/3/project/search` | Optional (anonymous) |
| `jira issue comments <KEY>` | `GET /rest/api/3/issue/{key}/comment` | Optional |
| `jira issue types` | `GET /rest/api/3/issuetype` | Optional |
| `jira myself` | `GET /rest/api/3/myself` | **Required** |

### Response Fields (Minimal)

**Projects**: `id`, `key`, `name`, `projectCategory`

**Comments**: `id`, `author.displayName`, `created`, `body` (raw ADF JSON)

**Issue Types**: `id`, `name`, `subtask`, `description`

**Myself**: `accountId`, `displayName`, `emailAddress`, `active`

---

## Deliverables (Ordered)

| # | Deliverable | Effort | Location | Depends On |
|---|-------------|--------|----------|------------|
| 1 | Add operation IDs to registry | S | `internal/ops/registry.go` | - |
| 2 | Implement `jira project list` operation | M | `internal/jira/operations.go` | 1 |
| 3 | Implement `jira issue comments` operation | M | `internal/jira/operations.go` | 1 |
| 4 | Implement `jira issue types` operation | S | `internal/jira/operations.go` | 1 |
| 5 | Implement `jira myself` operation | S | `internal/jira/operations.go` | 1 |
| 6 | Add CLI commands | S | `cmd/jira/jira.go` | 2-5 |
| 7 | Run `just check` | S | - | 6 |

### Operation IDs

```go
OpJiraProjectList   = "jira.project.list"
OpJiraIssueComments = "jira.issue.comments"
OpJiraIssueTypes    = "jira.issue.types"
OpJiraMyself        = "jira.myself"
```

---

## CLI Usage

```bash
# List all projects
atlas jira project list

# Get comments on an issue
atlas jira issue comments PROJ-123

# List all issue types
atlas jira issue types

# Verify current user
atlas jira myself
```

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| ADF format for comments is complex | Consumers need to parse ADF | Return raw ADF JSON; don't attempt conversion |
| No pagination on large lists | Timeout/memory issues | Document limitation; add `--limit` later if needed |
| Issue key validation | Invalid keys return 404 | Reuse existing `issue get` validation pattern |
| `myself` requires auth | 401 error | Return clear error message; useful for auth testing |

---

## Testing

- Add testscript tests in `test/testscript/scripts/jira_*.txtar`
- Test each command with valid and invalid inputs
- Verify JSONL output format matches existing commands

---

## Open Questions

None remaining.

---

**Phase**: DONE  
**Status**: Ready for implementation
