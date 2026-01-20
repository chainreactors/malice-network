package core

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/chainreactors/malice-network/client/assets"
)

// AIClient handles communication with AI APIs (OpenAI and Claude)
type AIClient struct {
	settings *assets.AISettings
	client   *http.Client
}

// NewAIClient creates a new AI client
func NewAIClient(settings *assets.AISettings) *AIClient {
	timeout := 30
	if settings != nil && settings.Timeout > 0 {
		timeout = settings.Timeout
	}
	return &AIClient{
		settings: settings,
		client: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
	}
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAI API structures
type OpenAIChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
}

type OpenAIChatResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message      Message `json:"message"`
		FinishReason string  `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// Claude API structures
type ClaudeChatRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	System    string          `json:"system,omitempty"`
	Messages  []ClaudeMessage `json:"messages"`
}

type ClaudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ClaudeChatResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Error      *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// CommandSuggestion represents a command extracted from AI response
type CommandSuggestion struct {
	Command     string
	Description string
}

// Ask sends a question to the AI with context
func (c *AIClient) Ask(ctx context.Context, question string, history []string) (string, error) {
	if c.settings == nil || !c.settings.Enable {
		return "", fmt.Errorf("AI is not enabled. Use 'ai-config --enable' to enable it")
	}

	if c.settings.APIKey == "" {
		return "", fmt.Errorf("AI API key is not configured. Use 'ai-config --api-key <key>' to set it")
	}

	systemPrompt := c.buildSystemPrompt(history)

	switch strings.ToLower(c.settings.Provider) {
	case "claude", "anthropic":
		return c.askClaude(ctx, systemPrompt, question)
	default: // openai and compatible
		return c.askOpenAI(ctx, systemPrompt, question)
	}
}

func (c *AIClient) buildSystemPrompt(history []string) string {
	var sb strings.Builder
	sb.WriteString("You are an AI assistant for IoM (Malice Network), a C2 framework. ")
	sb.WriteString("Help users with commands, security operations, and answer questions. ")
	sb.WriteString("Be concise and provide actionable suggestions when possible.\n\n")

	sb.WriteString("When suggesting commands, wrap them in backticks like `command`. ")
	sb.WriteString("This helps users identify executable commands.\n\n")

	sb.WriteString("IMPORTANT: Use EXACT command names as listed below. Do NOT use plural forms or variations. ")
	sb.WriteString("For example, use `session` NOT `sessions`, use `listener` NOT `listeners`.\n\n")

	if len(history) > 0 {
		sb.WriteString("Recent command history:\n")
		for _, cmd := range history {
			sb.WriteString(fmt.Sprintf("- %s\n", cmd))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("Available commands (use these EXACT names):\n")
	sb.WriteString("- session: List and manage sessions (NOT 'sessions')\n")
	sb.WriteString("- listener: List listeners in server (NOT 'listeners')\n")
	sb.WriteString("- use <session_id>: Switch to a session\n")
	sb.WriteString("- ps: List processes\n")
	sb.WriteString("- ls, cd, pwd: File system navigation\n")
	sb.WriteString("- download, upload: File transfer\n")
	sb.WriteString("- execute, shell, run: Run commands on target\n")
	sb.WriteString("- job: List jobs\n")
	sb.WriteString("- pipeline: Manage pipelines\n")
	sb.WriteString("- build: Build implants\n")

	return sb.String()
}

// doRequest sends an HTTP POST request and returns the response body.
func (c *AIClient) doRequest(ctx context.Context, endpoint string, headers map[string]string, body []byte) ([]byte, int, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	for k, v := range headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response: %w", err)
	}

	return respBody, resp.StatusCode, nil
}

// buildEndpoint constructs the API endpoint URL with the given suffix.
func (c *AIClient) buildEndpoint(suffix string) (string, error) {
	base := strings.TrimSuffix(strings.TrimSpace(c.settings.Endpoint), "/")
	if base == "" {
		return "", fmt.Errorf("AI endpoint is not configured. Use 'ai-config --endpoint <url>' to set it")
	}
	if !strings.HasSuffix(base, suffix) {
		return base + suffix, nil
	}
	return base, nil
}

func (c *AIClient) askOpenAI(ctx context.Context, systemPrompt, question string) (string, error) {
	return c.askOpenAIWith(ctx, systemPrompt, question, c.settings.MaxTokens, 0.7)
}

func (c *AIClient) askOpenAIWith(ctx context.Context, systemPrompt, question string, maxTokens int, temperature float64) (string, error) {
	if maxTokens <= 0 {
		maxTokens = c.settings.MaxTokens
	}
	if temperature < 0 {
		temperature = 0.7
	}

	req := OpenAIChatRequest{
		Model:       c.settings.Model,
		Messages:    []Message{{Role: "system", Content: systemPrompt}, {Role: "user", Content: question}},
		MaxTokens:   maxTokens,
		Temperature: temperature,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint, err := c.buildEndpoint("/chat/completions")
	if err != nil {
		return "", err
	}

	respBody, statusCode, err := c.doRequest(ctx, endpoint, map[string]string{
		"Authorization": "Bearer " + c.settings.APIKey,
	}, body)
	if err != nil {
		return "", err
	}

	var chatResp OpenAIChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		if statusCode < 200 || statusCode >= 300 {
			return "", fmt.Errorf("API error (%d): %s", statusCode, strings.TrimSpace(string(respBody)))
		}
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if statusCode < 200 || statusCode >= 300 {
		if chatResp.Error != nil {
			return "", fmt.Errorf("API error (%d): %s", statusCode, chatResp.Error.Message)
		}
		return "", fmt.Errorf("API error (%d): %s", statusCode, strings.TrimSpace(string(respBody)))
	}

	if chatResp.Error != nil {
		return "", fmt.Errorf("API error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no response from AI")
	}

	return chatResp.Choices[0].Message.Content, nil
}

func (c *AIClient) askClaude(ctx context.Context, systemPrompt, question string) (string, error) {
	return c.askClaudeWith(ctx, systemPrompt, question, c.settings.MaxTokens)
}

func (c *AIClient) askClaudeWith(ctx context.Context, systemPrompt, question string, maxTokens int) (string, error) {
	if maxTokens <= 0 {
		maxTokens = c.settings.MaxTokens
	}
	if maxTokens <= 0 {
		maxTokens = 256
	}

	req := ClaudeChatRequest{
		Model:     c.settings.Model,
		MaxTokens: maxTokens,
		System:    systemPrompt,
		Messages:  []ClaudeMessage{{Role: "user", Content: question}},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint, err := c.buildEndpoint("/messages")
	if err != nil {
		return "", err
	}

	respBody, statusCode, err := c.doRequest(ctx, endpoint, map[string]string{
		"x-api-key":         c.settings.APIKey,
		"anthropic-version": "2023-06-01",
	}, body)
	if err != nil {
		return "", err
	}

	var chatResp ClaudeChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		if statusCode < 200 || statusCode >= 300 {
			return "", fmt.Errorf("API error (%d): %s", statusCode, strings.TrimSpace(string(respBody)))
		}
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if statusCode < 200 || statusCode >= 300 {
		if chatResp.Error != nil {
			return "", fmt.Errorf("API error (%d): %s", statusCode, chatResp.Error.Message)
		}
		return "", fmt.Errorf("API error (%d): %s", statusCode, strings.TrimSpace(string(respBody)))
	}

	if chatResp.Error != nil {
		return "", fmt.Errorf("API error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Content) == 0 {
		return "", fmt.Errorf("no response from AI")
	}

	var result strings.Builder
	for _, content := range chatResp.Content {
		if content.Type == "text" {
			result.WriteString(content.Text)
		}
	}

	return result.String(), nil
}

// ParseCommandSuggestions extracts command suggestions from AI response
// Commands are expected to be wrapped in backticks like `command`
func ParseCommandSuggestions(response string) []CommandSuggestion {
	var suggestions []CommandSuggestion

	// Match single backtick commands: `command`
	singlePattern := regexp.MustCompile("`([^`\n]+)`")
	matches := singlePattern.FindAllStringSubmatch(response, -1)

	seen := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 {
			cmd := strings.TrimSpace(match[1])
			// Skip if it looks like code/variable rather than command
			if strings.Contains(cmd, "=") || strings.HasPrefix(cmd, "$") {
				continue
			}
			// Skip shell escape syntax (! prefix)
			if strings.HasPrefix(cmd, "!") {
				continue
			}
			if !seen[cmd] {
				seen[cmd] = true
				suggestions = append(suggestions, CommandSuggestion{
					Command:     cmd,
					Description: "",
				})
			}
		}
	}

	return suggestions
}

// FormatResponseWithCommands formats the AI response with numbered command suggestions
func FormatResponseWithCommands(response string, commands []CommandSuggestion) string {
	if len(commands) == 0 {
		return response
	}

	var sb strings.Builder
	sb.WriteString(response)
	sb.WriteString("\n\n")
	sb.WriteString("Suggested commands:\n")

	for i, cmd := range commands {
		sb.WriteString(fmt.Sprintf("  [%d] %s\n", i+1, cmd.Command))
	}

	return sb.String()
}

// OpenAI streaming response structures
type OpenAIStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

// Claude streaming response structures
type ClaudeStreamEvent struct {
	Type  string `json:"type"`
	Index int    `json:"index,omitempty"`
	Delta *struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"delta,omitempty"`
}

// AskStream sends a question to the AI and streams the response
func (c *AIClient) AskStream(ctx context.Context, question string, history []string, onChunk func(chunk string)) (string, error) {
	if c.settings == nil || !c.settings.Enable {
		return "", fmt.Errorf("AI is not enabled. Use 'ai-config --enable' to enable it")
	}

	if c.settings.APIKey == "" {
		return "", fmt.Errorf("AI API key is not configured. Use 'ai-config --api-key <key>' to set it")
	}

	systemPrompt := c.buildSystemPrompt(history)

	switch strings.ToLower(c.settings.Provider) {
	case "claude", "anthropic":
		return c.askClaudeStream(ctx, systemPrompt, question, onChunk)
	default: // openai and compatible
		return c.askOpenAIStream(ctx, systemPrompt, question, onChunk)
	}
}

func (c *AIClient) askOpenAIStream(ctx context.Context, systemPrompt, question string, onChunk func(chunk string)) (string, error) {
	req := OpenAIChatRequest{
		Model:       c.settings.Model,
		Messages:    []Message{{Role: "system", Content: systemPrompt}, {Role: "user", Content: question}},
		MaxTokens:   c.settings.MaxTokens,
		Temperature: 0.7,
		Stream:      true,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint, err := c.buildEndpoint("/chat/completions")
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("Authorization", "Bearer "+c.settings.APIKey)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error (%d): %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var fullResponse strings.Builder
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk OpenAIStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			content := chunk.Choices[0].Delta.Content
			fullResponse.WriteString(content)
			if onChunk != nil {
				onChunk(content)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fullResponse.String(), fmt.Errorf("stream read error: %w", err)
	}

	return fullResponse.String(), nil
}

func (c *AIClient) askClaudeStream(ctx context.Context, systemPrompt, question string, onChunk func(chunk string)) (string, error) {
	reqBody := map[string]interface{}{
		"model":      c.settings.Model,
		"max_tokens": c.settings.MaxTokens,
		"system":     systemPrompt,
		"messages":   []ClaudeMessage{{Role: "user", Content: question}},
		"stream":     true,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint, err := c.buildEndpoint("/messages")
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("x-api-key", c.settings.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error (%d): %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var fullResponse strings.Builder
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		var event ClaudeStreamEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		if event.Type == "content_block_delta" && event.Delta != nil && event.Delta.Text != "" {
			fullResponse.WriteString(event.Delta.Text)
			if onChunk != nil {
				onChunk(event.Delta.Text)
			}
		}

		if event.Type == "message_stop" {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return fullResponse.String(), fmt.Errorf("stream read error: %w", err)
	}

	return fullResponse.String(), nil
}
