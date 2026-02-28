---
name: atlas-cli
description: Interact with Atlassian Jira and Confluence via the atlas CLI. Use when querying issues, searching with JQL/CQL, reading Confluence pages, or listing projects/spaces.
---

# Atlas CLI

Agent-first CLI for Atlassian Cloud products (Jira, Confluence). All operations are read-only.

## Authentication Policy

**NEVER read, write, or modify authentication config files** (`atlas.json`, `~/.config/atlas/atlas.json`). Never pass `--email`, `--api-token`, or `--site` flags inline. Auth is pre-configured by the user via environment variables or config files.

If a command fails with `AUTH_FAILED`, `FORBIDDEN`, or missing `--site`, tell the user:

> Authentication failed. Please check your atlas configuration. Run `atlas --help` or see the atlas docs for setup instructions.

Do not attempt to fix auth issues, read config files, or suggest token values.

## Quick Start

```bash
atlas jira issue describe PROJ-123
```

## Command Tree

```
atlas
├── jira
│   ├── issue describe <KEY>     # Get issue details
│   ├── issue search --jql "..." # Search issues with JQL
│   ├── issue comments <KEY>     # Get issue comments
│   ├── issue types              # List all issue types
│   ├── project list             # List accessible projects
│   └── myself                   # Current authenticated user
└── confluence
    ├── space list               # List accessible spaces
    ├── space describe <KEY>     # Describe space by key
    ├── page describe <ID>       # Page metadata by ID
    ├── page view <ID>           # Page body content (HTML/markdown)
    ├── page search --cql "..."  # Search pages with CQL
    └── page comments <ID>       # Footer comments on a page
```

## Global Flags

| Flag | Default | Purpose |
|------|---------|---------|
| `--output` | `jsonl` | Format: `jsonl`, `text`, `toon` |
| `--timeout` | `30s` | HTTP timeout |
| `--verbose` | `false` | Log HTTP requests to stderr |
| `--raw` | `false` | Emit full API payload (skip compact projection) |

## Decision Tree

```
What do you need?
├─ Jira data → See references/jira.md
│   ├─ Specific issue details → jira issue describe
│   ├─ Search/filter issues → jira issue search (needs JQL)
│   ├─ Issue comments → jira issue comments
│   └─ List projects/types → jira project list, jira issue types
├─ Confluence data → See references/confluence.md
│   ├─ Read page content → confluence page view
│   ├─ Page metadata → confluence page describe
│   ├─ Search pages → confluence page search (needs CQL)
│   ├─ Page comments → confluence page comments
│   └─ List/describe spaces → confluence space list/describe
├─ Auth issues → Tell user to check their atlas config (never touch auth yourself)
└─ Output/error handling → See references/output-and-errors.md
```

## Key Concepts

- **Compact mode** (default): Strips noisy fields from API responses to reduce token count. Use `--raw` for full payloads.
- **JSONL output** (default): Each result is `{"data": ...}` on stdout, one per line. Errors go to stderr as `{"error": {...}}`.
- **Pagination**: Controlled via `--limit` (total items) and `--page-size` (per-request batch). Results stream one-at-a-time.
## In This Reference

| File | Purpose |
|------|---------|
| [jira.md](./references/jira.md) | Jira commands, JQL examples, fields |
| [confluence.md](./references/confluence.md) | Confluence commands, CQL examples |
| [output-and-errors.md](./references/output-and-errors.md) | Output formats, error codes, retryable errors |

## Reading Order

| Task | Files |
|------|-------|
| Query Jira issues | jira.md |
| Read Confluence pages | confluence.md |
| Parse/handle output | output-and-errors.md |
