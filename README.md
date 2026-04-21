# endgame-cli

CLI for Endgame's public MCP `v1` endpoint.

## Install

Build locally:

```bash
go build -o endgame .
```

Install from source:

```bash
go build -o ~/go/bin/endgame github.com/Endgame-Labs/endgame-cli
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
endgame tools fetch_knowledge_documents --json '{"account_ids":["VENDOR"],"filename":"battlecard"}'
endgame tools search_vendor_documents --json '{"search_query":"security questionnaire","max_results":5}'
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
- auth metadata is discovered from `https://login.endgame.io/.well-known/openid-configuration`
- the CLI uses a public OAuth client ID bundled with the binary
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

Tool subcommands are loaded from a locally cached snapshot of live MCP `tools/list` data.

- `endgame auth login` refreshes the tools cache after successful auth
- `endgame tools sync` refreshes the cache on demand
- `endgame tools --help` shows the currently cached live tools
- `endgame tools --help` is hierarchical: a concise command index first, then full per-tool documentation
- after command execution, the CLI may start a background cache refresh if the last sync is older than 12 hours

Cache location:

- `~/.endgame/tools-cache.json`

Invoke any tool using its runtime-discovered name:

```bash
endgame tools <mcp_tool_name> --json '{"...":"..."}'
```

`--json` may also be omitted when piping a JSON object on stdin.

Examples:

```bash
endgame tools search_meetings --json '{"account_ids":["001..."],"start_date":"2026-04-01","end_date":"2026-04-30"}'
endgame tools query_data --json '{"messages":[{"user_message":"show top 10 accounts by ARR"}]}'
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
- `pkg/toolscache/` contains local tools cache persistence and staleness checks
- `pkg/buildinfo/` contains build metadata used by `endgame version`
