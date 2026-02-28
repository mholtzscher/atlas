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

# Run example command
atlas example

# Run with verbose output
atlas --verbose example
```

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
