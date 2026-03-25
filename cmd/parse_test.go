package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestRootCommandExists(t *testing.T) {
	cmd := NewRootCmd()
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected root command to execute without error, got %v", err)
	}
}

func TestParseCommandReadsFile(t *testing.T) {
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"parse", "../testdata/emails/plaintext.eml"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected parse command to succeed: %v", err)
	}

	if !strings.Contains(out.String(), "\"content\"") {
		t.Fatalf("expected json output")
	}
}

func TestParseCommandReadsStdin(t *testing.T) {
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetIn(strings.NewReader(loadFixture(t, "../testdata/emails/plaintext.eml")))
	cmd.SetArgs([]string{"parse", "-"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected stdin parse to succeed: %v", err)
	}

	if !strings.Contains(out.String(), "\"content\"") {
		t.Fatalf("expected json output")
	}
}

func loadFixture(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return string(data)
}
