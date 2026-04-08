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
endgame auth
endgame auth login
endgame auth status
```

`endgame auth` and `endgame auth login` prompt for a login method when run interactively:

- browser login on this machine
- device-code login for a remote or separate browser session

You can also select the mode explicitly:

```bash
endgame auth login --mode browser
endgame auth login --mode device
```

If the CLI is running on a remote server and you use browser mode, forward the callback port first:

```bash
ssh -L 8788:127.0.0.1:8788 <server>
```

Then run `endgame auth login --mode browser --callback-port 8788`.

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

Authentication uses OAuth against the public Endgame auth server.

- `endgame auth` prompts for browser login or device-code login when run in an interactive terminal
- `endgame auth login --mode browser|device` selects the login flow explicitly
- browser login discovers metadata from `https://app.endgame.io/.well-known/oauth-authorization-server`
- the CLI dynamically registers a public OAuth client
- browser login completes with authorization code + PKCE on a localhost callback
- device login uses the production Connect device flow on `https://login.endgame.io/oauth2/device_authorization`
- device login requests `openid profile email offline_access`
- refreshable OAuth tokens are stored in `~/.endgame-auth.json`
- `endgame auth status` verifies the saved token set against MCP

For repeated CLI use, including agent-heavy workflows, the CLI does not refresh on every command and does not run a background daemon:

- it reuses the cached access token until it is within a 2 minute refresh window
- if `expires_in` is missing, it derives expiry from the JWT `exp` claim
- if the token is near expiry, it refreshes once before the request
- refresh token rotation is persisted back to `~/.endgame-auth.json`
- if MCP returns `401 invalid_token`, it forces one refresh and retries once
- login fails immediately if no refresh token is returned, instead of saving a token set that will break later

This was soak-tested with repeated authenticated CLI queries over 30 minutes, including crossing the original access-token expiry boundary and observing successful token rotation on disk.

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
find_person
find_relevant_documents
get_document
get_interaction_history
get_person_details
get_user_preferences
news_search
query_data
query_dataset
research_company
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
summarize_earnings_calls
web_search
```

Tool descriptions below are based on the live MCP `tools/list` response from the Endgame server.

People:

- `find_person`: Look up a specific person by name, email address, Salesforce ID, LinkedIn profile ID, or Endgame Person ID. Accepts an optional `account_id` to scope the search to a specific account.
- `get_person_details`: Load detailed profile data for one or more people by Endgame person ID. Accepts an optional `account_id` to scope results to a specific account.
- `search_people`: Search for people at a target company using multiple data sources with filtering and pagination. Accepts an optional `account_id` to scope results to a specific account.

Documents and knowledge:

- `fetch_knowledge_documents`: List knowledge documents with metadata, previews, and pagination support.
- `find_relevant_documents`: Automatically find vendor knowledge documents relevant to a raw user message.
- `get_document`: Fetch the full content of a specific document by ID.
- `search_document_insights`: Search extracted facts and insights across all document types. Accepts an optional `account_id` to scope results to a specific account.
- `search_knowledge_articles`: Search synthesized knowledge articles about customers and topics. Accepts an optional `account_id` to scope results to a specific account.
- `search_vendor_documents`: Search vendor knowledge documents using semantic and keyword matching.

Meetings, interactions, and notes:

- `get_interaction_history`: Retrieve chronological meeting and email history across one or more accounts, or for a specific person. Accepts an optional `account_id` to scope results to a specific account.
- `search_meetings`: Search meetings, calls, and emails across the organization. Accepts an optional `account_id` to scope results to a specific account.
- `search_my_meetings`: Search meetings, calls, and emails where the current user is a participant.
- `search_salesforce_notes`: Fetch Salesforce notes and activity history. Accepts an optional `account_id` to scope results to a specific account.
- `search_slack_messages`: Fetch Slack messages. Accepts an optional `account_id` to scope results to a specific account.

Preferences and memory:

- `get_user_preferences`: Retrieve the current user's core preferences and standing instructions.
- `search_user_preferences`: Search the current user's contextual memories and situational preferences semantically.

Research and external search:

- `news_search`: Search Google News for editorial coverage within a date range.
- `research_company`: Research one or more companies and return structured answers with quotes and source URLs.
- `web_search`: Search the public web via Google for pages, snippets, sources, and published dates.

Analytics and datasets:

- `query_data`: Convert natural-language questions into SQL against the organization's warehouse data.
- `query_dataset`: Query uploaded structured datasets with natural language.
- `search_datasets`: Discover which structured datasets are available in the organization.

Earnings:

- `summarize_earnings_calls`: Retrieve and summarize earnings call transcripts across one or more accounts. Accepts an optional `account_id` to scope results to a specific account.

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
