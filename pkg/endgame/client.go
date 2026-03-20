package endgame

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
)

const BaseURL = "https://app.endgame.io/api/bridges"

type Client struct {
	apiKey string
	orgID  string
	reqID  int
	client *http.Client
}

type Tool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
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

type toolCallResult struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

func NewClient(apiKey, orgID string) (*Client, error) {
	apiKey = strings.TrimSpace(apiKey)
	orgID = strings.TrimSpace(orgID)

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

	return &Client{
		apiKey: apiKey,
		orgID:  orgID,
		client: &http.Client{Timeout: timeout},
	}, nil
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

func (c *Client) OrgID() string {
	return c.orgID
}

func (c *Client) endpoint() string {
	return fmt.Sprintf("%s/%s/mcp", BaseURL, c.orgID)
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

func (c *Client) Initialize() error {
	_, err := c.Call("initialize", map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]string{
			"name":    "endgame-cli",
			"version": "0.1.0",
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

	result, err := c.Call("tools/call", map[string]any{
		"name":      "prompt_endgame",
		"arguments": arguments,
	})
	if err != nil {
		return nil, err
	}

	parsed, err := decodeToolCallResult(result)
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
	result, err := c.Call("tools/call", map[string]any{
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
