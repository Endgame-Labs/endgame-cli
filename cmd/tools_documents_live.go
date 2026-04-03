package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Endgame-Labs/endgame-cli/pkg/auth"
	"github.com/Endgame-Labs/endgame-cli/pkg/endgame"
	"github.com/spf13/cobra"
)

func newDocumentsKnowledgeCommand() *cobra.Command {
	return newDocumentsKnowledgeCommandWithName("knowledge")
}

func newDocumentsKnowledgeCommandWithName(use string) *cobra.Command {
	var accountIDs []string
	var afterDate string
	var beforeDate string
	var contentType string
	var filename string
	var pageOffset int

	cmd := &cobra.Command{
		Use:   use,
		Short: "Fetch knowledge documents and metadata",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = args

			arguments := map[string]any{}
			if len(accountIDs) > 0 {
				arguments["account_ids"] = accountIDs
			}
			if v := strings.TrimSpace(afterDate); v != "" {
				arguments["after_date"] = v
			}
			if v := strings.TrimSpace(beforeDate); v != "" {
				arguments["before_date"] = v
			}
			if v := strings.TrimSpace(contentType); v != "" {
				arguments["content_type"] = v
			}
			if v := strings.TrimSpace(filename); v != "" {
				arguments["filename"] = v
			}
			if pageOffset > 0 {
				arguments["page_offset"] = pageOffset
			}

			return executeMCPTool(cmd, "fetch_knowledge_documents", arguments)
		},
	}

	cmd.Flags().StringSliceVar(&accountIDs, "account-id", nil, "account ID to filter by; repeat to pass multiple values")
	cmd.Flags().StringVar(&afterDate, "after-date", "", "only include documents uploaded after this ISO-8601 timestamp")
	cmd.Flags().StringVar(&beforeDate, "before-date", "", "only include documents uploaded before this ISO-8601 timestamp")
	cmd.Flags().StringVar(&contentType, "content-type", "", "filter by MIME content type")
	cmd.Flags().StringVar(&filename, "filename", "", "filter by filename substring")
	cmd.Flags().IntVar(&pageOffset, "page-offset", 0, "page offset for paginated results")
	return cmd
}

func newDocumentsFetchCommand() *cobra.Command {
	return newDocumentsFetchCommandWithName("fetch")
}

func newDocumentsFetchCommandWithName(use string) *cobra.Command {
	var accountID string
	var documentType string
	var documentID string

	cmd := &cobra.Command{
		Use:   use,
		Short: "Retrieve a full document by document ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = args

			if strings.TrimSpace(documentID) == "" {
				return fmt.Errorf("--document-id is required")
			}
			if strings.TrimSpace(accountID) == "" {
				return fmt.Errorf("--account-id is required")
			}

			return executeMCPTool(cmd, "get_document", map[string]any{
				"account_id":    accountID,
				"document_type": documentType,
				"document_id":   documentID,
			})
		},
	}

	cmd.Flags().StringVar(&accountID, "account-id", "", "account ID for the document, for example VENDOR")
	cmd.Flags().StringVar(&documentType, "document-type", "knowledge_document", "document type to fetch")
	cmd.Flags().StringVar(&documentID, "document-id", "", "document ID to fetch")
	return cmd
}

func newDocumentsVendorKnowledgeCommand() *cobra.Command {
	return newDocumentsVendorKnowledgeCommandWithName("vendor-knowledge")
}

func newDocumentsVendorKnowledgeCommandWithName(use string) *cobra.Command {
	var query string
	var maxResults int

	cmd := &cobra.Command{
		Use:   use,
		Short: "Search vendor knowledge documents semantically",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = args

			if strings.TrimSpace(query) == "" {
				return fmt.Errorf("--query is required")
			}
			if maxResults < 1 || maxResults > 10 {
				return fmt.Errorf("--max-results must be between 1 and 10")
			}

			return executeMCPTool(cmd, "search_vendor_documents", map[string]any{
				"search_query": query,
				"max_results":  maxResults,
			})
		},
	}

	cmd.Flags().StringVar(&query, "query", "", "semantic search query")
	cmd.Flags().IntVar(&maxResults, "max-results", 5, "maximum number of results to return (1-10)")
	return cmd
}

func executeMCPTool(cmd *cobra.Command, toolName string, arguments map[string]any) error {
	client, err := auth.NewClient()
	if err != nil {
		return err
	}
	if err := client.Initialize(); err != nil {
		return err
	}

	result, err := client.CallTool(toolName, arguments)
	if err != nil {
		return err
	}

	return printToolCallResponse(cmd, result)
}

func printToolCallResponse(cmd *cobra.Command, result *endgame.ToolCallResponse) error {
	for _, item := range result.Content {
		text := strings.TrimSpace(item.Text)
		if text == "" {
			continue
		}

		var payload any
		if err := json.Unmarshal([]byte(text), &payload); err == nil {
			data, err := json.MarshalIndent(payload, "", "  ")
			if err != nil {
				return fmt.Errorf("format tool response: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(data))
			continue
		}

		fmt.Fprintln(cmd.OutOrStdout(), text)
	}

	return nil
}
