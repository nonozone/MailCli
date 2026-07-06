package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/nonozone/MailCli/examples/internal/agent"
)

const systemPrompt = `You are an email analysis provider for MailCLI.

Return JSON that matches the required schema.
Choose exactly one decision from:
- review
- capture_code
- draft_reply
- escalate_delivery_error

Use:
- capture_code when the message includes verification codes in message.codes
- escalate_delivery_error when the message includes error_context
- draft_reply only when wants_reply is true and a short safe reply is appropriate
- review otherwise

Keep summary short. Only include reply_text when decision is draft_reply.`

var outputSchema = map[string]any{
	"type":                 "object",
	"additionalProperties": false,
	"properties": map[string]any{
		"decision": map[string]any{
			"type": "string",
			"enum": []string{
				"capture_code",
				"draft_reply",
				"escalate_delivery_error",
				"review",
			},
		},
		"summary": map[string]any{
			"type": "string",
		},
		"reply_text": map[string]any{
			"type": "string",
		},
	},
	"required": []string{"decision", "summary"},
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	if apiKey == "" {
		return fmt.Errorf("OPENAI_API_KEY is required")
	}

	var payload any
	if err := json.NewDecoder(os.Stdin).Decode(&payload); err != nil {
		return err
	}
	payload = normalizePayload(payload)

	result, err := callResponsesAPI(apiKey, payload)
	if err != nil {
		return err
	}
	if err := validateResult(result); err != nil {
		return err
	}
	return agent.WriteJSON(os.Stdout, result)
}

func callResponsesAPI(apiKey string, payload any) (map[string]any, error) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	model := strings.TrimSpace(os.Getenv("OPENAI_MODEL"))
	if model == "" {
		model = "gpt-5-mini"
	}

	requestBody := map[string]any{
		"model": model,
		"input": []map[string]any{
			{
				"role": "system",
				"content": []map[string]any{
					{"type": "input_text", "text": systemPrompt},
				},
			},
			{
				"role": "user",
				"content": []map[string]any{
					{"type": "input_text", "text": string(payloadJSON)},
				},
			},
		},
		"text": map[string]any{
			"format": map[string]any{
				"type":   "json_schema",
				"name":   "mailcli_agent_decision",
				"schema": outputSchema,
				"strict": true,
			},
		},
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, responsesURL(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := strings.TrimSpace(string(responseBody))
		if message == "" {
			message = resp.Status
		}
		return nil, fmt.Errorf("OpenAI provider request failed: %s", message)
	}

	var response map[string]any
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("OpenAI provider returned invalid response JSON: %w", err)
	}

	outputText := extractOutputText(response)
	if outputText == "" {
		return nil, errors.New("OpenAI provider response did not include output text")
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(outputText), &result); err != nil {
		return nil, fmt.Errorf("OpenAI provider returned invalid JSON: %w", err)
	}
	return result, nil
}

func responsesURL() string {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("OPENAI_BASE_URL")), "/")
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	if strings.HasSuffix(base, "/responses") {
		return base
	}
	return base + "/responses"
}

func normalizePayload(payload any) any {
	object, ok := payload.(map[string]any)
	if !ok {
		return payload
	}

	message, ok := object["message"].(map[string]any)
	if ok && len(message) > 0 {
		return payload
	}

	latest, ok := object["latest_message"].(map[string]any)
	if !ok {
		return payload
	}
	latestMessage, ok := latest["message"].(map[string]any)
	if !ok || len(latestMessage) == 0 {
		return payload
	}

	normalized := make(map[string]any, len(object)+1)
	for key, value := range object {
		normalized[key] = value
	}
	normalized["message"] = latestMessage
	return normalized
}

func extractOutputText(response map[string]any) string {
	if outputText, ok := response["output_text"].(string); ok {
		return outputText
	}

	output, ok := response["output"].([]any)
	if !ok {
		return ""
	}
	for _, item := range output {
		message, ok := item.(map[string]any)
		if !ok {
			continue
		}
		content, ok := message["content"].([]any)
		if !ok {
			continue
		}
		for _, part := range content {
			contentPart, ok := part.(map[string]any)
			if !ok {
				continue
			}
			if text, ok := contentPart["text"].(string); ok && text != "" {
				return text
			}
		}
	}
	return ""
}

func validateResult(result map[string]any) error {
	if err := agent.ValidateAnalysis(result, "OpenAI provider"); err != nil {
		return err
	}
	summary, ok := result["summary"].(string)
	if !ok || strings.TrimSpace(summary) == "" {
		return fmt.Errorf("OpenAI provider response must include a non-empty summary")
	}
	return nil
}
