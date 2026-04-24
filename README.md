# redash

A command-line client for the [Redash](https://redash.io) REST API — single static binary, works against any Redash instance (cloud or self-hosted).

Built in Go with `cobra` + `viper`. Outputs pretty tables by default, with `--format json` and `--format csv` for scripting.

## Features

- **Ad-hoc queries** — send SQL to a data source and print results (table, JSON, or CSV).
- **Saved queries** — list, get, run, create, update, archive.
- **Parameters** — pass query parameters via repeatable `--param key=value` flags or a single `--params '{...}'` JSON object.
- **Data sources** — list, get, fetch schema.
- **Dashboards** — list, get.
- **Users** — list, get, create, disable, enable.
- **Multiple Redash instances** — config file with named profiles, switchable via `--profile` or `REDASH_PROFILE`.
- **Safe by default** — `--yes` required for destructive operations; API keys redacted in `config show`.

## Install

### Homebrew (macOS, Linux)

```bash
brew install leroy/tap/redash
```

### One-line install (macOS, Linux)

```bash
curl -sSfL https://raw.githubusercontent.com/leroy/redash/main/install.sh | sh
```

Honours `VERSION` (e.g. `VERSION=v1.0.0`) and `INSTALL_DIR` (default `/usr/local/bin`; uses `sudo` if needed):

```bash
curl -sSfL https://raw.githubusercontent.com/leroy/redash/main/install.sh \
  | VERSION=v1.1.0 INSTALL_DIR=$HOME/.local/bin sh
```

The script detects OS/arch, verifies the SHA-256 against `checksums.txt`, and refuses to install on mismatch.

### Go

```bash
go install github.com/leroy/redash@latest
```

### Pre-built binaries

Download from the [Releases](https://github.com/leroy/redash/releases) page (Linux, macOS, Windows — amd64 and arm64).

### Build locally

```bash
git clone https://github.com/leroy/redash
cd redash
go build -o redash .
```

## Quickstart

```bash
# 1. Create a profile (interactive; prompts for URL + API key).
redash config init --name prod --default

# 2. Verify.
redash config show

# 3. List data sources to find the one you want to query.
redash datasources list

# 4. Run an ad-hoc query.
redash query --datasource 3 "SELECT count(*) FROM users"

# 5. Pipe SQL from a file or stdin.
cat report.sql | redash query -d 3 -f -
redash query -d 3 -f report.sql -o csv > report.csv
```

## Configuration

The config file lives at `$XDG_CONFIG_HOME/redash-cli/config.yaml` (falling back to `~/.config/redash-cli/config.yaml`).

```yaml
default_profile: prod
profiles:
  prod:
    url: https://redash.example.com
    api_key: xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
    timeout: 30s
  staging:
    url: https://staging.redash.example.com
    api_key: yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy
    insecure: true
```

Override any field with environment variables:

| Variable          | Effect                                            |
| ----------------- | ------------------------------------------------- |
| `REDASH_PROFILE`  | Choose which profile to use                       |
| `REDASH_URL`      | Override the resolved profile's URL               |
| `REDASH_API_KEY`  | Override the resolved profile's API key           |
| `REDASH_TIMEOUT`  | Override the request timeout (Go duration string) |

The API key is a user-level key or a per-query key from the Redash UI. For admin-only endpoints (user management, creating data sources, etc.) you'll need an admin user's key.

## Command reference

Every command accepts `--format table|json|csv` (default `table`), `--profile NAME`, `--config PATH`, `--timeout DURATION`, and `--insecure`.

### Ad-hoc queries — `redash query`

```bash
redash query --datasource 3 "SELECT 1"
redash query -d 3 -f query.sql --param country=US --param limit=100
redash query -d 3 "SELECT * FROM orders WHERE country = '{{country}}'" \
  --params '{"country":"US"}' \
  --max-age 300 \
  --format csv > orders.csv
```

- `--datasource, -d` — data source ID (required)
- `--file, -f` — read SQL from file (or `-` for stdin)
- `--param k=v` — repeatable query parameter (values auto-coerced to int/float/bool/string)
- `--params '{...}'` — full JSON parameters object (takes precedence over `--param`)
- `--max-age SECS` — accept a cached result this old (0 = always execute)
- `--poll DURATION` — job polling interval (default 500ms)

### Saved queries — `redash queries`

```bash
redash queries list --search "active users" --tag revenue
redash queries get 42
redash queries run 42 --param country=US
redash queries create --name "MRR" --datasource 3 -f mrr.sql --tag revenue
redash queries update 42 --name "MRR (v2)" --publish
redash queries archive 42 --yes
```

### Data sources — `redash datasources`

```bash
redash datasources list
redash datasources get 3
redash datasources schema 3
redash datasources schema 3 --refresh --format json
```

### Dashboards — `redash dashboards`

```bash
redash dashboards list --search revenue
redash dashboards get my-dashboard-slug
```

### Users — `redash users`

```bash
redash users list
redash users get 7
redash users create --name "Alice" --email alice@example.com --group 1
redash users disable 7
redash users enable 7
```

### Config — `redash config`

```bash
redash config path
redash config show
redash config init --name staging --url https://... --api-key ...
redash config set prod timeout 60s
redash config use staging
redash config remove old-profile
```

## For AI agents

`redash` is designed to be driven by AI agents as a drop-in replacement for a
Redash MCP server. The authoritative usage guide is **compiled into the binary**
and is always version-matched:

```bash
redash manual                      # full agent-oriented markdown
redash manual --format json        # structured catalog for machine parsing
redash manual --topic query        # single section
redash manual --list-topics        # enumerate available topics
```

Run `redash manual --format json` once at startup — it's equivalent to an
MCP server's `list_tools` + `describe_tool` handshake.

A short [`AGENTS.md`](./AGENTS.md) at the repo root is auto-discovered by
Cursor, Claude Code, and other coding agents; it points to the same command.

Drift is prevented by a unit test: every subcommand must have a manual entry,
or CI fails.

## Output formats

- **`table`** (default) — ANSI-styled table with header, good for terminals.
- **`json`** — for query results, an array of row objects preserving native types (numbers as numbers, bools as bools). For `get` commands, the full raw Redash response.
- **`csv`** — RFC 4180 CSV with header row. Use for `> file.csv` pipelines.

## Exit codes

- `0` — success
- `1` — any error (config, API, network, user error). Error message is written to stderr.

## Development

```bash
# Run the test suite.
go test -race ./...

# Vet.
go vet ./...

# Build a dev binary.
go build -o redash .
./redash --help

# Try a release build locally (requires goreleaser).
goreleaser release --snapshot --clean
```

### Layout

```
.
├── main.go               # thin entry point
├── cmd/                  # cobra commands
│   ├── root.go
│   ├── query.go          # ad-hoc query
│   ├── queries.go        # saved queries CRUD + execute
│   ├── datasources.go
│   ├── dashboards.go
│   ├── users.go
│   ├── config.go
│   └── version.go
└── internal/
    ├── client/           # Redash HTTP client
    ├── config/           # config file + env overrides
    └── output/           # table/json/csv formatters
```

## License

MIT — see [LICENSE](./LICENSE).
