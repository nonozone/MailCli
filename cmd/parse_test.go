package cmd

import "testing"

func TestRootCommandExists(t *testing.T) {
	cmd := NewRootCmd()
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected root command to execute without error, got %v", err)
	}
}
