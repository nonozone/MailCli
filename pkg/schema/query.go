package schema

type SearchQuery struct {
	Query string `json:"query,omitempty" yaml:"query,omitempty"`
}

type MessageMetaSummary struct {
	ID      string `json:"id,omitempty" yaml:"id,omitempty"`
	Subject string `json:"subject,omitempty" yaml:"subject,omitempty"`
}
