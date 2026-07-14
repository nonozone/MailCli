package cmd

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
)

type mcpRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type mcpResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *mcpError       `json:"error,omitempty"`
}

type mcpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type mcpTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

type mcpToolCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

type mcpToolResult struct {
	Content []mcpContent `json:"content"`
	IsError bool         `json:"isError,omitempty"`
}

type mcpContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func newMCPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Serve MailCLI tools through the Model Context Protocol",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newMCPServeCmd())
	return cmd
}

func newMCPServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run a stdio MCP server for safe MailCLI read and setup tools",
		RunE: func(cmd *cobra.Command, args []string) error {
			return serveMCP(cmd.Context(), cmd.InOrStdin(), cmd.OutOrStdout())
		},
	}
	return cmd
}

func serveMCP(ctx context.Context, in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	encoder := json.NewEncoder(out)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var req mcpRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			if err := encoder.Encode(mcpResponse{
				JSONRPC: "2.0",
				Error:   &mcpError{Code: -32700, Message: "parse error"},
			}); err != nil {
				return err
			}
			continue
		}
		if len(req.ID) == 0 {
			continue
		}
		resp := handleMCPRequest(ctx, req)
		if err := encoder.Encode(resp); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func handleMCPRequest(ctx context.Context, req mcpRequest) mcpResponse {
	switch req.Method {
	case "initialize":
		return mcpResult(req.ID, map[string]any{
			"protocolVersion": "2025-06-18",
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
			"serverInfo": map[string]string{
				"name":    "mailcli",
				"version": Version,
			},
		})
	case "ping":
		return mcpResult(req.ID, map[string]any{})
	case "tools/list":
		return mcpResult(req.ID, map[string]any{
			"tools": safeMCPTools(),
		})
	case "tools/call":
		var params mcpToolCallParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return mcpProtocolError(req.ID, -32602, "invalid tools/call params")
		}
		result := callMCPTool(ctx, params)
		return mcpResult(req.ID, result)
	default:
		return mcpProtocolError(req.ID, -32601, "method not found")
	}
}

func mcpResult(id json.RawMessage, result any) mcpResponse {
	return mcpResponse{JSONRPC: "2.0", ID: id, Result: result}
}

func mcpProtocolError(id json.RawMessage, code int, message string) mcpResponse {
	return mcpResponse{JSONRPC: "2.0", ID: id, Error: &mcpError{Code: code, Message: message}}
}

func safeMCPTools() []mcpTool {
	return []mcpTool{
		{
			Name:        "mailcli_parse",
			Description: "Parse a local .eml file into a StandardMessage JSON document.",
			InputSchema: objectSchema(map[string]any{
				"file": stringSchema("Local .eml file path"),
			}, []string{"file"}),
		},
		{
			Name:        "mailcli_list",
			Description: "List message metadata directly from the configured mailbox.",
			InputSchema: objectSchema(commonMailboxProperties(map[string]any{
				"limit": integerSchema("Maximum messages to list"),
			}), nil),
		},
		{
			Name:        "mailcli_get",
			Description: "Fetch and parse a single message by Message-ID, UID, or sequence number.",
			InputSchema: objectSchema(commonMailboxProperties(map[string]any{
				"id": stringSchema("Message-ID, IMAP UID, or sequence number"),
			}), []string{"id"}),
		},
		{
			Name:        "mailcli_sync",
			Description: "Fetch recent messages into the local index for search and thread workflows.",
			InputSchema: objectSchema(commonMailboxProperties(map[string]any{
				"index":   stringSchema("Local index path"),
				"limit":   integerSchema("Maximum messages to sync"),
				"since":   stringSchema("RFC3339 start date"),
				"before":  stringSchema("RFC3339 end date"),
				"refresh": booleanSchema("Re-fetch messages already in the index"),
			}), nil),
		},
		{
			Name:        "mailcli_search",
			Description: "Search local indexed messages.",
			InputSchema: objectSchema(commonIndexProperties(map[string]any{
				"query":  stringSchema("Search query"),
				"limit":  integerSchema("Maximum results"),
				"full":   booleanSchema("Return full message content"),
				"since":  stringSchema("RFC3339 start date"),
				"before": stringSchema("RFC3339 end date"),
			}), []string{"query"}),
		},
		{
			Name:        "mailcli_threads",
			Description: "List local thread summaries from indexed messages.",
			InputSchema: objectSchema(commonIndexProperties(map[string]any{
				"query":     stringSchema("Optional thread query"),
				"category":  stringSchema("Filter by category"),
				"action":    stringSchema("Filter by action type"),
				"has_codes": booleanSchema("Only return threads with extracted codes"),
				"limit":     integerSchema("Maximum threads"),
				"since":     stringSchema("RFC3339 start date"),
				"before":    stringSchema("RFC3339 end date"),
			}), nil),
		},
		{
			Name:        "mailcli_thread",
			Description: "Return all indexed messages in a thread.",
			InputSchema: objectSchema(commonIndexProperties(map[string]any{
				"thread_id": stringSchema("Thread ID from mailcli_threads"),
				"limit":     integerSchema("Maximum messages"),
			}), []string{"thread_id"}),
		},
		{
			Name:        "mailcli_triage_message",
			Description: "Build deterministic triage evidence for one local .eml message without guessing priority or reply state.",
			InputSchema: objectSchema(map[string]any{
				"file": stringSchema("Local .eml file path"),
			}, []string{"file"}),
		},
		{
			Name:        "mailcli_triage_thread",
			Description: "Build deterministic triage evidence for a complete local thread, preserving one compact fact entry per message.",
			InputSchema: objectSchema(commonIndexProperties(map[string]any{
				"thread_id": stringSchema("Thread ID from mailcli_threads"),
			}), []string{"thread_id"}),
		},
		{
			Name:        "mailcli_config_doctor",
			Description: "Diagnose local MailCLI account configuration without connecting to mailbox servers.",
			InputSchema: objectSchema(map[string]any{
				"config": stringSchema("MailCLI config path"),
			}, nil),
		},
		{
			Name:        "mailcli_config_capabilities",
			Description: "Print machine-readable capabilities for a configured account.",
			InputSchema: objectSchema(map[string]any{
				"config":  stringSchema("MailCLI config path"),
				"account": stringSchema("Account name"),
			}, nil),
		},
	}
}

func callMCPTool(ctx context.Context, params mcpToolCallParams) mcpToolResult {
	args, err := mcpToolCommandArgs(params.Name, params.Arguments)
	if err != nil {
		return mcpToolError(err)
	}

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd := NewRootCmd()
	cmd.SetContext(ctx)
	cmd.SetOut(&out)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		text := strings.TrimSpace(stderr.String())
		if text == "" {
			text = err.Error()
		}
		return mcpToolError(fmt.Errorf("%s", text))
	}
	return mcpToolResult{
		Content: []mcpContent{{Type: "text", Text: out.String()}},
	}
}

func mcpToolError(err error) mcpToolResult {
	return mcpToolResult{
		IsError: true,
		Content: []mcpContent{{
			Type: "text",
			Text: err.Error(),
		}},
	}
}

func mcpToolCommandArgs(name string, input map[string]any) ([]string, error) {
	switch name {
	case "mailcli_parse":
		file := stringArg(input, "file")
		if file == "" {
			return nil, fmt.Errorf("file is required")
		}
		return []string{"parse", "--format", "json", file}, nil
	case "mailcli_list":
		args := []string{"list", "--format", "json"}
		args = appendCommonMailboxArgs(args, input)
		args = appendOptionalIntFlag(args, "--limit", input, "limit")
		return args, nil
	case "mailcli_get":
		id := stringArg(input, "id")
		if id == "" {
			return nil, fmt.Errorf("id is required")
		}
		args := []string{"get"}
		args = appendCommonMailboxArgs(args, input)
		return append(args, id), nil
	case "mailcli_sync":
		args := []string{"sync"}
		args = appendCommonMailboxArgs(args, input)
		args = appendOptionalStringFlag(args, "--index", input, "index")
		args = appendOptionalStringFlag(args, "--since", input, "since")
		args = appendOptionalStringFlag(args, "--before", input, "before")
		args = appendOptionalIntFlag(args, "--limit", input, "limit")
		args = appendOptionalBoolFlag(args, "--refresh", input, "refresh")
		return args, nil
	case "mailcli_search":
		query := stringArg(input, "query")
		if query == "" {
			return nil, fmt.Errorf("query is required")
		}
		args := []string{"search"}
		args = appendCommonIndexArgs(args, input)
		args = appendOptionalStringFlag(args, "--since", input, "since")
		args = appendOptionalStringFlag(args, "--before", input, "before")
		args = appendOptionalIntFlag(args, "--limit", input, "limit")
		args = appendOptionalBoolFlag(args, "--full", input, "full")
		return append(args, query), nil
	case "mailcli_threads":
		args := []string{"threads"}
		args = appendCommonIndexArgs(args, input)
		args = appendOptionalStringFlag(args, "--category", input, "category")
		args = appendOptionalStringFlag(args, "--action", input, "action")
		args = appendOptionalStringFlag(args, "--since", input, "since")
		args = appendOptionalStringFlag(args, "--before", input, "before")
		args = appendOptionalIntFlag(args, "--limit", input, "limit")
		args = appendOptionalBoolFlag(args, "--has-codes", input, "has_codes")
		if query := stringArg(input, "query"); query != "" {
			args = append(args, query)
		}
		return args, nil
	case "mailcli_thread":
		threadID := stringArg(input, "thread_id")
		if threadID == "" {
			return nil, fmt.Errorf("thread_id is required")
		}
		args := []string{"thread"}
		args = appendCommonIndexArgs(args, input)
		args = appendOptionalIntFlag(args, "--limit", input, "limit")
		return append(args, threadID), nil
	case "mailcli_triage_message":
		file := stringArg(input, "file")
		if file == "" {
			return nil, fmt.Errorf("file is required")
		}
		return []string{"triage", "message", file}, nil
	case "mailcli_triage_thread":
		threadID := stringArg(input, "thread_id")
		if threadID == "" {
			return nil, fmt.Errorf("thread_id is required")
		}
		args := []string{"triage", "thread"}
		args = appendCommonIndexArgs(args, input)
		return append(args, threadID), nil
	case "mailcli_config_doctor":
		args := []string{"config", "doctor"}
		args = appendOptionalStringFlag(args, "--config", input, "config")
		return args, nil
	case "mailcli_config_capabilities":
		args := []string{"config", "capabilities"}
		args = appendOptionalStringFlag(args, "--config", input, "config")
		args = appendOptionalStringFlag(args, "--account", input, "account")
		return args, nil
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

func commonMailboxProperties(extra map[string]any) map[string]any {
	props := map[string]any{
		"config":  stringSchema("MailCLI config path"),
		"account": stringSchema("Account name"),
		"mailbox": stringSchema("Mailbox name"),
	}
	for key, value := range extra {
		props[key] = value
	}
	return props
}

func commonIndexProperties(extra map[string]any) map[string]any {
	props := map[string]any{
		"index":   stringSchema("Local index path"),
		"account": stringSchema("Account name"),
		"mailbox": stringSchema("Mailbox name"),
	}
	for key, value := range extra {
		props[key] = value
	}
	return props
}

func objectSchema(properties map[string]any, required []string) map[string]any {
	schema := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func stringSchema(description string) map[string]any {
	return map[string]any{"type": "string", "description": description}
}

func integerSchema(description string) map[string]any {
	return map[string]any{"type": "integer", "description": description}
}

func booleanSchema(description string) map[string]any {
	return map[string]any{"type": "boolean", "description": description}
}

func appendCommonMailboxArgs(args []string, input map[string]any) []string {
	args = appendOptionalStringFlag(args, "--config", input, "config")
	args = appendOptionalStringFlag(args, "--account", input, "account")
	args = appendOptionalStringFlag(args, "--mailbox", input, "mailbox")
	return args
}

func appendCommonIndexArgs(args []string, input map[string]any) []string {
	args = appendOptionalStringFlag(args, "--index", input, "index")
	args = appendOptionalStringFlag(args, "--account", input, "account")
	args = appendOptionalStringFlag(args, "--mailbox", input, "mailbox")
	return args
}

func appendOptionalStringFlag(args []string, flag string, input map[string]any, key string) []string {
	value := stringArg(input, key)
	if value == "" {
		return args
	}
	return append(args, flag, value)
}

func appendOptionalIntFlag(args []string, flag string, input map[string]any, key string) []string {
	value, ok := intArg(input, key)
	if !ok {
		return args
	}
	return append(args, flag, fmt.Sprintf("%d", value))
}

func appendOptionalBoolFlag(args []string, flag string, input map[string]any, key string) []string {
	value, ok := boolArg(input, key)
	if !ok || !value {
		return args
	}
	return append(args, flag)
}

func stringArg(input map[string]any, key string) string {
	if input == nil {
		return ""
	}
	value, ok := input[key]
	if !ok || value == nil {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func intArg(input map[string]any, key string) (int, bool) {
	if input == nil {
		return 0, false
	}
	value, ok := input[key]
	if !ok || value == nil {
		return 0, false
	}
	switch v := value.(type) {
	case float64:
		return int(v), true
	case int:
		return v, true
	default:
		return 0, false
	}
}

func boolArg(input map[string]any, key string) (bool, bool) {
	if input == nil {
		return false, false
	}
	value, ok := input[key]
	if !ok || value == nil {
		return false, false
	}
	v, ok := value.(bool)
	return v, ok
}
