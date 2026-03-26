package schema

type SearchQuery struct {
	Query   string `json:"query,omitempty" yaml:"query,omitempty"`
	Mailbox string `json:"mailbox,omitempty" yaml:"mailbox,omitempty"`
	Limit   int    `json:"limit,omitempty" yaml:"limit,omitempty"`
}

type MessageMetaSummary struct {
	ID      string `json:"id,omitempty" yaml:"id,omitempty"`
	From    string `json:"from,omitempty" yaml:"from,omitempty"`
	Subject string `json:"subject,omitempty" yaml:"subject,omitempty"`
	Date    string `json:"date,omitempty" yaml:"date,omitempty"`
}
