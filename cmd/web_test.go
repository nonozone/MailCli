package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
)

func TestWebRejectsNonLocalHost(t *testing.T) {
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"web", "--host", "0.0.0.0", "--no-open"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected non-local host to be rejected")
	}
	if !strings.Contains(err.Error(), "host must be localhost") {
		t.Fatalf("expected localhost error, got %v", err)
	}
}

func TestWebUsesDefaultLocalPort(t *testing.T) {
	original := runWebServerFunc
	t.Cleanup(func() { runWebServerFunc = original })
	runWebServerFunc = func(ctx context.Context, opts webRunOptions, out io.Writer) error {
		if opts.Host != "127.0.0.1" {
			t.Fatalf("expected default localhost host, got %q", opts.Host)
		}
		if opts.Port != 5566 {
			t.Fatalf("expected default local port 5566, got %d", opts.Port)
		}
		_, err := fmt.Fprintln(out, "MailCLI Web: http://127.0.0.1:5566/?token=test")
		return err
	}

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"web", "--no-open"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected web command to start and stop cleanly in test mode: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "http://127.0.0.1:5566/") || !strings.Contains(out.String(), "token=") {
		t.Fatalf("expected local URL with token, got %s", out.String())
	}
}

func TestWebAllowsRandomPort(t *testing.T) {
	original := runWebServerFunc
	t.Cleanup(func() { runWebServerFunc = original })
	runWebServerFunc = func(ctx context.Context, opts webRunOptions, out io.Writer) error {
		if opts.Port != 0 {
			t.Fatalf("expected requested random port, got %d", opts.Port)
		}
		_, err := fmt.Fprintln(out, "MailCLI Web: http://127.0.0.1:49231/?token=test")
		return err
	}

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"web", "--port", "0", "--no-open"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected web command to start and stop cleanly in test mode: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "http://127.0.0.1:49231/") || !strings.Contains(out.String(), "token=") {
		t.Fatalf("expected local URL with token, got %s", out.String())
	}
}
