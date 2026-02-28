# atlas

Agent first CLI for Atlassian products

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

# Use global defaults from config file
atlas --config atlas.json jira issue search --jql 'project = ABC'
```

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

Precedence is `flags > env > config`.

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
