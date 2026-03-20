# endgame-cli

Minimal Go CLI for the Endgame.io MCP bridge.

## Layout

- `main.go` is the thin entrypoint.
- `cmd/` contains Cobra root commands, supercommands, and subcommands.
- `pkg/auth/` contains credential persistence and auth verification.
- `pkg/endgame/` contains the MCP client and request/polling logic.

## Commands

```bash
endgame auth
endgame auth login
endgame auth status
endgame auth logout
endgame whoami

endgame threads tools
endgame threads prompt "What deals closed this week?"
endgame threads followup <operation-id>
endgame threads ask "What deals closed this week?"
endgame threads env
```

## Auth

Authentication follows the same pattern as `linctl`:

- `ENDGAME_API_KEY` and `ENDGAME_ORG_ID` are the highest-precedence auth source.
- Stored credentials live in `~/.endgame-auth.json`.
- `endgame auth login` prompts for API key + org ID, verifies them against Endgame, and saves them.
- `endgame auth status` verifies the saved or env-provided credentials against the MCP endpoint.

## TODO

- Add WorkOS OAuth-based auth flow so `endgame auth login` can avoid manual API key entry.
- Add a top-level `endgame tools` supercommand to wrap Endgame tools like Entity Loader and Fact Search.
