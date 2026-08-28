package endgame

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Endgame-Labs/endgame-cli/pkg/buildinfo"
)

const BaseURL = "https://app.endgame.io/api/v1/mcp"

const previewURLFormat = "https://pr-%d-vite.preview.end-p1.endgame.build/api/v1/mcp"

type Client struct {
	reqID       int
	client      *http.Client
	headers     map[string]string
	endpointURL string
}

type ClientOption func(*Client)

type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema,omitempty"`
}

type ToolCallContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type ToolCallResponse struct {
	Content []ToolCallContent `json:"content"`
}

type PromptResult struct {
	OperationID string
	ThreadID    string
	RawText     string
}

type rpcEnvelope struct {
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func WithHeaders(headers map[string]string) ClientOption {
	return func(client *Client) {
		client.headers = cloneHeaders(headers)
	}
}

// WithEndpoint directs MCP calls to a prevalidated endpoint. The CLI exposes
// only production and deterministic Endgame preview hosts to avoid forwarding
// bearer tokens to arbitrary servers.
func WithEndpoint(endpoint string) ClientOption {
	return func(client *Client) {
		client.endpointURL = strings.TrimSpace(endpoint)
	}
}

// PreviewURL returns the public MCP endpoint for a Cerebro PR preview.
func PreviewURL(pullRequestNumber int) (string, error) {
	if pullRequestNumber <= 0 {
		return "", errors.New("preview pull request number must be positive")
	}
	return fmt.Sprintf(previewURLFormat, pullRequestNumber), nil
}

func NewClient(httpClient *http.Client, options ...ClientOption) (*Client, error) {
	if httpClient == nil {
		return nil, errors.New("http client is required")
	}
	timeout, err := getTimeout()
	if err != nil {
		return nil, err
	}

	httpClient.Timeout = timeout

	client := &Client{
		client:      httpClient,
		endpointURL: BaseURL,
	}
	for _, option := range options {
		if option != nil {
			option(client)
		}
	}

	return client, nil
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

func (c *Client) endpoint() string {
	return c.endpointURL
}

func (c *Client) nextID() int {
	c.reqID++
	return c.reqID
}

func (c *Client) Call(method string, params any) (json.RawMessage, error) {
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
	for key, value := range c.headers {
		req.Header[key] = []string{value}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

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

	if strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "text/event-stream") {
		result, err := readSSEResult(resp.Body)
		if err != nil {
			return nil, err
		}
		return result, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	var rpcResp rpcEnvelope
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return nil, fmt.Errorf("decode response body: %w", err)
	}
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("rpc error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}
	if rpcResp.Result == nil {
		return nil, errors.New("no result in response body")
	}

	return rpcResp.Result, nil
}

func cloneHeaders(headers map[string]string) map[string]string {
	if len(headers) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(headers))
	for key, value := range headers {
		cloned[key] = value
	}
	return cloned
}

func (c *Client) Initialize() error {
	_, err := c.Call("initialize", map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]string{
			"name":    "endgame-cli",
			"version": buildinfo.Version,
		},
	})
	return err
}

func (c *Client) ListTools() ([]Tool, error) {
	result, err := c.Call("tools/list", nil)
	if err != nil {
		return nil, err
	}

	var parsed struct {
		Tools []Tool `json:"tools"`
	}
	if err := json.Unmarshal(result, &parsed); err != nil {
		return nil, fmt.Errorf("decode tools list: %w", err)
	}

	return parsed.Tools, nil
}

func (c *Client) SubmitPrompt(question, threadID string) (*PromptResult, error) {
	arguments := map[string]string{"prompt": question}
	if strings.TrimSpace(threadID) != "" {
		arguments["threadId"] = threadID
	}

	parsed, err := c.CallTool("prompt_endgame", arguments)
	if err != nil {
		return nil, err
	}

	for _, item := range parsed.Content {
		var inner map[string]any
		if err := json.Unmarshal([]byte(item.Text), &inner); err != nil {
			continue
		}

		response := &PromptResult{
			RawText: item.Text,
		}
		if opID, ok := inner["operationId"].(string); ok && opID != "" {
			response.OperationID = opID
		}
		if tid, ok := inner["threadId"].(string); ok && tid != "" {
			response.ThreadID = tid
		}
		if response.OperationID != "" || response.ThreadID != "" {
			return response, nil
		}
	}

	return nil, errors.New("no operationId or threadId in response")
}

func (c *Client) Followup(opID string) (string, string, error) {
	parsed, err := c.CallTool("message_followup", map[string]string{"operationId": opID})
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

func (c *Client) CallTool(name string, arguments any) (*ToolCallResponse, error) {
	result, err := c.Call("tools/call", map[string]any{
		"name":      name,
		"arguments": arguments,
	})
	if err != nil {
		return nil, err
	}

	var parsed ToolCallResponse
	if err := json.Unmarshal(result, &parsed); err != nil {
		return nil, fmt.Errorf("decode tool call result: %w", err)
	}
	return &parsed, nil
}

func WaitForCompletion(client *Client, opID string, maxPolls int, delay time.Duration) (string, error) {
	for i := 0; i < maxPolls; i++ {
		if i > 0 {
			time.Sleep(delay)
		}

		status, answer, err := client.Followup(opID)
		if err != nil {
			return "", err
		}
		if status == "completed" {
			return answer, nil
		}
	}

	return "", fmt.Errorf("operation %s did not complete after %d polls", opID, maxPolls)
}

func readSSEResult(body io.Reader) (json.RawMessage, error) {
	reader := bufio.NewReader(body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("read sse stream: %w", err)
		}

		line = strings.TrimRight(line, "\r\n")
		if !strings.HasPrefix(line, "data: ") {
			if errors.Is(err, io.EOF) {
				break
			}
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

		if errors.Is(err, io.EOF) {
			break
		}
	}

	return nil, errors.New("no result in SSE stream")
}
