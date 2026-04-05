# endgame-cli Agent Notes

This repository is safe to use from agent-driven workflows, but there are a few auth behaviors worth knowing.

## Auth Behavior

- Tokens are stored in `~/.endgame-auth.json`.
- `endgame auth` prompts for `browser` or `device` mode when run interactively.
- `endgame auth login --mode browser|device` selects the mode explicitly.
- Device auth uses the production Connect device flow on `https://login.endgame.io/oauth2/device_authorization` and `https://login.endgame.io/oauth2/token`.
- Device auth explicitly requests `openid profile email offline_access`.
- Login fails fast if the provider does not return a refresh token.

## Token Reuse

- The CLI is not a daemon and does not refresh in the background.
- On each authenticated invocation, the CLI reuses the cached access token if it still has enough lifetime left.
- The refresh window is 2 minutes before expiry.
- If `expires_in` is missing, the CLI derives expiry from the JWT `exp` claim.
- If MCP returns `401 invalid_token`, the CLI forces one refresh and retries once.

This is designed for agent-heavy workloads where the CLI may be called many times in a short burst:

- no refresh on every command
- no background polling
- one refresh only when the token is near expiry or MCP rejects it as stale

## Operational Notes

- If users appear to get logged out shortly after device login, the first thing to check is whether the saved token file contains a `refresh_token`.
- If no refresh token is present, the auth provider did not grant one and the CLI will now fail login immediately instead of saving a broken token set.
