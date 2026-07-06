package agent

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strings"
)

var allowedDecisions = map[string]bool{
	"review":                  true,
	"capture_code":            true,
	"draft_reply":             true,
	"escalate_delivery_error": true,
}

func WriteJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func RunCommand(command []string, stdin string) (string, error) {
	return RunCommandInDir(command, stdin, "")
}

func RunCommandInDir(command []string, stdin, cwd string) (string, error) {
	if len(command) == 0 || strings.TrimSpace(command[0]) == "" {
		return "", errors.New("command is required")
	}

	cmd := exec.Command(command[0], command[1:]...)
	if cwd != "" {
		cmd.Dir = cwd
	}
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = strings.TrimSpace(stdout.String())
		}
		if message == "" {
			message = err.Error()
		}
		return "", errors.New(message)
	}
	return stdout.String(), nil
}

func RunJSON(command []string, stdin string) (any, error) {
	return RunJSONInDir(command, stdin, "")
}

func RunJSONInDir(command []string, stdin, cwd string) (any, error) {
	output, err := RunCommandInDir(command, stdin, cwd)
	if err != nil {
		return nil, err
	}

	var value any
	if err := json.Unmarshal([]byte(output), &value); err != nil {
		return nil, fmt.Errorf("expected JSON output from %s: %w", strings.Join(command, " "), err)
	}
	return value, nil
}

func ParseExternalAnalysis(output string) (map[string]any, error) {
	var value any
	if err := json.Unmarshal([]byte(output), &value); err != nil {
		return nil, errors.New("external provider returned invalid JSON")
	}

	analysis, ok := value.(map[string]any)
	if !ok {
		return nil, errors.New("external provider must return a JSON object")
	}

	if err := ValidateAnalysis(analysis, "external provider"); err != nil {
		return nil, err
	}
	return analysis, nil
}

func ValidateAnalysis(analysis map[string]any, label string) error {
	rawDecision, ok := analysis["decision"].(string)
	decision := strings.TrimSpace(rawDecision)
	if !ok || decision == "" {
		return fmt.Errorf("%s response must include a non-empty decision", label)
	}
	if !allowedDecisions[decision] {
		allowed := make([]string, 0, len(allowedDecisions))
		for decision := range allowedDecisions {
			allowed = append(allowed, decision)
		}
		sort.Strings(allowed)
		return fmt.Errorf("%s decision must be one of: %s", label, strings.Join(allowed, ", "))
	}

	if replyText, ok := analysis["reply_text"]; ok && replyText != nil {
		if _, ok := replyText.(string); !ok {
			return fmt.Errorf("%s reply_text must be a string when present", label)
		}
	}
	return nil
}

func AnalyzeMessage(message map[string]any, wantsReply bool) map[string]any {
	content := MapValue(message, "content")
	snippet := StringValue(content, "snippet")
	if snippet == "" {
		snippet = StringValue(content, "body_md")
	}

	codes := SliceValue(message, "codes")
	if len(codes) > 0 {
		summary := fmt.Sprintf("Verification email with %d code(s).", len(codes))
		if first, ok := codes[0].(map[string]any); ok {
			if expires, ok := first["expires_in_seconds"]; ok && expires != nil {
				summary += fmt.Sprintf(" First code expires in %s seconds.", JSONNumberString(expires))
			}
		}
		return map[string]any{
			"decision": "capture_code",
			"summary":  summary,
		}
	}

	errorContext := MapValue(message, "error_context")
	if len(errorContext) > 0 {
		summary := StringValue(errorContext, "diagnostic_code")
		if summary == "" {
			summary = StringValue(errorContext, "status_code")
		}
		if summary == "" {
			summary = "Delivery error detected."
		}
		return map[string]any{
			"decision": "escalate_delivery_error",
			"summary":  summary,
		}
	}

	if wantsReply {
		return map[string]any{
			"decision": "draft_reply",
			"summary":  snippet,
		}
	}

	return map[string]any{
		"decision": "review",
		"summary":  snippet,
	}
}

func AnalyzeTemplatePayload(payload map[string]any) map[string]any {
	message := MapValue(payload, "message")
	if len(message) == 0 {
		latest := MapValue(payload, "latest_message")
		message = MapValue(latest, "message")
	}

	wantsReply, _ := payload["wants_reply"].(bool)
	actions := SliceValue(message, "actions")
	unsubscribeCount := 0
	for _, item := range actions {
		action, ok := item.(map[string]any)
		if ok && StringValue(action, "type") == "unsubscribe" {
			unsubscribeCount++
		}
	}

	if unsubscribeCount > 0 && !wantsReply {
		return map[string]any{
			"decision": "review",
			"summary":  fmt.Sprintf("Subscription email with %d unsubscribe action(s).", unsubscribeCount),
		}
	}

	analysis := AnalyzeMessage(message, wantsReply)
	if wantsReply && analysis["decision"] == "draft_reply" {
		analysis["reply_text"] = "Thanks for your email."
	}
	return analysis
}

func MapValue(parent map[string]any, key string) map[string]any {
	if parent == nil {
		return map[string]any{}
	}
	if value, ok := parent[key].(map[string]any); ok {
		return value
	}
	return map[string]any{}
}

func SliceValue(parent map[string]any, key string) []any {
	if parent == nil {
		return nil
	}
	if value, ok := parent[key].([]any); ok {
		return value
	}
	return nil
}

func StringValue(parent map[string]any, key string) string {
	if parent == nil {
		return ""
	}
	value, ok := parent[key].(string)
	if !ok {
		return ""
	}
	return value
}

func StringSlice(value any) []any {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]any, 0, len(items))
	for _, item := range items {
		if text, ok := item.(string); ok {
			out = append(out, text)
		}
	}
	return out
}

func JSONNumberString(value any) string {
	switch v := value.(type) {
	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%g", v)
	case json.Number:
		return v.String()
	case string:
		return v
	default:
		return fmt.Sprint(v)
	}
}

func MarshalCompact(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
