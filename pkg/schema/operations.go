package schema

type OperationSummary struct {
	Subject         string    `json:"subject,omitempty" yaml:"subject,omitempty"`
	From            *Address  `json:"from,omitempty" yaml:"from,omitempty"`
	To              []Address `json:"to,omitempty" yaml:"to,omitempty"`
	Cc              []Address `json:"cc,omitempty" yaml:"cc,omitempty"`
	BccCount        int       `json:"bcc_count,omitempty" yaml:"bcc_count,omitempty"`
	AttachmentCount int       `json:"attachment_count,omitempty" yaml:"attachment_count,omitempty"`
}

type OperationIntentResult struct {
	Status         string           `json:"status" yaml:"status"`
	IntentID       string           `json:"intent_id" yaml:"intent_id"`
	Operation      string           `json:"operation" yaml:"operation"`
	Account        string           `json:"account,omitempty" yaml:"account,omitempty"`
	MessageID      string           `json:"message_id,omitempty" yaml:"message_id,omitempty"`
	OperationsPath string           `json:"operations_path,omitempty" yaml:"operations_path,omitempty"`
	ConfirmCommand string           `json:"confirm_command,omitempty" yaml:"confirm_command,omitempty"`
	Summary        OperationSummary `json:"summary" yaml:"summary"`
}

type OperationLogEntry struct {
	ID        string            `json:"id" yaml:"id"`
	IntentID  string            `json:"intent_id,omitempty" yaml:"intent_id,omitempty"`
	Operation string            `json:"operation" yaml:"operation"`
	Status    string            `json:"status" yaml:"status"`
	Account   string            `json:"account,omitempty" yaml:"account,omitempty"`
	MessageID string            `json:"message_id,omitempty" yaml:"message_id,omitempty"`
	CreatedAt string            `json:"created_at" yaml:"created_at"`
	Summary   *OperationSummary `json:"summary,omitempty" yaml:"summary,omitempty"`
	Error     *SendError        `json:"error,omitempty" yaml:"error,omitempty"`
}

type OperationListResult struct {
	Operations []OperationLogEntry `json:"operations" yaml:"operations"`
}
