# TOON Output Format Support

**Status:** Removed

This feature has been removed from atlas. The `--output=toon` option and all associated flags (`--toon-indent`, `--toon-delimiter`, `--toon-length-marker`) are no longer supported.

## Rationale

The TOON format support was removed to simplify the codebase and reduce external dependencies. Users who need TOON output should pipe JSONL output through external conversion tools.

## Migration

If you were using TOON output:

```bash
# Before:
atlas jira issue search --output=toon

# After (pipe to external converter):
atlas jira issue search | jq -c . | toon-converter
```
