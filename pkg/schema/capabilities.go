package schema

type AccountCapabilities struct {
	Account       string                         `json:"account" yaml:"account"`
	Driver        string                         `json:"driver" yaml:"driver"`
	Mailbox       string                         `json:"mailbox,omitempty" yaml:"mailbox,omitempty"`
	Capabilities  MailCapabilities               `json:"capabilities" yaml:"capabilities"`
	Configuration AccountCapabilityConfiguration `json:"configuration" yaml:"configuration"`
}

type MailCapabilities struct {
	List       bool `json:"list" yaml:"list"`
	FetchRaw   bool `json:"fetch_raw" yaml:"fetch_raw"`
	Search     bool `json:"search" yaml:"search"`
	Threads    bool `json:"threads" yaml:"threads"`
	Watch      bool `json:"watch" yaml:"watch"`
	Send       bool `json:"send" yaml:"send"`
	Reply      bool `json:"reply" yaml:"reply"`
	Delete     bool `json:"delete" yaml:"delete"`
	Move       bool `json:"move" yaml:"move"`
	MarkRead   bool `json:"mark_read" yaml:"mark_read"`
	LocalIndex bool `json:"local_index" yaml:"local_index"`
}

type AccountCapabilityConfiguration struct {
	InboundConfigured  bool `json:"inbound_configured" yaml:"inbound_configured"`
	OutboundConfigured bool `json:"outbound_configured" yaml:"outbound_configured"`
	UsesLocalStorage   bool `json:"uses_local_storage" yaml:"uses_local_storage"`
}
