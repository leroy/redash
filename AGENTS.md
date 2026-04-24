# Agent instructions

**redash is designed to be driven by AI agents.** It replaces the functionality
of a Redash MCP server with a single static CLI binary. If you are an agent
(Cursor, Claude Code, Codex, or similar), this file is for you.

## Start here

The authoritative, version-matched usage guide is compiled into the binary:

```
redash manual
```

Run that **once** before calling any other subcommand. The contents always
match the installed version — safer than relying on this file or the README,
which can drift.

For machine parsing, prefer the JSON catalog:

```
redash manual --format json
```

For a single topic:

```
redash manual --topic query          # or: queries, datasources, dashboards,
                                     #     users, config, auth, output,
                                     #     errors, workflows, overview
redash manual --list-topics          # enumerate available topics
```

## Operating principles

1. **Always pass `--format json`** for any command whose output you will parse.
   Native types are preserved (numbers stay numbers, booleans stay booleans,
   missing values are `null`).
2. **stdout is data, stderr is logs.** Exit code is `0` on success, `1` on
   any error. Error messages are actionable — read them.
3. **Auth via env vars** when running as an agent: `REDASH_URL` and
   `REDASH_API_KEY`. Do not write config files on the user's disk.
4. **Destructive ops require `--yes`.** `redash queries archive ID` errors
   without it. This is a deliberate guard, not a bug.
5. **Cache when appropriate.** `--max-age N` on `query` / `queries run`
   accepts a cached result up to N seconds old. Use it for repeat lookups
   to avoid re-running expensive SQL.

## Minimum viable workflow

```bash
# 1. Understand the tool.
redash manual --format json

# 2. Find the data source.
redash datasources list --format json

# 3. (Optional) inspect the schema.
redash datasources schema <ID> --format json

# 4. Run a query.
redash query -d <ID> --format json "SELECT ..."
```

That's the whole MCP-replacement loop.

## If something goes wrong

- `no profile configured` → set `REDASH_URL` and `REDASH_API_KEY` env vars.
- `403 Forbidden` → the API key doesn't have permission for that endpoint
  (user management requires an admin key).
- `redash job failed: ...` → the SQL errored; fix and retry.
- Any other error is printed verbatim on stderr, prefixed with `redash:`.

See `redash manual --topic errors` for the full table.
