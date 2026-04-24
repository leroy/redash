// Package manual holds the authoritative, agent-oriented documentation
// for the redash CLI.
//
// The content is compiled into the binary so that `redash manual` always
// matches the installed version. A drift test in the cmd package ensures
// that every Cobra subcommand has a manual entry, preventing docs from
// silently falling out of sync with the command tree.
package manual

import (
	"fmt"
	"sort"
	"strings"
)

// Topic is a single section of the manual. Name is the lookup key used by
// `redash manual --topic NAME`. Command is non-empty when the topic maps
// directly to a Cobra subcommand (this is what the drift test uses).
type Topic struct {
	Name    string
	Command string // cobra subcommand name, empty for meta topics
	Title   string
	Body    string
}

// Topics returns the full ordered list of topics.
func Topics() []Topic {
	return []Topic{
		overview(),
		installAuth(),
		output(),
		errors(),
		workflows(),
		cmdQuery(),
		cmdQueries(),
		cmdDatasources(),
		cmdDashboards(),
		cmdVisualizations(),
		cmdWidgets(),
		cmdUsers(),
		cmdConfig(),
		cmdVersion(),
		cmdManual(),
	}
}

// TopicByName returns a single topic or an error listing available names.
func TopicByName(name string) (Topic, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, t := range Topics() {
		if t.Name == name {
			return t, nil
		}
	}
	names := make([]string, 0, len(Topics()))
	for _, t := range Topics() {
		names = append(names, t.Name)
	}
	sort.Strings(names)
	return Topic{}, fmt.Errorf("unknown topic %q (available: %s)", name, strings.Join(names, ", "))
}

// CommandTopics returns only the topics that document a Cobra subcommand.
// Used by the drift-prevention test.
func CommandTopics() map[string]Topic {
	out := map[string]Topic{}
	for _, t := range Topics() {
		if t.Command != "" {
			out[t.Command] = t
		}
	}
	return out
}

// Markdown renders the full manual (or a single topic) as markdown.
func Markdown(topic string) (string, error) {
	if topic != "" {
		t, err := TopicByName(topic)
		if err != nil {
			return "", err
		}
		return "# " + t.Title + "\n\n" + strings.TrimSpace(t.Body) + "\n", nil
	}
	var b strings.Builder
	b.WriteString("# redash CLI — manual\n\n")
	b.WriteString("_This document is compiled into the binary. It always matches the installed version. " +
		"Pipe it into an agent's context, save it as `AGENTS.md`, or use `--topic NAME` to fetch a single section._\n\n")
	b.WriteString("## Table of contents\n\n")
	for _, t := range Topics() {
		b.WriteString(fmt.Sprintf("- [%s](#%s) — `--topic %s`\n", t.Title, anchor(t.Title), t.Name))
	}
	b.WriteString("\n")
	for _, t := range Topics() {
		b.WriteString("## " + t.Title + "\n\n")
		b.WriteString(strings.TrimSpace(t.Body))
		b.WriteString("\n\n")
	}
	return b.String(), nil
}

// JSON returns a structured catalog of topics; agents that prefer
// deterministic parsing should use this over Markdown.
type JSONCatalog struct {
	CLI     string        `json:"cli"`
	Version string        `json:"version"`
	Topics  []JSONSection `json:"topics"`
}

// JSONSection is a topic in the JSON catalog.
type JSONSection struct {
	Name    string `json:"name"`
	Command string `json:"command,omitempty"`
	Title   string `json:"title"`
	Body    string `json:"body"`
}

// Catalog returns the JSON-friendly structure.
func Catalog(version string) JSONCatalog {
	ts := Topics()
	sections := make([]JSONSection, len(ts))
	for i, t := range ts {
		sections[i] = JSONSection{Name: t.Name, Command: t.Command, Title: t.Title, Body: strings.TrimSpace(t.Body)}
	}
	return JSONCatalog{CLI: "redash", Version: version, Topics: sections}
}

func anchor(title string) string {
	s := strings.ToLower(title)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "`", "")
	s = strings.ReplaceAll(s, "—", "")
	s = strings.ReplaceAll(s, "/", "")
	s = strings.ReplaceAll(s, "(", "")
	s = strings.ReplaceAll(s, ")", "")
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return strings.Trim(s, "-")
}

// ---- topic content ---------------------------------------------------

func overview() Topic {
	return Topic{
		Name: "overview", Title: "Overview",
		Body: `redash is a command-line client for the Redash REST API, designed to be driven
by AI agents as well as humans. It replaces the functionality of a Redash MCP
server with a single static binary, so anything an agent could do via MCP
(list queries, run SQL, inspect schemas, manage users) it can do by shelling
out to ` + "`redash`" + `.

Design tenets relevant to agents:

- **Predictable output**: every command supports ` + "`--format table|json|csv`" + `.
  Default is ` + "`table`" + ` for humans; agents should prefer ` + "`--format json`" + `.
- **Non-interactive by default**: everything except ` + "`redash config init`" + ` runs
  without prompts. Destructive operations require ` + "`--yes`" + `.
- **Exit codes**: 0 on success, 1 on any error. Error text goes to stderr, data
  to stdout, so pipelines stay clean.
- **Self-describing**: this manual is embedded in the binary. Run
  ` + "`redash manual`" + ` for everything, ` + "`redash manual --topic NAME`" + ` for a
  single section, ` + "`redash manual --format json`" + ` for a structured catalog.
- **Actionable errors**: error messages tell the agent what failed, what
  state the remote is in, and (if fixable) what to do next.`,
	}
}

func installAuth() Topic {
	return Topic{
		Name: "auth", Title: "Installation & authentication",
		Body: `### Install

` + "```" + `
go install github.com/leroy/redash@latest
# or download a release binary from https://github.com/leroy/redash/releases
` + "```" + `

### Authenticate

Two options — the CLI accepts either.

**Environment variables** (recommended for agents, CI, and one-offs):

` + "```" + `
export REDASH_URL=https://redash.example.com
export REDASH_API_KEY=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
redash datasources list
` + "```" + `

**Named profiles** (recommended for humans with multiple instances):

` + "```" + `
redash config init --name prod --url https://... --api-key ... --default
redash config use prod
redash datasources list             # uses the default profile
redash --profile staging queries list
` + "```" + `

The config file lives at ` + "`$XDG_CONFIG_HOME/redash-cli/config.yaml`" + `
(fallback: ` + "`~/.config/redash-cli/config.yaml`" + `). It is written with 0600
permissions because it contains API keys.

### Env var reference

| Variable | Purpose |
|---|---|
| ` + "`REDASH_URL`" + ` | Base URL of the Redash instance |
| ` + "`REDASH_API_KEY`" + ` | User- or query-level API key |
| ` + "`REDASH_TIMEOUT`" + ` | Request timeout (Go duration, e.g. ` + "`30s`" + `) |
| ` + "`REDASH_PROFILE`" + ` | Which profile to use (overridden by ` + "`--profile`" + `) |

If both a file profile and env vars are present, env vars override the
profile's ` + "`url`" + ` / ` + "`api_key`" + ` / ` + "`timeout`" + ` fields.`,
	}
}

func output() Topic {
	return Topic{
		Name: "output", Title: "Output formats",
		Body: `Every command accepts ` + "`--format table|json|csv`" + ` (alias ` + "`-o`" + `).
Agents should almost always pass ` + "`--format json`" + ` and parse stdout.

### Query results (` + "`redash query`" + ` and ` + "`redash queries run`" + `)

- ` + "`--format json`" + ` emits an array of row objects with **native types
  preserved** (numbers as numbers, booleans as booleans, nulls as null):

  ` + "```json" + `
  [{"id": 1, "email": "a@b.com", "active": true}, {"id": 2, "email": "c@d.com", "active": false}]
  ` + "```" + `

- ` + "`--format csv`" + ` emits RFC 4180 CSV with a header row.
- ` + "`--format table`" + ` (default) emits a human-readable ANSI table.

### List commands (queries, datasources, dashboards, users)

- ` + "`--format json`" + ` emits ` + "`[{...}, ...]`" + ` of flattened records.
- ` + "`--format table`" + ` / ` + "`--format csv`" + ` emit a fixed column subset.

### ` + "`get`" + ` commands

- ` + "`--format json`" + ` emits the full raw Redash payload for the resource
  (everything the API returned), pretty-printed.
- ` + "`--format table`" + ` emits a two-column ` + "`field` / `value`" + ` table.

### Logging

Informational messages (e.g. row counts, profile in use) go to **stderr** as
` + "`redash: ...`" + `. Use ` + "`--quiet` / `-q`" + ` to suppress. Never parse stderr.`,
	}
}

func errors() Topic {
	return Topic{
		Name: "errors", Title: "Error handling",
		Body: `- Exit code **0** on success, **1** on any error.
- Error text goes to **stderr** prefixed with ` + "`redash: `" + `.
- API errors include the HTTP method, path, status code, and Redash's
  message when present, e.g.:

  ` + "```" + `
  redash: redash api: GET /api/queries/999 -> 404: query not found
  ` + "```" + `

- Context cancellation (Ctrl-C, SIGTERM) aborts in-flight requests cleanly.
- Network/timeout errors look like:

  ` + "```" + `
  redash: GET /api/queries: Get \"...\": dial tcp: connect: connection refused
  ` + "```" + `

### Common failures and how to recover

| Symptom | Cause | Fix |
|---|---|---|
| ` + "`no profile configured`" + ` | No config file, no env vars | ` + "`redash config init`" + ` or set ` + "`REDASH_URL`" + ` + ` + "`REDASH_API_KEY`" + ` |
| ` + "`profile \"X\" not found`" + ` | ` + "`--profile X`" + ` with no matching entry | Run ` + "`redash config show`" + ` to see what's configured |
| ` + "`403 Forbidden`" + ` | API key lacks permission (e.g. user management needs admin key) | Use a key from an account with the required role |
| ` + "`redash job failed: ...`" + ` | SQL error or data source issue | Inspect the quoted message; fix the SQL |
| TLS verification error | Self-signed cert | Add ` + "`--insecure`" + ` or set ` + "`insecure: true`" + ` on the profile |`,
	}
}

func workflows() Topic {
	return Topic{
		Name: "workflows", Title: "Common workflows",
		Body: `### 1. Run an ad-hoc query and consume JSON

` + "```" + `
redash query -d 3 --format json "SELECT id, email FROM users LIMIT 10"
` + "```" + `

### 2. Find a data source, then run against it

` + "```" + `
redash datasources list --format json | jq '.[] | select(.name=="warehouse") | .id'
` + "```" + `

### 3. Execute a saved query with parameters

` + "```" + `
redash queries run 42 \
  --param country=US --param since=2025-01-01 \
  --format json
` + "```" + `

Complex params as JSON (wins over individual ` + "`--param`" + `):

` + "```" + `
redash queries run 42 --params '{"country":"US","ids":[1,2,3]}'
` + "```" + `

### 4. Cache-aware execution

Pass ` + "`--max-age N`" + ` (seconds) to accept a cached result if one exists
younger than N. Avoids re-running expensive queries:

` + "```" + `
redash queries run 42 --max-age 3600     # accept any result from the last hour
redash query -d 3 --max-age 300 \"SELECT ...\"
` + "```" + `

### 5. Inspect a schema before writing SQL

` + "```" + `
redash datasources schema 3 --format json \
  | jq '.schema[] | select(.name==\"orders\") | .columns'
` + "```" + `

### 6. Search saved queries

` + "```" + `
redash queries list --search \"monthly revenue\" --format json
` + "```" + `

### 7. Dry-run: preview a saved query's SQL

` + "```" + `
redash queries get 42 --format json | jq -r .query
` + "```" + `

### 8. Create / update / archive

` + "```" + `
echo \"SELECT count(*) FROM users\" | redash queries create \
  --name \"User count\" --datasource 3 --file - --tag \"ops\"
redash queries update 42 --name \"Renamed\" --publish
redash queries archive 42 --yes
` + "```" + `

### 9. Agent tip: always prefer JSON

For anything a program (or agent) will consume, pass ` + "`--format json`" + `
and parse stdout. Numbers stay numbers; missing values stay ` + "`null`" + `.`,
	}
}

func cmdQuery() Topic {
	return Topic{
		Name: "query", Command: "query", Title: "`redash query` — run an ad-hoc SQL query",
		Body: `Runs a one-off SQL statement against a data source and prints the results.

### Synopsis

` + "```" + `
redash query [SQL] -d|--datasource ID [flags]
` + "```" + `

### Required

- ` + "`-d, --datasource ID`" + ` — numeric data source ID (use ` + "`redash datasources list`" + ` to find it)

### SQL input (pick one)

- Positional argument: ` + "`redash query -d 3 \"SELECT 1\"`" + `
- ` + "`-f, --file PATH`" + `: ` + "`redash query -d 3 -f report.sql`" + `
- Stdin: ` + "`echo 'SELECT 1' | redash query -d 3 -f -`" + `

Combining an argument with ` + "`--file`" + ` is an error.

### Parameters

- ` + "`--param KEY=VALUE`" + ` — repeatable. Values are coerced: ` + "`true`/`false`" + ` to bool,
  integers to int, floats to float, everything else to string.
- ` + "`--params '{...}'`" + ` — full JSON object. Wins over ` + "`--param`" + ` for conflicts.

### Other flags

- ` + "`--max-age N`" + ` — accept a cached result up to N seconds old (0 = always run)
- ` + "`--poll DURATION`" + ` — job polling interval (default 500ms)

### Example (agent-facing)

` + "```" + `
redash query -d 3 --format json --param country=US \
  "SELECT count(*) FROM orders WHERE country = '{{country}}'"
` + "```" + `

### Output

JSON mode: an array of row objects with native types. Table/CSV mode: columns
ordered as returned by Redash. A line like ` + "`redash: fetched 1234 rows in 0.42s`" + `
is written to stderr unless ` + "`--quiet`" + `.`,
	}
}

func cmdQueries() Topic {
	return Topic{
		Name: "queries", Command: "queries", Title: "`redash queries` — manage saved queries",
		Body: `Subcommands for managing saved queries stored in Redash.

### ` + "`redash queries list`" + `

List saved queries with pagination and search.

` + "```" + `
redash queries list [--page N] [--page-size N] [-s|--search TERM] [--tag T ...]
` + "```" + `

JSON output columns (table mode): ` + "`id, name, data_source, tags, user, updated_at`" + `.

### ` + "`redash queries get ID`" + `

Fetch a single saved query. In ` + "`--format json`" + ` mode, returns the full raw
Redash query payload (including ` + "`query`" + ` text, ` + "`options`" + `, ` + "`schedule`" + `).

### ` + "`redash queries run ID`" + `

Execute a saved query and print results. Same parameter/cache flags as
` + "`redash query`" + `:

- ` + "`--param KEY=VALUE`" + ` (repeatable)
- ` + "`--params '{...}'`" + ` (JSON)
- ` + "`--max-age N`" + `
- ` + "`--poll DURATION`" + `

### ` + "`redash queries create [SQL]`" + `

Create a new **draft** query. Required flags: ` + "`--name`" + `, ` + "`--datasource`" + `.
SQL comes from argument, ` + "`--file`" + `, or stdin (same rules as ` + "`redash query`" + `).

### ` + "`redash queries update ID [SQL]`" + `

Partial update. Only fields explicitly passed on the command line are sent:

- ` + "`--name NAME`" + `
- ` + "`--description TEXT`" + `
- ` + "`-d|--datasource ID`" + `
- ` + "`--tag T`" + ` (repeatable; replaces the existing tag set)
- SQL: positional arg after ID, or ` + "`--file`" + `
- ` + "`--publish`" + ` / ` + "`--unpublish`" + ` (mutually exclusive)
- ` + "`--parameters JSON|FILE|-`" + ` — replace the parameter list (see below)

#### Updating parameters

Redash stores query parameters as an array under ` + "`options.parameters`" + `.
` + "`--parameters`" + ` accepts **inline JSON**, a **file path**, or ` + "`-`" + ` for stdin.
The CLI fetches the current options, splices in the new parameter array,
and writes them back — other fields under ` + "`options`" + ` are preserved.

Parameter object shape (one per parameter):

` + "```json" + `
{
  "name": "country",
  "title": "Country",
  "type": "text",
  "value": "US",
  "global": false
}
` + "```" + `

Inline example:

` + "```" + `
redash queries update 42 --parameters '[{"name":"country","title":"Country","type":"text","value":"US","global":false}]'
` + "```" + `

From a file or stdin:

` + "```" + `
redash queries update 42 --parameters ./params.json
jq -n '[{name:"limit",title:"Limit",type:"number",value:100,global:false}]' \
  | redash queries update 42 --parameters -
` + "```" + `

Parameter ` + "`type`" + ` values: ` + "`text`" + `, ` + "`number`" + `, ` + "`enum`" + `, ` + "`query`" + `, ` + "`date`" + `,
` + "`datetime-local`" + `, ` + "`datetime-with-seconds`" + `, ` + "`date-range`" + `,
` + "`datetime-range`" + `, ` + "`datetime-range-with-seconds`" + `.

### ` + "`redash queries archive ID --yes`" + `

Soft-delete. The ` + "`--yes`" + ` flag is required; omitting it errors out as a
safety guard for agents.`,
	}
}

func cmdDatasources() Topic {
	return Topic{
		Name: "datasources", Command: "datasources", Title: "`redash datasources` — inspect data sources",
		Body: `Aliases: ` + "`redash ds`" + `, ` + "`redash datasource`" + `.

### ` + "`redash datasources list`" + `

List all data sources visible to the current API key.

Columns: ` + "`id, name, type, syntax, view_only`" + `.

### ` + "`redash datasources get ID`" + `

Return a single data source with options (credentials redacted by Redash).

### ` + "`redash datasources schema ID`" + `

Fetch the table/column schema. ` + "`--refresh`" + ` requests a server-side refresh
(some Redash versions return a job on refresh; retry if the response looks
empty).

- ` + "`--format table`" + ` / ` + "`--format csv`" + ` flattens to ` + "`table, column, type`" + ` rows.
- ` + "`--format json`" + ` emits the full ` + "`{\"schema\": [{\"name\":..., \"columns\":[...]}]}`" + ` payload.`,
	}
}

func cmdDashboards() Topic {
	return Topic{
		Name: "dashboards", Command: "dashboards", Title: "`redash dashboards` — manage dashboards",
		Body: `Alias: ` + "`redash dashboard`" + `.

Widgets on a dashboard are managed separately via ` + "`redash widgets`" + `
(see ` + "`--topic widgets`" + `).

### ` + "`redash dashboards list`" + `

Paginated list with ` + "`--page`" + `, ` + "`--page-size`" + `, ` + "`-s|--search`" + `.
Columns: ` + "`id, slug, name, tags, user, updated_at`" + `.

### ` + "`redash dashboards get SLUG_OR_ID`" + `

Get a single dashboard by slug. Newer Redash releases also accept numeric IDs
at the same endpoint. In ` + "`--format json`" + ` the full raw payload (including
widgets) is returned.

### ` + "`redash dashboards create --name NAME`" + `

Create a new, empty, **draft** dashboard. The response includes the
assigned ` + "`id`" + ` and ` + "`slug`" + ` — you'll need the ID to attach widgets.

### ` + "`redash dashboards update ID`" + `

Partial update of dashboard metadata. Only passed flags are sent.

- ` + "`--name NAME`" + ` — rename
- ` + "`--tag T`" + ` (repeatable) — replace the tag set
- ` + "`--publish`" + ` / ` + "`--unpublish`" + ` — toggle ` + "`is_draft`" + ` (mutually exclusive)
- ` + "`--enable-filters`" + ` / ` + "`--disable-filters`" + ` — dashboard-level filters

### ` + "`redash dashboards archive ID --yes`" + `

Soft-delete a dashboard. ` + "`--yes`" + ` is required.`,
	}
}

func cmdVisualizations() Topic {
	return Topic{
		Name: "visualizations", Command: "visualizations", Title: "`redash visualizations` — manage query visualizations",
		Body: `Aliases: ` + "`redash viz`" + `, ` + "`redash visualization`" + `.

Every saved query gets an implicit TABLE visualization automatically;
this subcommand tree is for additional visualizations (charts, counters,
pivots, maps, ...).

### ` + "`redash visualizations create`" + `

` + "```" + `
redash visualizations create \
  --query 42 \
  --type CHART \
  --name "Orders over time" \
  --description "Line chart of daily orders" \
  --options ./chart-options.json
` + "```" + `

Required: ` + "`--query`" + `, ` + "`--type`" + `, ` + "`--name`" + `.

Common ` + "`--type`" + ` values: ` + "`TABLE`" + `, ` + "`CHART`" + `, ` + "`COUNTER`" + `, ` + "`PIVOT_TABLE`" + `,
` + "`MAP`" + `, ` + "`WORD_CLOUD`" + `, ` + "`SUNBURST_SEQUENCE`" + `, ` + "`SANKEY`" + `, ` + "`BOXPLOT`" + `,
` + "`CHOROPLETH`" + `, ` + "`DETAILS`" + `.

` + "`--options`" + ` is the type-specific options JSON (inline, file path, or ` + "`-`" + `
for stdin). The shape is complex and differs per type; to discover a
valid shape, create one via the Redash UI then fetch it with:

` + "```" + `
redash queries get 42 --format json | jq '.visualizations'
` + "```" + `

### ` + "`redash visualizations update ID`" + `

Partial update: ` + "`--type`" + `, ` + "`--name`" + `, ` + "`--description`" + `, ` + "`--options`" + `.
At least one must be passed.

### ` + "`redash visualizations delete ID --yes`" + `

Deletes a visualization. The implicit TABLE visualization on a query
cannot be deleted (Redash rejects the request). ` + "`--yes`" + ` is required.`,
	}
}

func cmdWidgets() Topic {
	return Topic{
		Name: "widgets", Command: "widgets", Title: "`redash widgets` — manage dashboard widgets",
		Body: `Alias: ` + "`redash widget`" + `.

A widget is either a **visualization** pinned to a dashboard, or a
**text** (markdown) widget. Create a dashboard first with
` + "`redash dashboards create`" + `, then attach widgets.

### ` + "`redash widgets add`" + `

Exactly one of ` + "`--visualization`" + ` / ` + "`--text`" + ` must be set.

` + "```" + `
# Pin a visualization at grid (col=0, row=0, width=3 cols, height=8 rows)
redash widgets add --dashboard 7 --visualization 99 \
  --col 0 --row 0 --size-x 3 --size-y 8

# Add a markdown text widget
redash widgets add --dashboard 7 --text "## Revenue\nSee chart below." \
  --col 0 --row 8 --size-x 6 --size-y 2
` + "```" + `

Flags:

- ` + "`--dashboard ID`" + ` (required) — dashboard to add to
- ` + "`--visualization ID`" + ` — visualization to pin
- ` + "`--text MARKDOWN`" + ` — markdown for a text widget
- ` + "`--col N`" + ` / ` + "`--row N`" + ` (default 0) — grid position, 0-indexed
- ` + "`--size-x N`" + ` (default 3) — grid width in columns
- ` + "`--size-y N`" + ` (default 8) — grid height in rows
- ` + "`--width N`" + ` (default 1) — legacy width field
- ` + "`--options JSON|FILE|-`" + ` — full options JSON (overrides --col/--row/--size-*)

### ` + "`redash widgets update ID`" + `

Partial update:

- ` + "`--text MARKDOWN`" + ` (text widgets only)
- ` + "`--width N`" + ` (legacy width)
- ` + "`--options JSON|FILE|-`" + ` — full options replacement (usually to reposition)

To move a widget without rewriting its whole options block, fetch the
current options, edit ` + "`options.position`" + `, and pass the result back:

` + "```" + `
redash dashboards get my-dashboard --format json \
  | jq '.widgets[] | select(.id==123) | .options' \
  | jq '.position.col = 3' \
  | redash widgets update 123 --options -
` + "```" + `

### ` + "`redash widgets remove ID --yes`" + `

Aliases: ` + "`rm`" + `, ` + "`delete`" + `. ` + "`--yes`" + ` is required.`,
	}
}

func cmdUsers() Topic {
	return Topic{
		Name: "users", Command: "users", Title: "`redash users` — manage users",
		Body: `Alias: ` + "`redash user`" + `.

These endpoints require an **admin** API key.

### ` + "`redash users list`" + `

` + "```" + `
redash users list [--page N] [--page-size N] [-s|--search TERM] [--disabled]
` + "```" + `

Columns: ` + "`id, name, email, groups, disabled`" + `.

### ` + "`redash users get ID`" + `

Full user record. JSON mode returns the raw payload.

### ` + "`redash users create`" + `

Sends an invitation email.

` + "```" + `
redash users create --name \"Alice\" --email alice@example.com --group 1 --group 2
` + "```" + `

### ` + "`redash users disable ID`" + ` / ` + "`redash users enable ID`" + `

Disable or re-enable a user. No confirmation flag — these are reversible.`,
	}
}

func cmdConfig() Topic {
	return Topic{
		Name: "config", Command: "config", Title: "`redash config` — manage CLI configuration",
		Body: `Manages the YAML config file at ` + "`~/.config/redash-cli/config.yaml`" + `.

### ` + "`redash config path`" + `

Print the resolved config file path.

### ` + "`redash config show`" + `

Print the active profile. API key is redacted (first 3 + last 3 chars).

### ` + "`redash config init`" + `

Create or update a profile. Fully non-interactive when all flags are passed:

` + "```" + `
redash config init --name prod --url https://... --api-key KEY --default
` + "```" + `

Without flags, prompts on stdin. Pass ` + "`--default`" + ` to make the profile the
default (also happens automatically if it's the first profile).

### ` + "`redash config set PROFILE KEY VALUE`" + `

Set a single field: ` + "`url`" + `, ` + "`api_key`" + `, ` + "`timeout`" + `, ` + "`insecure`" + `.

### ` + "`redash config use PROFILE`" + `

Set the default profile.

### ` + "`redash config remove PROFILE`" + `

Delete a profile. Aliases: ` + "`rm`" + `.

### Agent recommendation

Don't write the config file from an agent. Use env vars (` + "`REDASH_URL`" + `,
` + "`REDASH_API_KEY`" + `) — they override file values cleanly and leave no
on-disk residue.`,
	}
}

func cmdVersion() Topic {
	return Topic{
		Name: "version", Command: "version", Title: "`redash version`",
		Body: `Prints the build version, commit hash, and build date in the form:

` + "```" + `
redash v0.1.0 (commit abc1234, built 2026-04-24T17:00:00Z)
` + "```" + `

In development builds the values are ` + "`dev` / `none` / `unknown`" + `.`,
	}
}

func cmdManual() Topic {
	return Topic{
		Name: "manual", Command: "manual", Title: "`redash manual` — this document",
		Body: `Prints the agent-oriented manual for the installed version of the CLI.

` + "```" + `
redash manual                      # full markdown to stdout
redash manual --topic query        # a single section
redash manual --list-topics        # enumerate available topics
redash manual --format json        # structured JSON catalog for machine parsing
` + "```" + `

The manual is compiled into the binary, so it always matches the installed
version — safer than relying on README files that can drift.

### JSON catalog shape

` + "```json" + `
{
  "cli": "redash",
  "version": "0.1.0",
  "topics": [
    {"name": "query", "command": "query", "title": "...", "body": "..."}
  ]
}
` + "```" + `

### Tip for agents

On first use in a new environment, run ` + "`redash manual --format json`" + ` once
and cache the output. That single call is equivalent to an MCP server's
` + "`list_tools` + `describe_tool`" + ` handshake.`,
	}
}
