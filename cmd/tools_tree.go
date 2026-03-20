package cmd

func init() {
	meetingsCmd := newToolGroup("meetings", "Meeting retrieval tools")
	meetingsCmd.AddCommand(
		newStubToolCommand(stubToolSpec{
			Use:           "fetch",
			Short:         "Fetch meetings across accounts and date ranges",
			BackendTool:   "fetch_meetings",
			RequiresFlags: "--account ... --from ... --to ...",
		}),
		newStubToolCommand(stubToolSpec{
			Use:         "mine",
			Short:       "Fetch the current user's meetings and details",
			BackendTool: "fetch_my_meetings",
		}),
	)

	peopleCmd := newToolGroup("people", "People and entity search tools")
	peopleCmd.AddCommand(
		newStubToolCommand(stubToolSpec{
			Use:           "find",
			Short:         "Find a person by name, email, or context",
			BackendTool:   "find_person",
			RequiresFlags: "--query ...",
		}),
		newStubToolCommand(stubToolSpec{
			Use:           "lookup",
			Short:         "Look up entities from natural-language user queries",
			BackendTool:   "entity_lookup",
			RequiresFlags: "--query ...",
		}),
		newStubToolCommand(stubToolSpec{
			Use:           "at-company",
			Short:         "Search for people associated with a company",
			BackendTool:   "search_people_at_company",
			RequiresFlags: "--company ... --query ...",
		}),
		newStubToolCommand(stubToolSpec{
			Use:           "details",
			Short:         "Load detailed profile data for a person",
			BackendTool:   "load_person_details",
			RequiresFlags: "--person-id ...",
		}),
	)

	accountsCmd := newToolGroup("accounts", "Account discovery tools")
	accountsCmd.AddCommand(
		newStubToolCommand(stubToolSpec{
			Use:           "find-for-user",
			Short:         "Find accounts associated with a user",
			BackendTool:   "find_accounts_for_user",
			RequiresFlags: "--user ...",
		}),
	)

	interactionsCmd := newToolGroup("interactions", "Interaction history tools")
	interactionsCmd.AddCommand(
		newStubToolCommand(stubToolSpec{
			Use:           "history",
			Short:         "Summarize historical interactions for accounts or topics",
			BackendTool:   "interaction_history",
			RequiresFlags: "--account ... | --topic ...",
		}),
	)

	documentsCmd := newToolGroup("documents", "Document and knowledge retrieval tools")
	documentsCmd.AddCommand(
		newStubToolCommand(stubToolSpec{
			Use:           "fetch",
			Short:         "Retrieve full document contents by document ID",
			BackendTool:   "fetch_whole_document_by_id",
			RequiresFlags: "--document-id ...",
		}),
		newStubToolCommand(stubToolSpec{
			Use:           "facts",
			Short:         "Search extracted facts across text documents",
			BackendTool:   "search_all_text_document_facts",
			RequiresFlags: "--query ...",
		}),
		newStubToolCommand(stubToolSpec{
			Use:           "knowledge",
			Short:         "Fetch matching knowledge documents for a topic",
			BackendTool:   "fetch_knowledge_documents",
			RequiresFlags: "--topic ...",
		}),
		newStubToolCommand(stubToolSpec{
			Use:           "vendor-knowledge",
			Short:         "Search vendor knowledge documents for answers",
			BackendTool:   "search_vendor_knowledge_documents",
			RequiresFlags: "--query ...",
		}),
	)

	datasetsCmd := newToolGroup("datasets", "Structured dataset tools")
	datasetsCmd.AddCommand(
		newStubToolCommand(stubToolSpec{
			Use:           "search",
			Short:         "Search available structured datasets by intent",
			BackendTool:   "search_structured_datasets",
			RequiresFlags: "--query ...",
		}),
		newStubToolCommand(stubToolSpec{
			Use:           "load-from-query",
			Short:         "Load structured data from a user query",
			BackendTool:   "structured_dataset_loader_from_user_query",
			RequiresFlags: "--query ...",
		}),
	)

	slackCmd := newToolGroup("slack", "Slack retrieval tools")
	slackCmd.AddCommand(
		newStubToolCommand(stubToolSpec{
			Use:           "fetch",
			Short:         "Fetch Slack messages related to an account",
			BackendTool:   "fetch_slack_messages",
			RequiresFlags: "--account ...",
		}),
	)

	salesforceCmd := newToolGroup("salesforce", "Salesforce retrieval tools")
	salesforceCmd.AddCommand(
		newStubToolCommand(stubToolSpec{
			Use:           "notes",
			Short:         "Fetch Salesforce notes tied to account activity",
			BackendTool:   "fetch_salesforce_notes",
			RequiresFlags: "--account ...",
		}),
	)

	companiesCmd := newToolGroup("companies", "Company research tools")
	companiesCmd.AddCommand(
		newStubToolCommand(stubToolSpec{
			Use:           "research",
			Short:         "Research companies using external web sources",
			BackendTool:   "company_research_tool",
			RequiresFlags: "--company ...",
		}),
		newStubToolCommand(stubToolSpec{
			Use:           "earnings",
			Short:         "Summarize earnings calls for relevant companies",
			BackendTool:   "summarize_earnings_calls",
			RequiresFlags: "--company ...",
		}),
	)

	webCmd := newToolGroup("web", "External web search tools")
	webCmd.AddCommand(
		newStubToolCommand(stubToolSpec{
			Use:           "search",
			Short:         "Run a Google web search via SerpAPI",
			BackendTool:   "serpapi_google_search",
			RequiresFlags: "--query ...",
		}),
		newStubToolCommand(stubToolSpec{
			Use:           "news",
			Short:         "Run a Google News search via SerpAPI",
			BackendTool:   "serpapi_google_news",
			RequiresFlags: "--query ...",
		}),
	)

	wikiCmd := newToolGroup("wiki", "Synthesized customer wiki article tools")
	wikiCmd.AddCommand(
		newStubToolCommand(stubToolSpec{
			Use:           "search",
			Short:         "Search synthesized wiki articles about customers and topics",
			BackendTool:   "search_wiki_articles",
			RequiresFlags: "--query ...",
		}),
	)

	memoriesCmd := newToolGroup("memories", "User memory tools")
	memoriesCmd.AddCommand(
		newStubToolCommand(stubToolSpec{
			Use:           "search",
			Short:         "Search stored user memories by relevance",
			BackendTool:   "search_user_memories",
			RequiresFlags: "--query ...",
		}),
		newStubToolCommand(stubToolSpec{
			Use:         "core",
			Short:       "Fetch the user's core remembered preferences",
			BackendTool: "fetch_core_user_memories",
		}),
	)

	skillsCmd := newToolGroup("skills", "Skill metadata and asset tools")
	skillsCmd.AddCommand(
		newStubToolCommand(stubToolSpec{
			Use:           "read-asset",
			Short:         "Read a file bundled inside a skill",
			BackendTool:   "read_skill_asset",
			RequiresFlags: "--skill ... --path ...",
		}),
		newStubToolCommand(stubToolSpec{
			Use:           "read-tool",
			Short:         "Read skill metadata and usage instructions",
			BackendTool:   "read_skill_tool",
			RequiresFlags: "--skill ... --tool ...",
		}),
	)

	rawCmd := newToolGroup("raw", "Raw backend tool-name compatibility commands")
	rawCmd.AddCommand(
		newStubToolCommand(stubToolSpec{Use: "fetch_meetings", Short: "Fetch meetings across accounts and date ranges", BackendTool: "fetch_meetings", Hidden: true}),
		newStubToolCommand(stubToolSpec{Use: "entity_lookup", Short: "Look up entities from natural-language user queries", BackendTool: "entity_lookup", Hidden: true}),
		newStubToolCommand(stubToolSpec{Use: "fetch_my_meetings", Short: "Fetch the current user's meetings and details", BackendTool: "fetch_my_meetings", Hidden: true}),
		newStubToolCommand(stubToolSpec{Use: "summarize_earnings_calls", Short: "Summarize earnings calls for relevant companies", BackendTool: "summarize_earnings_calls", Hidden: true}),
		newStubToolCommand(stubToolSpec{Use: "fetch_whole_document_by_id", Short: "Retrieve full document contents by document ID", BackendTool: "fetch_whole_document_by_id", Hidden: true}),
		newStubToolCommand(stubToolSpec{Use: "load_person_details", Short: "Load detailed profile data for a person", BackendTool: "load_person_details", Hidden: true}),
		newStubToolCommand(stubToolSpec{Use: "find_person", Short: "Find a person by name, email, or context", BackendTool: "find_person", Hidden: true}),
		newStubToolCommand(stubToolSpec{Use: "search_people_at_company", Short: "Search for people associated with a company", BackendTool: "search_people_at_company", Hidden: true}),
		newStubToolCommand(stubToolSpec{Use: "interaction_history", Short: "Summarize historical interactions for accounts or topics", BackendTool: "interaction_history", Hidden: true}),
		newStubToolCommand(stubToolSpec{Use: "fetch_slack_messages", Short: "Fetch Slack messages related to an account", BackendTool: "fetch_slack_messages", Hidden: true}),
		newStubToolCommand(stubToolSpec{Use: "fetch_salesforce_notes", Short: "Fetch Salesforce notes tied to account activity", BackendTool: "fetch_salesforce_notes", Hidden: true}),
		newStubToolCommand(stubToolSpec{Use: "search_all_text_document_facts", Short: "Search extracted facts across text documents", BackendTool: "search_all_text_document_facts", Hidden: true}),
		newStubToolCommand(stubToolSpec{Use: "search_wiki_articles", Short: "Search synthesized wiki articles about customers and topics", BackendTool: "search_wiki_articles", Hidden: true}),
		newStubToolCommand(stubToolSpec{Use: "search_structured_datasets", Short: "Search available structured datasets by intent", BackendTool: "search_structured_datasets", Hidden: true}),
		newStubToolCommand(stubToolSpec{Use: "structured_dataset_loader_from_user_query", Short: "Load structured data from a user query", BackendTool: "structured_dataset_loader_from_user_query", Hidden: true}),
		newStubToolCommand(stubToolSpec{Use: "fetch_knowledge_documents", Short: "Fetch matching knowledge documents for a topic", BackendTool: "fetch_knowledge_documents", Hidden: true}),
		newStubToolCommand(stubToolSpec{Use: "search_vendor_knowledge_documents", Short: "Search vendor knowledge documents for answers", BackendTool: "search_vendor_knowledge_documents", Hidden: true}),
		newStubToolCommand(stubToolSpec{Use: "serpapi_google_search", Short: "Run a Google web search via SerpAPI", BackendTool: "serpapi_google_search", Hidden: true}),
		newStubToolCommand(stubToolSpec{Use: "serpapi_google_news", Short: "Run a Google News search via SerpAPI", BackendTool: "serpapi_google_news", Hidden: true}),
		newStubToolCommand(stubToolSpec{Use: "company_research_tool", Short: "Research companies using external web sources", BackendTool: "company_research_tool", Hidden: true}),
		newStubToolCommand(stubToolSpec{Use: "find_accounts_for_user", Short: "Find accounts associated with a user", BackendTool: "find_accounts_for_user", Hidden: true}),
		newStubToolCommand(stubToolSpec{Use: "search_user_memories", Short: "Search stored user memories by relevance", BackendTool: "search_user_memories", Hidden: true}),
		newStubToolCommand(stubToolSpec{Use: "fetch_core_user_memories", Short: "Fetch the user's core remembered preferences", BackendTool: "fetch_core_user_memories", Hidden: true}),
		newStubToolCommand(stubToolSpec{Use: "read_skill_asset", Short: "Read a file bundled inside a skill", BackendTool: "read_skill_asset", Hidden: true}),
		newStubToolCommand(stubToolSpec{Use: "read_skill_tool", Short: "Read skill metadata and usage instructions", BackendTool: "read_skill_tool", Hidden: true}),
	)

	toolsCmd.AddCommand(
		meetingsCmd,
		peopleCmd,
		accountsCmd,
		interactionsCmd,
		documentsCmd,
		datasetsCmd,
		slackCmd,
		salesforceCmd,
		companiesCmd,
		webCmd,
		wikiCmd,
		memoriesCmd,
		skillsCmd,
		rawCmd,
	)
}
