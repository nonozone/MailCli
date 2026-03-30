package schema

type SearchQuery struct {
	Query   string `json:"query,omitempty" yaml:"query,omitempty"`
	Mailbox string `json:"mailbox,omitempty" yaml:"mailbox,omitempty"`
	Limit   int    `json:"limit,omitempty" yaml:"limit,omitempty"`
	// Since and Before are optional RFC3339 timestamps for date-range filtering.
	// Drivers that support server-side filtering (e.g. IMAP) will use them in
	// the search criteria; otherwise filtering is applied client-side.
	Since  string `json:"since,omitempty" yaml:"since,omitempty"`
	Before string `json:"before,omitempty" yaml:"before,omitempty"`
}

type MessageMetaSummary struct {
	ID      string `json:"id,omitempty" yaml:"id,omitempty"`
	From    string `json:"from,omitempty" yaml:"from,omitempty"`
	Subject string `json:"subject,omitempty" yaml:"subject,omitempty"`
	Date    string `json:"date,omitempty" yaml:"date,omitempty"`
}
