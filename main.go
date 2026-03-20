package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
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

type authConfig struct {
	APIKey string `json:"api_key,omitempty"`
	OrgID  string `json:"org_id,omitempty"`
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
	apiKey, orgID, err := loadCredentials()

	if err != nil {
		return nil, err
	}

	switch {
	case apiKey == "":
		return nil, errors.New("ENDGAME_API_KEY is required")
	case orgID == "":
		return nil, errors.New("ENDGAME_ORG_ID is required")
	}

	timeout, err := getTimeout()
	if err != nil {
		return nil, err
	}

	return buildClient(apiKey, orgID, timeout), nil
}

func getTimeout() (time.Duration, error) {
	timeout := 120 * time.Second
	if timeoutValue := strings.TrimSpace(os.Getenv("ENDGAME_TIMEOUT_SECONDS")); timeoutValue != "" {
		timeoutSeconds, err := strconv.Atoi(timeoutValue)
		if err != nil || timeoutSeconds <= 0 {
			return 0, fmt.Errorf("invalid ENDGAME_TIMEOUT_SECONDS: %q", timeoutValue)
		}
		timeout = time.Duration(timeoutSeconds) * time.Second
	}

	return timeout, nil
}

func buildClient(apiKey, orgID string, timeout time.Duration) *endgameClient {
	return &endgameClient{
		apiKey: apiKey,
		orgID:  orgID,
		client: &http.Client{Timeout: timeout},
	}
}

func getAuthConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, ".endgame-auth.json"), nil
}

func saveAuth(config authConfig) error {
	configPath, err := getAuthConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0600)
}

func loadAuth() (*authConfig, error) {
	configPath, err := getAuthConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New("not authenticated")
		}
		return nil, err
	}

	var config authConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func loadCredentials() (string, string, error) {
	apiKey := strings.TrimSpace(os.Getenv("ENDGAME_API_KEY"))
	orgID := strings.TrimSpace(os.Getenv("ENDGAME_ORG_ID"))

	if apiKey != "" && orgID != "" {
		return apiKey, orgID, nil
	}

	config, err := loadAuth()
	if err != nil {
		if apiKey != "" || orgID != "" {
			return apiKey, orgID, nil
		}
		return "", "", err
	}

	if apiKey == "" {
		apiKey = strings.TrimSpace(config.APIKey)
	}
	if orgID == "" {
		orgID = strings.TrimSpace(config.OrgID)
	}

	return apiKey, orgID, nil
}

func promptInput(label string) (string, error) {
	fmt.Print(label)
	reader := bufio.NewReader(os.Stdin)
	value, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(value), nil
}

func login() error {
	configPath, err := getAuthConfigPath()
	if err != nil {
		return err
	}

	fmt.Println("🔐 Endgame Authentication")
	fmt.Println()
	fmt.Printf("Credentials will be stored in: %s\n", configPath)
	fmt.Println()

	apiKey, err := promptInput("Enter your Endgame API key: ")
	if err != nil {
		return err
	}
	if apiKey == "" {
		return errors.New("API key cannot be empty")
	}

	orgID, err := promptInput("Enter your Endgame org ID: ")
	if err != nil {
		return err
	}
	if orgID == "" {
		return errors.New("org ID cannot be empty")
	}

	client, err := newEndgameClientFromValues(apiKey, orgID)
	if err != nil {
		return err
	}
	if err := client.initialize(); err != nil {
		return fmt.Errorf("invalid credentials: %w", err)
	}
	if _, err := client.listTools(); err != nil {
		return fmt.Errorf("invalid credentials: %w", err)
	}

	if err := saveAuth(authConfig{APIKey: apiKey, OrgID: orgID}); err != nil {
		return err
	}

	fmt.Printf("\n✅ Authenticated for org %s\n", orgID)
	return nil
}

func logout() error {
	configPath, err := getAuthConfigPath()
	if err != nil {
		return err
	}

	err = os.Remove(configPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

func newEndgameClientFromValues(apiKey, orgID string) (*endgameClient, error) {
	timeout, err := getTimeout()
	if err != nil {
		return nil, err
	}
	return buildClient(apiKey, orgID, timeout), nil
}

func verifyAuth() (string, error) {
	client, err := newEndgameClient()
	if err != nil {
		return "", err
	}

	if err := client.initialize(); err != nil {
		return "", err
	}
	if _, err := client.listTools(); err != nil {
		return "", err
	}

	return client.orgID, nil
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
		Use:           "endgame",
		Short:         "CLI for the Endgame.io API",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Authenticate with Endgame",
		Long: `Authenticate with Endgame using an API key and org ID.

Examples:
  endgame auth              # Interactive authentication
  endgame auth login        # Same as above
  endgame auth status       # Check authentication status
  endgame auth logout       # Clear stored credentials`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return login()
		},
	}

	loginCmd := &cobra.Command{
		Use:   "login",
		Short: "Login to Endgame",
		RunE: func(cmd *cobra.Command, args []string) error {
			return login()
		},
	}

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Check authentication status",
		RunE: func(cmd *cobra.Command, args []string) error {
			orgID, err := verifyAuth()
			if err != nil {
				return fmt.Errorf("not authenticated: %w", err)
			}

			source := "config file"
			if strings.TrimSpace(os.Getenv("ENDGAME_API_KEY")) != "" && strings.TrimSpace(os.Getenv("ENDGAME_ORG_ID")) != "" {
				source = "environment"
			}

			fmt.Fprintln(cmd.OutOrStdout(), "✅ Authenticated")
			fmt.Fprintf(cmd.OutOrStdout(), "Org: %s\n", orgID)
			fmt.Fprintf(cmd.OutOrStdout(), "Source: %s\n", source)
			return nil
		},
	}

	logoutCmd := &cobra.Command{
		Use:   "logout",
		Short: "Logout from Endgame",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := logout(); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "✅ Successfully logged out")
			return nil
		},
	}

	whoamiCmd := &cobra.Command{
		Use:   "whoami",
		Short: "Show current authentication target",
		RunE: func(cmd *cobra.Command, args []string) error {
			return statusCmd.RunE(cmd, args)
		},
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

	rootCmd.AddCommand(authCmd, whoamiCmd, threadsCmd)
	authCmd.AddCommand(loginCmd, statusCmd, logoutCmd)
	threadsCmd.AddCommand(toolsCmd, promptCmd, followupCmd, askCmd, envCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
