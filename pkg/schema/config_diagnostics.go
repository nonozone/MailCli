package schema

type ConfigInitResult struct {
	Status     string `json:"status"`
	ConfigPath string `json:"config_path"`
	Account    string `json:"account"`
	Driver     string `json:"driver"`
}

type ConfigDiagnostics struct {
	ConfigPath string              `json:"config_path"`
	Status     string              `json:"status"`
	Accounts   []AccountDiagnostic `json:"accounts"`
	Problems   []ConfigDiagnostic  `json:"problems,omitempty"`
}

type AccountDiagnostic struct {
	Name         string              `json:"name"`
	Driver       string              `json:"driver"`
	Status       string              `json:"status"`
	Capabilities AccountCapabilities `json:"capabilities"`
	Checks       []ConfigDiagnostic  `json:"checks"`
}

type ConfigDiagnostic struct {
	Status  string `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}
