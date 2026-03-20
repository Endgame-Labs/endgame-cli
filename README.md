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

## Tools

The current grouped tool tree is stubbed in the CLI and ready for backend implementation:

```bash
endgame tools meetings fetch
endgame tools meetings mine

endgame tools people find
endgame tools people lookup
endgame tools people at-company
endgame tools people details

endgame tools accounts find-for-user

endgame tools interactions history

endgame tools documents fetch
endgame tools documents facts
endgame tools documents knowledge
endgame tools documents vendor-knowledge

endgame tools datasets search
endgame tools datasets load-from-query

endgame tools slack fetch
endgame tools salesforce notes

endgame tools companies research
endgame tools companies earnings

endgame tools web search
endgame tools web news

endgame tools wiki search

endgame tools memories search
endgame tools memories core

endgame tools skills read-asset
endgame tools skills read-tool
```

Backend tool mapping:

- `endgame tools meetings fetch` -> `fetch_meetings`
- `endgame tools meetings mine` -> `fetch_my_meetings`
- `endgame tools people find` -> `find_person`
- `endgame tools people lookup` -> `entity_lookup`
- `endgame tools people at-company` -> `search_people_at_company`
- `endgame tools people details` -> `load_person_details`
- `endgame tools accounts find-for-user` -> `find_accounts_for_user`
- `endgame tools interactions history` -> `interaction_history`
- `endgame tools documents fetch` -> `fetch_whole_document_by_id`
- `endgame tools documents facts` -> `search_all_text_document_facts`
- `endgame tools documents knowledge` -> `fetch_knowledge_documents`
- `endgame tools documents vendor-knowledge` -> `search_vendor_knowledge_documents`
- `endgame tools datasets search` -> `search_structured_datasets`
- `endgame tools datasets load-from-query` -> `structured_dataset_loader_from_user_query`
- `endgame tools slack fetch` -> `fetch_slack_messages`
- `endgame tools salesforce notes` -> `fetch_salesforce_notes`
- `endgame tools companies research` -> `company_research_tool`
- `endgame tools companies earnings` -> `summarize_earnings_calls`
- `endgame tools web search` -> `serpapi_google_search`
- `endgame tools web news` -> `serpapi_google_news`
- `endgame tools wiki search` -> `search_wiki_articles`
- `endgame tools memories search` -> `search_user_memories`
- `endgame tools memories core` -> `fetch_core_user_memories`
- `endgame tools skills read-asset` -> `read_skill_asset`
- `endgame tools skills read-tool` -> `read_skill_tool`

## Auth

Authentication follows the same pattern as `linctl`:

- `ENDGAME_API_KEY` and `ENDGAME_ORG_ID` are the highest-precedence auth source.
- Stored credentials live in `~/.endgame-auth.json`.
- `endgame auth login` prompts for API key + org ID, verifies them against Endgame, and saves them.
- `endgame auth status` verifies the saved or env-provided credentials against the MCP endpoint.

## TODO

- Add WorkOS OAuth-based auth flow so `endgame auth login` can avoid manual API key entry.
- Replace stubbed `endgame tools ...` commands with live API implementations as backend support lands.
