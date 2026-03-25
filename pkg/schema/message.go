package schema

type StandardMessage struct {
	ID           string        `json:"id" yaml:"id"`
	Meta         MessageMeta   `json:"meta" yaml:"meta"`
	Content      Content       `json:"content" yaml:"content"`
	Actions      []Action      `json:"actions,omitempty" yaml:"actions,omitempty"`
	ErrorContext *ErrorContext `json:"error_context,omitempty" yaml:"error_context,omitempty"`
	Labels       []string      `json:"labels,omitempty" yaml:"labels,omitempty"`
	TokenUsage   *TokenUsage   `json:"token_usage,omitempty" yaml:"token_usage,omitempty"`
}

type MessageMeta struct {
	Subject string `json:"subject,omitempty" yaml:"subject,omitempty"`
}

type Address struct {
	Name    string `json:"name,omitempty" yaml:"name,omitempty"`
	Address string `json:"address,omitempty" yaml:"address,omitempty"`
}

type Content struct {
	Format  string `json:"format,omitempty" yaml:"format,omitempty"`
	BodyMD  string `json:"body_md,omitempty" yaml:"body_md,omitempty"`
	Snippet string `json:"snippet,omitempty" yaml:"snippet,omitempty"`
}

type Action struct {
	Type string `json:"type" yaml:"type"`
	URL  string `json:"url,omitempty" yaml:"url,omitempty"`
}

type ErrorContext struct {
	StatusCode string `json:"status_code,omitempty" yaml:"status_code,omitempty"`
}

type TokenUsage struct {
	EstimatedInputTokens int `json:"estimated_input_tokens,omitempty" yaml:"estimated_input_tokens,omitempty"`
}
