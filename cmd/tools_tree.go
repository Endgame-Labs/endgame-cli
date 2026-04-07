package cmd

func init() {
	toolsCmd.AddCommand(
		newDocumentsKnowledgeCommandWithName("fetch_knowledge_documents"),
		newDocumentsFetchCommandWithName("get_document"),
		newDocumentsVendorKnowledgeCommandWithName("search_vendor_documents"),

		// --- People ---
		newStubToolCommand(stubToolSpec{Use: "find_person", Short: "Find a specific person by name, email, or ID", BackendTool: "find_person", RequiresFlags: `{"search_query":"Sarah Chen"}`}),
		newStubToolCommand(stubToolSpec{Use: "get_person_details", Short: "Load detailed profile data for one or more people", BackendTool: "get_person_details", RequiresFlags: `{"person_ids":"guid-1,guid-2"}`}),
		newStubToolCommand(stubToolSpec{Use: "search_people", Short: "Browse and filter people at an account", BackendTool: "search_people", RequiresFlags: `{"account_id":"001...","areas":["crm_contacts"]}`}),

		// --- Interaction history (consolidated — replaces get_account_interaction_history) ---
		newStubToolCommand(stubToolSpec{Use: "get_interaction_history", Short: "Get interaction history (optionally scoped to an account)", BackendTool: "get_interaction_history", RequiresFlags: `{"account_ids":["001..."],"output":"summary"}`}),

		// --- Meetings (consolidated — replaces search_account_meetings) ---
		newStubToolCommand(stubToolSpec{Use: "search_meetings", Short: "Search meetings across the organization", BackendTool: "search_meetings", RequiresFlags: `{"account_ids":["001..."],"start_date":"2026-04-01","end_date":"2026-04-30"}`}),
		newStubToolCommand(stubToolSpec{Use: "search_my_meetings", Short: "Search the current user's meetings", BackendTool: "search_my_meetings", RequiresFlags: `{"account_ids":["001..."],"start_date":"2026-04-01","end_date":"2026-04-30"}`}),

		// --- Knowledge & documents (consolidated — replaces search_account_* variants) ---
		newStubToolCommand(stubToolSpec{Use: "find_relevant_documents", Short: "Find vendor documents relevant to a user message", BackendTool: "find_relevant_documents", RequiresFlags: `{"user_message":"prep me for the Acme security review"}`}),
		newStubToolCommand(stubToolSpec{Use: "search_document_insights", Short: "Search extracted facts across all document sources", BackendTool: "search_document_insights", RequiresFlags: `{"search_sentences":["security questionnaire","redlines"]}`}),
		newStubToolCommand(stubToolSpec{Use: "search_knowledge_articles", Short: "Search synthesized customer knowledge articles", BackendTool: "search_knowledge_articles", RequiresFlags: `{"search_query":"renewal risks"}`}),
		newStubToolCommand(stubToolSpec{Use: "search_salesforce_notes", Short: "Fetch Salesforce notes for an account", BackendTool: "search_salesforce_notes", RequiresFlags: `{"account_id":"001..."}`}),
		newStubToolCommand(stubToolSpec{Use: "search_slack_messages", Short: "Fetch Slack messages for an account", BackendTool: "search_slack_messages", RequiresFlags: `{"account_id":"001..."}`}),

		// --- User preferences ---
		newStubToolCommand(stubToolSpec{Use: "get_user_preferences", Short: "Fetch the current user's core preferences", BackendTool: "get_user_preferences"}),
		newStubToolCommand(stubToolSpec{Use: "search_user_preferences", Short: "Search the current user's contextual memories", BackendTool: "search_user_preferences", RequiresFlags: `{"query":"pricing preferences"}`}),

		// --- Data & research ---
		newStubToolCommand(stubToolSpec{Use: "query_data", Short: "Query warehouse data with natural language", BackendTool: "query_data", RequiresFlags: `{"messages":[{"user_message":"show top 10 accounts by ARR"}]}`}),
		newStubToolCommand(stubToolSpec{Use: "query_dataset", Short: "Query uploaded datasets with natural language", BackendTool: "query_dataset", RequiresFlags: `{"user_message":"top 10 accounts by deal size"}`}),
		newStubToolCommand(stubToolSpec{Use: "search_datasets", Short: "Search available structured datasets", BackendTool: "search_datasets", RequiresFlags: `{"query":"renewals dataset"}`}),
		newStubToolCommand(stubToolSpec{Use: "research_company", Short: "Research a company using external sources", BackendTool: "research_company", RequiresFlags: `{"company_name":"Acme"}`}),
		newStubToolCommand(stubToolSpec{Use: "news_search", Short: "Run a Google News search", BackendTool: "news_search", RequiresFlags: `{"query":"Acme funding"}`}),
		newStubToolCommand(stubToolSpec{Use: "web_search", Short: "Run a public web search", BackendTool: "web_search", RequiresFlags: `{"query":"Acme pricing page"}`}),

		// --- Earnings calls (consolidated — replaces summarize_account_earnings_calls) ---
		newStubToolCommand(stubToolSpec{Use: "summarize_earnings_calls", Short: "Summarize earnings calls across accounts", BackendTool: "summarize_earnings_calls", RequiresFlags: `{"account_ids":["001..."]}`}),
	)
}
