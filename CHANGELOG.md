# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0](https://github.com/mholtzscher/atlas/compare/v0.1.1...v0.2.0) (2026-03-01)


### ⚠ BREAKING CHANGES

* The --no-color global flag has been removed. Users previously relying on this flag to disable colored output will need to update their scripts and configurations.
* **confluence:** CLI flag interface changed. Users must migrate from:   --include-labels --include-properties to:   --include labels --include properties or use --include all to enable all fields.
* **cli:** --jql and --cql flags removed, use --query instead

### Features

* **cli:** consolidate confluence include flags into single --include flag ([8186ecf](https://github.com/mholtzscher/atlas/commit/8186ecfb6c7edd67fb92fc57e93a173a946d59a2))
* **cli:** unify search flags to --query ([#11](https://github.com/mholtzscher/atlas/issues/11)) ([d6e4854](https://github.com/mholtzscher/atlas/commit/d6e48545a048fa732313bd412edaff6eaeb650f4))
* **operations:** add pagination metadata support to list operations ([#13](https://github.com/mholtzscher/atlas/issues/13)) ([4d0b8bf](https://github.com/mholtzscher/atlas/commit/4d0b8bf5a18d908a8d20b0c6bf36a80e41924669))
* remove --no-color flag and related options ([e1a1a07](https://github.com/mholtzscher/atlas/commit/e1a1a07025f58d0524f865eeafa0e21904fe9ccc))


### Code Refactoring

* **confluence:** consolidate include flags into single --include option ([b4e97c0](https://github.com/mholtzscher/atlas/commit/b4e97c05290edee4f68954cd03d5a489b3ed3a5d))

## [0.1.1](https://github.com/mholtzscher/atlas/compare/v0.1.0...v0.1.1) (2026-03-01)


### Bug Fixes

* **confluence:** correct page search API endpoint path ([70f0eb7](https://github.com/mholtzscher/atlas/commit/70f0eb7669a75c7d20e2b16bdb8dab0332df3ffd))

## 0.1.0 (2026-03-01)


### ⚠ BREAKING CHANGES

* The `--output=toon` option and all `--toon-*` flags are no longer supported. Users should pipe JSONL output through external converters if they need TOON format.
* rename 'get' commands to 'describe' for jira and confluence
    - jira issue get -> jira issue describe
    - confluence space get -> confluence space describe
    - confluence page get -> confluence page describe

### Features

* add agent-first JSONL output, Jira/Confluence read operations, and config support ([75d7b37](https://github.com/mholtzscher/atlas/commit/75d7b3753381202cb8ce5fa84a059dc775db96dc))
* compact output by default with --raw flag for full payloads ([20a9c05](https://github.com/mholtzscher/atlas/commit/20a9c052c1970eee5efb8deab5eab09c36d22c79))
* **confluence:** add HTML cleaning to remove Confluence-specific attributes ([eb01dcd](https://github.com/mholtzscher/atlas/commit/eb01dcddfda7214c8d93ec3e70857010bfead101))
* **confluence:** add markdown output format to page view command ([669e255](https://github.com/mholtzscher/atlas/commit/669e255d11ec92a50dbb27ca711b0a116c1b2dea))
* **confluence:** add space operations and page comments support ([f0ee237](https://github.com/mholtzscher/atlas/commit/f0ee237f2dfaaa37a0dd764a3abb1b7584ab1b54))
* **confluence:** add support for multiple body formats and change default to view ([30981c1](https://github.com/mholtzscher/atlas/commit/30981c1b9d52c2f564d501c973b7ca9c8a2200b5))
* **confluence:** simplify comment body output with plain text ([b8b44a8](https://github.com/mholtzscher/atlas/commit/b8b44a83c5408d9eee302d06a970b5df02de6bba))
* **jira:** add project list, issue comments, types, and myself commands ([7e58265](https://github.com/mholtzscher/atlas/commit/7e58265739e8e3818504a4cf0f95a3102a6cf69f))
* **jira:** convert ADF comment bodies to plain text ([96d467c](https://github.com/mholtzscher/atlas/commit/96d467c672b8498601a811fdc29a43b4f6f7688a))
* **output:** add TOON output format with configurable options ([e1934da](https://github.com/mholtzscher/atlas/commit/e1934da15f965101cb5de8f52e108b9494322dd0))


### Bug Fixes

* improve cli error messages and test commands ([ccfb852](https://github.com/mholtzscher/atlas/commit/ccfb852f76b1b1bcaa5b84e47c2d5d10c9159741))


### Miscellaneous Chores

* release 0.1.0 ([efe406f](https://github.com/mholtzscher/atlas/commit/efe406ffc9e9371ceff0a86e350fed085e3ac646))


### Code Refactoring

* remove TOON output format ([c0f73a2](https://github.com/mholtzscher/atlas/commit/c0f73a230ee1e0f277952bf87cb8dadd719dcf2f))

## [0.0.0](https://github.com/mholtzscher/atlas/releases/tag/v0.1.0) (YYYY-MM-DD)

### Features

- Initial release
- Basic CLI structure with urfave/cli/v3
- Example subcommand
- Nix flake support
- GitHub Actions CI/CD
