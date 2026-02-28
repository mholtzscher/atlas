# atlas

Agent-first CLI for Atlassian products (Jira and Confluence). Designed for AI agents and automation workflows with machine-readable output formats.

## Purpose

Atlas provides programmatic access to Atlassian Cloud APIs with output formats optimized for AI consumption. All commands emit structured data (JSONL by default) with operation identifiers, making it easy for agents to parse and act on Atlassian data.

## Functionality

### Jira Operations

- **Issue search**: Query issues with JQL (`atlas jira issue search --jql 'project = ABC'`)
- **Issue describe**: Retrieve specific issues by key with compact or raw output
- **Issue comments**: Get comments on an issue
- **Issue types**: List all available issue types
- **Project list**: List all accessible projects
- **Myself**: Get current user information

### Confluence Operations

- **Space list/describe**: List accessible spaces or describe specific space by key
- **Page describe**: Get page metadata by ID
- **Page view**: Display page body content (formatted HTML)
- **Page search**: Search pages with CQL
- **Page comments**: Get footer comments on a page

### Output Formats

- **JSONL** (default): Newline-delimited JSON with operation identifiers
- **Text**: Plain text tab-separated output
- **TOON**: Token-Oriented Object Notation for compact structured output

### Configuration

Config file format is JSON and uses global flag names as keys:

```json
{
  "output": "jsonl",
  "site": "https://acme.atlassian.net",
  "auth": "pat",
  "email": "agent@example.com",
  "api-token": "<token>",
  "timeout": "30s",
  "verbose": false,
  "no-color": false
}
```

**Precedence**: flags > env > config

Config locations (in order of precedence):
- `./atlas.json` (local project config)
- `~/.config/atlas/atlas.json` (XDG config directory)

## Installation

### Using Nix

```bash
nix run github:mholtzscher/atlas
```

### Using Homebrew

```bash
brew tap mholtzscher/tap
brew install atlas
```

### From Source

```bash
git clone https://github.com/mholtzscher/atlas.git
cd atlas
nix build
```

## Usage

```bash
# Show help
atlas --help

# List machine-readable operation IDs
atlas meta ops

# Search Jira (JSONL by default)
atlas jira issue search --jql 'project = ABC ORDER BY updated DESC'

# Describe specific issue with compact output
atlas jira issue describe ABC-123

# Describe issue with full raw payload
atlas jira issue describe ABC-123 --raw

# List Confluence spaces
atlas confluence space list

# Search Confluence pages
atlas confluence page search --cql 'space = DEV'

# View page content (formatted HTML output)
atlas confluence page view 123456789
```

## Content Defaults

| Setting | Default Value | Description |
|---------|--------------|-------------|
| `output` | `jsonl` | Output format (jsonl, text, toon) |
| `auth` | `pat` | Authentication mode (pat or oauth) |
| `timeout` | `30s` | HTTP request timeout |
| `toon-indent` | `2` | TOON format indentation spaces |
| `toon-delimiter` | `comma` | TOON format delimiter (comma, tab, pipe) |
| `toon-length-marker` | `false` | Add # prefix to TOON array lengths |

**Pagination Defaults:**
- Jira issue search: 50 results per page, 50 max results
- Confluence space/page search: 25 results per page, 25 max results

## Development

This project uses Nix for reproducible development environments.

```bash
# Enter development shell
nix develop

# Or use direnv
direnv allow

# Run checks
just check

# Build
just build

# Run tests
just test
```

## License

MIT
