//go:build bridge_agent_proto
// +build bridge_agent_proto

package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// reactTool defines a tool the ReAct agent can invoke.
type reactTool struct {
	Name        string
	Description string
	Parameters  map[string]any // JSON Schema for parameters
	Fn          func(args map[string]string) string
}

// reactAgent implements a minimal ReAct agent loop using OpenAI function calling.
// This mirrors the agent loop that runs inside the implant's bridge_agent module.
type reactAgent struct {
	baseURL  string
	apiKey   string
	model    string
	tools    []reactTool
	maxTurns int
}

type chatMessage struct {
	Role       string    `json:"role"`
	Content    *string   `json:"content,omitempty"`
	ToolCalls  []toolUse `json:"tool_calls,omitempty"`
	ToolCallID string    `json:"tool_call_id,omitempty"`
}

type toolUse struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function functionCall `json:"function"`
}

type functionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message      chatMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
}

// run executes the agent loop and returns the final text, iteration count, and tool calls made.
func (a *reactAgent) run(prompt string) (finalText string, iterations int, toolCallsMade int, err error) {
	systemPrompt := "You are a ReAct agent. Use the provided tools to answer questions. " +
		"When you have the final answer, respond with plain text (no tool calls)."
	sysContent := systemPrompt
	userContent := prompt
	messages := []chatMessage{
		{Role: "system", Content: &sysContent},
		{Role: "user", Content: &userContent},
	}

	toolDefs := a.buildToolDefs()
	toolMap := make(map[string]func(map[string]string) string, len(a.tools))
	for _, t := range a.tools {
		toolMap[t.Name] = t.Fn
	}

	maxTurns := a.maxTurns
	if maxTurns == 0 {
		maxTurns = 10
	}

	for iterations = 1; iterations <= maxTurns; iterations++ {
		resp, callErr := a.callCompletion(messages, toolDefs)
		if callErr != nil {
			return "", iterations, toolCallsMade, callErr
		}

		if len(resp.Choices) == 0 {
			return "", iterations, toolCallsMade, fmt.Errorf("no choices in response")
		}

		choice := resp.Choices[0]
		assistantMsg := choice.Message

		// Append assistant message to history
		messages = append(messages, assistantMsg)

		if choice.FinishReason == "tool_calls" || len(assistantMsg.ToolCalls) > 0 {
			for _, tc := range assistantMsg.ToolCalls {
				toolCallsMade++
				fn, ok := toolMap[tc.Function.Name]
				if !ok {
					result := fmt.Sprintf("error: unknown tool %q", tc.Function.Name)
					messages = append(messages, chatMessage{
						Role:       "tool",
						Content:    &result,
						ToolCallID: tc.ID,
					})
					continue
				}

				// Parse arguments
				var args map[string]string
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
					// Try parsing as map[string]any and convert
					var anyArgs map[string]any
					if err2 := json.Unmarshal([]byte(tc.Function.Arguments), &anyArgs); err2 != nil {
						result := fmt.Sprintf("error: invalid arguments: %s", err2)
						messages = append(messages, chatMessage{
							Role:       "tool",
							Content:    &result,
							ToolCallID: tc.ID,
						})
						continue
					}
					args = make(map[string]string, len(anyArgs))
					for k, v := range anyArgs {
						args[k] = fmt.Sprintf("%v", v)
					}
				}

				result := fn(args)
				messages = append(messages, chatMessage{
					Role:       "tool",
					Content:    &result,
					ToolCallID: tc.ID,
				})
			}
			continue
		}

		// No tool calls: final answer
		if assistantMsg.Content != nil {
			return *assistantMsg.Content, iterations, toolCallsMade, nil
		}
		return "", iterations, toolCallsMade, nil
	}

	return "", iterations - 1, toolCallsMade, fmt.Errorf("exceeded max turns (%d)", maxTurns)
}

func (a *reactAgent) buildToolDefs() []map[string]any {
	defs := make([]map[string]any, len(a.tools))
	for i, t := range a.tools {
		params := t.Parameters
		if params == nil {
			params = map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			}
		}
		defs[i] = map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        t.Name,
				"description": t.Description,
				"parameters":  params,
			},
		}
	}
	return defs
}

func (a *reactAgent) callCompletion(messages []chatMessage, tools []map[string]any) (*chatCompletionResponse, error) {
	reqBody := map[string]any{
		"model":    a.model,
		"messages": messages,
	}
	if len(tools) > 0 {
		reqBody["tools"] = tools
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	endpoint := strings.TrimSuffix(a.baseURL, "/") + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(
		context.Background(), "POST", endpoint, bytes.NewReader(bodyBytes),
	)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+a.apiKey)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	var result chatCompletionResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return &result, nil
}

// --- Test tools ---

func newTestTools() []reactTool {
	return []reactTool{
		{
			Name:        "get_current_time",
			Description: "Returns the current date and time in RFC3339 format",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
			Fn: func(args map[string]string) string {
				return time.Now().Format(time.RFC3339)
			},
		},
		{
			Name:        "reverse_string",
			Description: "Reverses a given string",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"text": map[string]any{
						"type":        "string",
						"description": "The string to reverse",
					},
				},
				"required": []string{"text"},
			},
			Fn: func(args map[string]string) string {
				text := args["text"]
				runes := []rune(text)
				for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
					runes[i], runes[j] = runes[j], runes[i]
				}
				return string(runes)
			},
		},
		{
			Name:        "calculate",
			Description: "Evaluates a simple math expression. Supports: add, subtract, multiply, divide",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"operation": map[string]any{
						"type":        "string",
						"description": "The operation: add, subtract, multiply, divide",
						"enum":        []string{"add", "subtract", "multiply", "divide"},
					},
					"a": map[string]any{
						"type":        "number",
						"description": "First operand",
					},
					"b": map[string]any{
						"type":        "number",
						"description": "Second operand",
					},
				},
				"required": []string{"operation", "a", "b"},
			},
			Fn: func(args map[string]string) string {
				var a, b float64
				fmt.Sscanf(args["a"], "%f", &a)
				fmt.Sscanf(args["b"], "%f", &b)
				switch args["operation"] {
				case "add":
					return fmt.Sprintf("%g", a+b)
				case "subtract":
					return fmt.Sprintf("%g", a-b)
				case "multiply":
					return fmt.Sprintf("%g", a*b)
				case "divide":
					if b == 0 {
						return "error: division by zero"
					}
					return fmt.Sprintf("%g", a/b)
				default:
					return fmt.Sprintf("error: unknown operation %q", args["operation"])
				}
			},
		},
	}
}

func newTestAgent() *reactAgent {
	baseURL, apiKey, _ := resolve(ProviderOpts{
		Provider: "openai",
		APIKey:   "sk-Kdp7jXbyICmcCh7k6",
		Endpoint: "https://wafcdn.aimeeting.store/v1",
	})
	return &reactAgent{
		baseURL:  baseURL,
		apiKey:   apiKey,
		model:    "gpt-5.4",
		tools:    newTestTools(),
		maxTurns: 10,
	}
}

// --- Tests ---

func TestReactAgent_SimpleChat(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live API test in short mode")
	}

	agent := newTestAgent()
	agent.tools = nil // no tools for pure chat

	text, iterations, toolCalls, err := agent.run("Say 'hello world' and nothing else.")
	if err != nil {
		t.Fatalf("agent.run: %v", err)
	}

	t.Logf("response: %q (iterations=%d, toolCalls=%d)", text, iterations, toolCalls)

	if !strings.Contains(strings.ToLower(text), "hello world") {
		t.Errorf("expected response containing 'hello world', got: %q", text)
	}
	if iterations != 1 {
		t.Errorf("expected 1 iteration for simple chat, got %d", iterations)
	}
	if toolCalls != 0 {
		t.Errorf("expected 0 tool calls, got %d", toolCalls)
	}
}

func TestReactAgent_SingleToolCall(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live API test in short mode")
	}

	agent := newTestAgent()

	text, iterations, toolCalls, err := agent.run(
		"What is the current time? Use the get_current_time tool to find out, then tell me.",
	)
	if err != nil {
		t.Fatalf("agent.run: %v", err)
	}

	t.Logf("response: %q (iterations=%d, toolCalls=%d)", text, iterations, toolCalls)

	if toolCalls < 1 {
		t.Errorf("expected at least 1 tool call, got %d", toolCalls)
	}
	if iterations < 2 {
		t.Errorf("expected at least 2 iterations (tool call + response), got %d", iterations)
	}
	if text == "" {
		t.Error("expected non-empty final text")
	}
}

func TestReactAgent_MultiToolCall(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live API test in short mode")
	}

	agent := newTestAgent()

	text, iterations, toolCalls, err := agent.run(
		"Please do these two things: 1) Reverse the string 'hello' using the reverse_string tool, " +
			"2) Get the current time using get_current_time. Report both results.",
	)
	if err != nil {
		t.Fatalf("agent.run: %v", err)
	}

	t.Logf("response: %q (iterations=%d, toolCalls=%d)", text, iterations, toolCalls)

	if toolCalls < 2 {
		t.Errorf("expected at least 2 tool calls, got %d", toolCalls)
	}
	if !strings.Contains(text, "olleh") {
		t.Errorf("expected response containing 'olleh' (reversed 'hello'), got: %q", text)
	}
}

func TestReactAgent_Calculate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live API test in short mode")
	}

	agent := newTestAgent()

	text, _, toolCalls, err := agent.run(
		"Use the calculate tool to multiply 7 by 8, then tell me the result.",
	)
	if err != nil {
		t.Fatalf("agent.run: %v", err)
	}

	t.Logf("response: %q (toolCalls=%d)", text, toolCalls)

	if toolCalls < 1 {
		t.Errorf("expected at least 1 tool call, got %d", toolCalls)
	}
	if !strings.Contains(text, "56") {
		t.Errorf("expected response containing '56', got: %q", text)
	}
}

// TestReactAgent_SkillRecon tests the recon skill prompt through the ReAct agent.
// It simulates the bridge_agent path: load SKILL.md → render → feed to agent with recon tools.
func TestReactAgent_SkillRecon(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live API test in short mode")
	}

	// The recon skill prompt (from helper/intl/community/resources/skills/recon/SKILL.md)
	reconPrompt := `Perform reconnaissance on the target system. Collect ALL of the following and output in a structured summary:

1. **OS & Host**: OS version, architecture, hostname, domain, kernel version
2. **Current User**: username, UID/SID, group memberships, privileges/sudo access
3. **Users & Groups**: all local users, recently logged-in users, admin/root group members
4. **Network**: interfaces, IP addresses, routing table, DNS servers, active connections (ESTABLISHED), listening ports
5. **Processes**: running processes with PID, user, command line — highlight security tools (AV/EDR/HIPS)
6. **Environment**: PATH, interesting environment variables (proxy, credentials, tokens)

Rules:
- Auto-detect OS (Linux/macOS/Windows) and use appropriate commands
- Run each command individually, do NOT chain with ` + "`&&`" + ` — if one fails, continue with the rest
- Do NOT install any packages or modify the system
- Output a final structured summary with all findings`

	// Simulated recon tools that return mock system data
	reconTools := []reactTool{
		{
			Name:        "execute_command",
			Description: "Execute a shell command on the target system and return its output",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"command": map[string]any{
						"type":        "string",
						"description": "The shell command to execute",
					},
				},
				"required": []string{"command"},
			},
			Fn: func(args map[string]string) string {
				cmd := args["command"]
				switch {
				case strings.Contains(cmd, "uname") && strings.Contains(cmd, "-a"):
					return "Linux recon-target 5.15.0-91-generic #101-Ubuntu SMP x86_64 GNU/Linux"
				case strings.Contains(cmd, "uname") && strings.Contains(cmd, "-r"):
					return "5.15.0-91-generic"
				case strings.Contains(cmd, "uname") && strings.Contains(cmd, "-m"):
					return "x86_64"
				case strings.Contains(cmd, "uname"):
					return "Linux"
				case strings.Contains(cmd, "hostname"):
					return "recon-target.lab.local"
				case strings.Contains(cmd, "cat /etc/os-release") || strings.Contains(cmd, "os-release"):
					return "NAME=\"Ubuntu\"\nVERSION=\"22.04.3 LTS (Jammy Jellyfish)\"\nID=ubuntu\nVERSION_ID=\"22.04\""
				case strings.Contains(cmd, "whoami"):
					return "operator"
				case strings.Contains(cmd, "sudo -l"):
					return "User operator may run the following commands on recon-target:\n    (ALL : ALL) NOPASSWD: ALL"
				case cmd == "id" || strings.HasPrefix(cmd, "id "):
					return "uid=1001(operator) gid=1001(operator) groups=1001(operator),27(sudo),999(docker)"
				case strings.Contains(cmd, "cat /etc/passwd") || strings.Contains(cmd, "getent passwd"):
					return "root:x:0:0:root:/root:/bin/bash\noperator:x:1001:1001::/home/operator:/bin/bash\nwww-data:x:33:33:www-data:/var/www:/usr/sbin/nologin\npostgres:x:114:120:PostgreSQL administrator:/var/lib/postgresql:/bin/bash"
				case strings.Contains(cmd, "last") || strings.Contains(cmd, "lastlog"):
					return "operator pts/0  192.168.1.50 Fri Mar 14 01:00   still logged in\nroot     pts/1  192.168.1.1  Thu Mar 13 22:15 - 23:00  (00:45)"
				case strings.Contains(cmd, "getent group") || strings.Contains(cmd, "cat /etc/group"):
					return "root:x:0:\nsudo:x:27:operator\ndocker:x:999:operator\nwww-data:x:33:"
				case strings.Contains(cmd, "ifconfig") || strings.Contains(cmd, "ip addr") || strings.Contains(cmd, "ip a"):
					return "1: lo: <LOOPBACK,UP> inet 127.0.0.1/8\n2: eth0: <BROADCAST,UP> inet 10.0.2.15/24\n3: docker0: <NO-CARRIER> inet 172.17.0.1/16"
				case strings.Contains(cmd, "route") || strings.Contains(cmd, "ip route") || strings.Contains(cmd, "ip r"):
					return "default via 10.0.2.1 dev eth0\n10.0.2.0/24 dev eth0 proto kernel src 10.0.2.15\n172.17.0.0/16 dev docker0 proto kernel src 172.17.0.1"
				case strings.Contains(cmd, "resolv.conf"):
					return "nameserver 10.0.2.1\nnameserver 8.8.8.8"
				case strings.Contains(cmd, "netstat") || strings.Contains(cmd, "ss "):
					return "tcp  ESTAB  0  0  10.0.2.15:22    192.168.1.50:54321\ntcp  LISTEN 0  128  0.0.0.0:22    0.0.0.0:*\ntcp  LISTEN 0  128  0.0.0.0:80    0.0.0.0:*\ntcp  LISTEN 0  128  127.0.0.1:5432  0.0.0.0:*\ntcp  LISTEN 0  128  0.0.0.0:443   0.0.0.0:*"
				case strings.Contains(cmd, "ps "):
					return "USER       PID  CMD\nroot         1  /sbin/init\nroot       412  /usr/sbin/sshd\nwww-data   891  nginx: worker process\npostgres   923  /usr/lib/postgresql/14/bin/postgres\nroot      1201  /usr/sbin/clamd\noperator  1450  -bash\nroot      1523  /opt/crowdstrike/falcond"
				case strings.Contains(cmd, "env") || strings.Contains(cmd, "printenv"):
					return "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin\nHOME=/home/operator\nHTTP_PROXY=http://proxy.corp:8080\nDB_PASSWORD=s3cret_db_pass\nAWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE"
				case strings.Contains(cmd, "arch") || strings.Contains(cmd, "dpkg --print-architecture"):
					return "x86_64"
				default:
					return fmt.Sprintf("[mock] command not recognized: %s", cmd)
				}
			},
		},
	}

	agent := newTestAgent()
	agent.tools = reconTools
	agent.maxTurns = 15 // recon needs multiple tool calls

	text, iterations, toolCalls, err := agent.run(reconPrompt)
	if err != nil {
		t.Fatalf("agent.run: %v", err)
	}

	t.Logf("=== RECON SKILL RESULT ===")
	t.Logf("iterations: %d, tool_calls: %d", iterations, toolCalls)
	t.Logf("--- response ---\n%s", text)
	t.Logf("--- end ---")

	// Verify the response contains key recon data
	checks := map[string]string{
		"hostname":        "recon-target",
		"os":              "Ubuntu",
		"kernel":          "5.15.0",
		"user":            "operator",
		"sudo":            "sudo",
		"ip address":      "10.0.2.15",
		"ssh port":        "22",
		"postgres":        "5432",
		"credential leak": "AWS_ACCESS_KEY",
	}

	lowerText := strings.ToLower(text)
	for category, keyword := range checks {
		if !strings.Contains(lowerText, strings.ToLower(keyword)) {
			t.Errorf("[%s] expected response to contain %q", category, keyword)
		}
	}

	// Security tool check: model may use "crowdstrike", "falcon", or "falcond"
	if !strings.Contains(lowerText, "crowdstrike") &&
		!strings.Contains(lowerText, "falcon") &&
		!strings.Contains(lowerText, "clamd") {
		t.Errorf("[security tool] expected response to mention CrowdStrike/falcond or ClamAV/clamd")
	}

	if toolCalls < 3 {
		t.Errorf("expected at least 3 tool calls for recon, got %d", toolCalls)
	}
	if iterations < 2 {
		t.Errorf("expected at least 2 iterations, got %d", iterations)
	}
}

func TestReactAgent_MaxTurnsLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live API test in short mode")
	}

	agent := newTestAgent()
	agent.maxTurns = 1 // only 1 turn allowed

	_, _, _, err := agent.run(
		"What is the current time? You must use the get_current_time tool.",
	)

	if err == nil {
		t.Fatal("expected error for max turns exceeded, got nil")
	}
	if !strings.Contains(err.Error(), "max turns") {
		t.Errorf("expected error containing 'max turns', got: %q", err.Error())
	}
}
