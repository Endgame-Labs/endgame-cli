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

endgame tools
endgame tools people find
endgame tools documents knowledge
endgame tools wiki search

endgame thread new --prompt "What deals closed this week?"
echo "What deals closed this week?" | endgame thread new
endgame thread continue --thread-id <thread-id> --prompt "What changed since yesterday?"
endgame thread followup <operation-id>
endgame thread env
```

## Auth

Authentication follows the same pattern as `linctl`:

- `ENDGAME_API_KEY` and `ENDGAME_ORG_ID` are the highest-precedence auth source.
- Stored credentials live in `~/.endgame-auth.json`.
- `endgame auth login` prompts for API key + org ID, verifies them against Endgame, and saves them.
- `endgame auth status` verifies the saved or env-provided credentials against the MCP endpoint.

## TODO

- Add WorkOS OAuth-based auth flow so `endgame auth login` can avoid manual API key entry.
- Replace stubbed `endgame tools ...` commands with live API implementations as backend support lands.
