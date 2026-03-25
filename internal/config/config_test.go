package config

import "testing"

func TestConfigRoundTrip(t *testing.T) {
	cfg := Config{
		CurrentAccount: "local",
		Accounts: []AccountConfig{
			{Name: "local", Driver: "imap"},
		},
	}

	data, err := Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}

	got, err := Unmarshal(data)
	if err != nil {
		t.Fatal(err)
	}

	if got.CurrentAccount != "local" {
		t.Fatalf("expected round-trip config")
	}
}
