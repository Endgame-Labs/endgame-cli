# endgame-cli

CLI for Endgame's public MCP `v1` endpoint.

## Install

Build locally:

```bash
go build -o endgame .
```

Install from source:

```bash
go install github.com/Endgame-Labs/endgame-cli@latest
```

Check the build metadata:

```bash
endgame version
```

## Quickstart

Authenticate:

```bash
endgame auth login
endgame auth status
```

List available MCP tools:

```bash
endgame tools --help
```

Run a few commands:

```bash
endgame tools find_person --json '{"search_query":"Sarah Chen"}'
endgame tools fetch_knowledge_documents --account-id VENDOR --filename battlecard
endgame tools search_vendor_documents --query "security questionnaire" --max-results 5
endgame tools web_search --json '{"query":"Acme pricing page"}'
```

Most tools take their MCP input object via `--json` or piped stdin:

```bash
echo '{"query":"pricing preferences"}' | endgame tools search_user_preferences
```

## Auth

Authentication is browser-based OAuth against the public Endgame auth server.

- `endgame auth login` discovers metadata from `https://app.endgame.io/.well-known/oauth-authorization-server`
- the CLI dynamically registers a public OAuth client
- login completes with authorization code + PKCE on a localhost callback
- refreshable OAuth tokens are stored in `~/.endgame-auth.json`
- `endgame auth status` verifies the saved token set against MCP

There are no required auth environment variables. `endgame thread env` reports `ENDGAME_TIMEOUT_SECONDS` when set.

## Commands

Top-level commands:

```text
auth
thread
tools
version
whoami
```

### Threads

```bash
endgame thread new --prompt "What deals closed this week?"
echo "What deals closed this week?" | endgame thread new
endgame thread continue --thread-id <thread-id> --prompt "What changed since yesterday?"
endgame thread followup <operation-id>
endgame thread env
```

### Tools

The CLI exposes MCP tools directly as:

```bash
endgame tools <mcp_tool_name>
```

Current raw MCP tool names:

```text
fetch_knowledge_documents
find_account_person
find_accounts
find_person
find_relevant_documents
get_account_interaction_history
get_account_person_details
get_document
get_interaction_history
get_person_details
get_user_preferences
news_search
query_data
query_dataset
research_company
search_account_document_insights
search_account_knowledge_articles
search_account_knowledge_documents
search_account_meetings
search_account_people
search_account_salesforce_notes
search_account_slack_messages
search_datasets
search_document_insights
search_knowledge_articles
search_meetings
search_my_meetings
search_people
search_salesforce_notes
search_slack_messages
search_user_preferences
search_vendor_documents
summarize_account_earnings_calls
summarize_earnings_calls
web_search
```

Invocation styles:

- First-class flagged commands:
  - `fetch_knowledge_documents`
  - `get_document`
  - `search_vendor_documents`
- Generic JSON commands:
  - pass the MCP input object with `--json '{...}'`
  - or pipe the JSON object on stdin

Example generic call:

```bash
endgame tools search_meetings --json '{"account_ids":["001..."],"start_date":"2026-04-01","end_date":"2026-04-30"}'
```

First-class flagged commands:

### `fetch_knowledge_documents`

```bash
endgame tools fetch_knowledge_documents --account-id VENDOR --filename battlecard
```

Flags:

- `--account-id` repeatable
- `--after-date`
- `--before-date`
- `--content-type`
- `--filename`
- `--page-offset`

### `get_document`

```bash
endgame tools get_document --account-id VENDOR --document-id abc123def
```

Flags:

- `--account-id` required
- `--document-id` required
- `--document-type` defaults to `knowledge_document`

### `search_vendor_documents`

```bash
endgame tools search_vendor_documents --query "security questionnaire" --max-results 5
```

Flags:

- `--query` required
- `--max-results` defaults to `5`

Generic examples:

### `find_person`

```bash
endgame tools find_person --json '{"search_query":"Sarah Chen"}'
```

### `search_people`

```bash
endgame tools search_people --json '{"account_id":"001...","areas":["crm_contacts"]}'
```

### `search_meetings`

```bash
endgame tools search_meetings --json '{"account_ids":["001..."],"start_date":"2026-04-01","end_date":"2026-04-30"}'
```

### `query_data`

```bash
endgame tools query_data --json '{"messages":[{"user_message":"show top 10 accounts by ARR"}]}'
```

### `search_document_insights`

```bash
endgame tools search_document_insights --json '{"search_sentences":["security questionnaire","redlines"]}'
```

### `search_user_preferences`

```bash
echo '{"query":"pricing preferences"}' | endgame tools search_user_preferences
```

## Versioning

`endgame version` prints the CLI version, commit, and build date.

For tagged builds, inject build metadata with `-ldflags`:

```bash
go build -ldflags "\
  -X github.com/Endgame-Labs/endgame-cli/pkg/buildinfo.Version=v0.1.0 \
  -X github.com/Endgame-Labs/endgame-cli/pkg/buildinfo.Commit=$(git rev-parse --short HEAD) \
  -X github.com/Endgame-Labs/endgame-cli/pkg/buildinfo.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
```

## Layout

- `main.go` is the entrypoint
- `cmd/` contains Cobra commands
- `pkg/auth/` contains OAuth login, token persistence, and refresh
- `pkg/endgame/` contains the MCP transport and tool invocation logic
- `pkg/buildinfo/` contains build metadata used by `endgame version`
