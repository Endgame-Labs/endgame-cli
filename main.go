package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	endgameBaseURL = "https://app.endgame.io/api/bridges"
	pollDelay      = 3 * time.Second
	defaultPolls   = 40
)

type endgameClient struct {
	apiKey string
	orgID  string
	reqID  int
	client *http.Client
}

type mcpTool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type rpcEnvelope struct {
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type toolCallResult struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

func newEndgameClient() (*endgameClient, error) {
	apiKey := strings.TrimSpace(os.Getenv("ENDGAME_API_KEY"))
	orgID := strings.TrimSpace(os.Getenv("ENDGAME_ORG_ID"))
	timeout := 120 * time.Second

	switch {
	case apiKey == "":
		return nil, errors.New("ENDGAME_API_KEY is required")
	case orgID == "":
		return nil, errors.New("ENDGAME_ORG_ID is required")
	}

	if timeoutValue := strings.TrimSpace(os.Getenv("ENDGAME_TIMEOUT_SECONDS")); timeoutValue != "" {
		timeoutSeconds, err := strconv.Atoi(timeoutValue)
		if err != nil || timeoutSeconds <= 0 {
			return nil, fmt.Errorf("invalid ENDGAME_TIMEOUT_SECONDS: %q", timeoutValue)
		}
		timeout = time.Duration(timeoutSeconds) * time.Second
	}

	return &endgameClient{
		apiKey: apiKey,
		orgID:  orgID,
		client: &http.Client{Timeout: timeout},
	}, nil
}

func (c *endgameClient) endpoint() string {
	return fmt.Sprintf("%s/%s/mcp", endgameBaseURL, c.orgID)
}

func (c *endgameClient) nextID() int {
	c.reqID++
	return c.reqID
}

// mcpCall sends a JSON-RPC 2.0 request and parses the SSE response stream.
func (c *endgameClient) mcpCall(method string, params any) (json.RawMessage, error) {
	reqBody := map[string]any{
		"jsonrpc": "2.0",
		"id":      c.nextID(),
		"method":  method,
	}
	if params != nil {
		reqBody["params"] = params
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.endpoint(), bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(resp.Body)
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(buf.String()))
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		payload := strings.TrimPrefix(line, "data: ")
		var rpcResp rpcEnvelope
		if err := json.Unmarshal([]byte(payload), &rpcResp); err != nil {
			continue
		}
		if rpcResp.Error != nil {
			return nil, fmt.Errorf("rpc error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
		}
		if rpcResp.Result != nil {
			return rpcResp.Result, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read sse stream: %w", err)
	}

	return nil, errors.New("no result in SSE stream")
}

func (c *endgameClient) initialize() error {
	_, err := c.mcpCall("initialize", map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]string{
			"name":    "endgame-cli",
			"version": "0.1.0",
		},
	})
	return err
}

func (c *endgameClient) listTools() ([]mcpTool, error) {
	result, err := c.mcpCall("tools/list", nil)
	if err != nil {
		return nil, err
	}

	var parsed struct {
		Tools []mcpTool `json:"tools"`
	}
	if err := json.Unmarshal(result, &parsed); err != nil {
		return nil, fmt.Errorf("decode tools list: %w", err)
	}

	return parsed.Tools, nil
}

func (c *endgameClient) prompt(question string) (string, error) {
	result, err := c.mcpCall("tools/call", map[string]any{
		"name":      "prompt_endgame",
		"arguments": map[string]string{"prompt": question},
	})
	if err != nil {
		return "", err
	}

	parsed, err := decodeToolCallResult(result)
	if err != nil {
		return "", err
	}

	for _, item := range parsed.Content {
		var inner map[string]any
		if err := json.Unmarshal([]byte(item.Text), &inner); err != nil {
			continue
		}
		if opID, ok := inner["operationId"].(string); ok && opID != "" {
			return opID, nil
		}
	}

	return "", errors.New("no operationId in response")
}

func (c *endgameClient) followup(opID string) (string, string, error) {
	result, err := c.mcpCall("tools/call", map[string]any{
		"name":      "message_followup",
		"arguments": map[string]string{"operationId": opID},
	})
	if err != nil {
		return "", "", err
	}

	parsed, err := decodeToolCallResult(result)
	if err != nil {
		return "", "", err
	}

	for _, item := range parsed.Content {
		var inner map[string]any
		if err := json.Unmarshal([]byte(item.Text), &inner); err != nil {
			continue
		}

		status, _ := inner["status"].(string)
		if status == "completed" {
			if artifact, ok := inner["artifact"].(map[string]any); ok {
				if content, ok := artifact["content"].(string); ok {
					return status, content, nil
				}
			}
			return status, item.Text, nil
		}

		if status != "" {
			return status, "", nil
		}
	}

	return "unknown", "", nil
}

func decodeToolCallResult(result json.RawMessage) (*toolCallResult, error) {
	var parsed toolCallResult
	if err := json.Unmarshal(result, &parsed); err != nil {
		return nil, fmt.Errorf("decode tool call result: %w", err)
	}
	return &parsed, nil
}

func waitForCompletion(client *endgameClient, opID string, maxPolls int, delay time.Duration) (string, error) {
	for i := 0; i < maxPolls; i++ {
		if i > 0 {
			time.Sleep(delay)
		}

		status, answer, err := client.followup(opID)
		if err != nil {
			return "", err
		}
		if status == "completed" {
			return answer, nil
		}
	}

	return "", fmt.Errorf("operation %s did not complete after %d polls", opID, maxPolls)
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "endgame",
		Short: "CLI for the Endgame.io API",
	}

	threadsCmd := &cobra.Command{
		Use:   "threads",
		Short: "Interact with Endgame MCP thread tools",
	}

	toolsCmd := &cobra.Command{
		Use:   "tools",
		Short: "List available MCP tools",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newEndgameClient()
			if err != nil {
				return err
			}
			if err := client.initialize(); err != nil {
				return err
			}

			tools, err := client.listTools()
			if err != nil {
				return err
			}

			for _, tool := range tools {
				if tool.Description == "" {
					fmt.Fprintln(cmd.OutOrStdout(), tool.Name)
					continue
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", tool.Name, tool.Description)
			}
			return nil
		},
	}

	promptCmd := &cobra.Command{
		Use:   "prompt <question>",
		Short: "Start a new Endgame thread prompt and print the operation ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newEndgameClient()
			if err != nil {
				return err
			}
			if err := client.initialize(); err != nil {
				return err
			}

			opID, err := client.prompt(args[0])
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), opID)
			return nil
		},
	}

	followupCmd := &cobra.Command{
		Use:   "followup <operation-id>",
		Short: "Fetch the latest status for an existing operation",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newEndgameClient()
			if err != nil {
				return err
			}
			if err := client.initialize(); err != nil {
				return err
			}

			status, answer, err := client.followup(args[0])
			if err != nil {
				return err
			}

			if answer == "" {
				fmt.Fprintln(cmd.OutOrStdout(), status)
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "status: %s\n%s\n", status, answer)
			return nil
		},
	}

	askCmd := &cobra.Command{
		Use:   "ask <question>",
		Short: "Prompt Endgame and wait for the completed response",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newEndgameClient()
			if err != nil {
				return err
			}
			if err := client.initialize(); err != nil {
				return err
			}

			maxPolls, err := cmd.Flags().GetInt("max-polls")
			if err != nil {
				return err
			}
			delaySeconds, err := cmd.Flags().GetInt("poll-delay")
			if err != nil {
				return err
			}

			opID, err := client.prompt(args[0])
			if err != nil {
				return err
			}

			answer, err := waitForCompletion(client, opID, maxPolls, time.Duration(delaySeconds)*time.Second)
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), answer)
			return nil
		},
	}

	askCmd.Flags().Int("max-polls", defaultPolls, "maximum number of followup polls before timing out")
	askCmd.Flags().Int("poll-delay", int(pollDelay/time.Second), "seconds to wait between followup polls")

	envCmd := &cobra.Command{
		Use:   "env",
		Short: "Show required environment variables",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(cmd.OutOrStdout(), "ENDGAME_API_KEY")
			fmt.Fprintln(cmd.OutOrStdout(), "ENDGAME_ORG_ID")
			if timeout := os.Getenv("ENDGAME_TIMEOUT_SECONDS"); timeout != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "ENDGAME_TIMEOUT_SECONDS=%s\n", timeout)
			}
		},
	}

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		_ = cmd
		_ = args
		if timeoutValue := strings.TrimSpace(os.Getenv("ENDGAME_TIMEOUT_SECONDS")); timeoutValue != "" {
			timeoutSeconds, err := strconv.Atoi(timeoutValue)
			if err != nil || timeoutSeconds <= 0 {
				return fmt.Errorf("invalid ENDGAME_TIMEOUT_SECONDS: %q", timeoutValue)
			}
		}
		return nil
	}

	rootCmd.AddCommand(threadsCmd)
	threadsCmd.AddCommand(toolsCmd, promptCmd, followupCmd, askCmd, envCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
