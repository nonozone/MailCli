package schema

type StandardMessage struct {
	ID           string              `json:"id" yaml:"id"`
	Meta         MessageMeta         `json:"meta" yaml:"meta"`
	Content      Content             `json:"content" yaml:"content"`
	Actions      []Action            `json:"actions,omitempty" yaml:"actions,omitempty"`
	Codes        []Code              `json:"codes,omitempty" yaml:"codes,omitempty"`
	Attachments  []InboundAttachment `json:"attachments,omitempty" yaml:"attachments,omitempty"`
	ErrorContext *ErrorContext       `json:"error_context,omitempty" yaml:"error_context,omitempty"`
	Labels       []string            `json:"labels,omitempty" yaml:"labels,omitempty"`
	TokenUsage   *TokenUsage         `json:"token_usage,omitempty" yaml:"token_usage,omitempty"`
}

// InboundAttachment describes one decoded MIME part without exposing its
// contents. PartID is the stable reference within the source RFC 822 message;
// callers must not treat Filename as a safe filesystem path.
type InboundAttachment struct {
	PartID      string `json:"part_id" yaml:"part_id"`
	Filename    string `json:"filename,omitempty" yaml:"filename,omitempty"`
	ContentType string `json:"content_type,omitempty" yaml:"content_type,omitempty"`
	SizeBytes   int64  `json:"size_bytes" yaml:"size_bytes"`
	Disposition string `json:"disposition,omitempty" yaml:"disposition,omitempty"`
	ContentID   string `json:"content_id,omitempty" yaml:"content_id,omitempty"`
	Inline      bool   `json:"inline,omitempty" yaml:"inline,omitempty"`
}

type MessageMeta struct {
	From            *Address  `json:"from,omitempty" yaml:"from,omitempty"`
	To              []Address `json:"to,omitempty" yaml:"to,omitempty"`
	Subject         string    `json:"subject,omitempty" yaml:"subject,omitempty"`
	Date            string    `json:"date,omitempty" yaml:"date,omitempty"`
	MessageID       string    `json:"message_id,omitempty" yaml:"message_id,omitempty"`
	InReplyTo       string    `json:"in_reply_to,omitempty" yaml:"in_reply_to,omitempty"`
	References      []string  `json:"references,omitempty" yaml:"references,omitempty"`
	ListUnsubscribe []string  `json:"list_unsubscribe,omitempty" yaml:"list_unsubscribe,omitempty"`
	AutoSubmitted   bool      `json:"is_auto_submitted,omitempty" yaml:"is_auto_submitted,omitempty"`
}

type Address struct {
	Name    string `json:"name,omitempty" yaml:"name,omitempty"`
	Address string `json:"address,omitempty" yaml:"address,omitempty"`
}

type Content struct {
	Format   string `json:"format,omitempty" yaml:"format,omitempty"`
	BodyMD   string `json:"body_md,omitempty" yaml:"body_md,omitempty"`
	Snippet  string `json:"snippet,omitempty" yaml:"snippet,omitempty"`
	Category string `json:"category,omitempty" yaml:"category,omitempty"`
}

type Action struct {
	Type  string `json:"type" yaml:"type"`
	Label string `json:"label,omitempty" yaml:"label,omitempty"`
	URL   string `json:"url,omitempty" yaml:"url,omitempty"`
}

type Code struct {
	Type             string `json:"type" yaml:"type"`
	Value            string `json:"value" yaml:"value"`
	Label            string `json:"label,omitempty" yaml:"label,omitempty"`
	ExpiresInSeconds int    `json:"expires_in_seconds,omitempty" yaml:"expires_in_seconds,omitempty"`
}

type ErrorContext struct {
	FailedRecipient   string `json:"failed_recipient,omitempty" yaml:"failed_recipient,omitempty"`
	StatusCode        string `json:"status_code,omitempty" yaml:"status_code,omitempty"`
	DiagnosticCode    string `json:"diagnostic_code,omitempty" yaml:"diagnostic_code,omitempty"`
	OriginalMessageID string `json:"original_message_id,omitempty" yaml:"original_message_id,omitempty"`
	OriginalSubject   string `json:"original_subject,omitempty" yaml:"original_subject,omitempty"`
}

type TokenUsage struct {
	EstimatedInputTokens int `json:"estimated_input_tokens,omitempty" yaml:"estimated_input_tokens,omitempty"`
}
