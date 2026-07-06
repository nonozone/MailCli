package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAgentDoctorReportsDetectedAgentsAndCommands(t *testing.T) {
	binDir := t.TempDir()
	writeExecutable(t, filepath.Join(binDir, "codex"))
	writeExecutable(t, filepath.Join(binDir, "claude"))
	t.Setenv("PATH", binDir)

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"agent", "doctor", "--mailcli-bin", "/usr/local/bin/mailcli"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected agent doctor to succeed: %v\n%s", err, out.String())
	}

	var result agentDoctorResult
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &result); err != nil {
		t.Fatalf("expected JSON doctor result: %v\n%s", err, out.String())
	}
	if result.Status != "ready" {
		t.Fatalf("expected ready status, got %+v", result)
	}
	if len(result.Agents) != 2 {
		t.Fatalf("expected two detected agents, got %+v", result.Agents)
	}
	if !strings.Contains(strings.Join(result.Agents[0].ConfigureCommand, " "), "mcp") {
		t.Fatalf("expected configure command to use MCP, got %+v", result.Agents)
	}
}

func TestAgentConfigureRunsSelectedAgentCommands(t *testing.T) {
	binDir := t.TempDir()
	logPath := filepath.Join(t.TempDir(), "calls.log")
	writeLoggingExecutable(t, filepath.Join(binDir, "codex"), logPath)
	writeLoggingExecutable(t, filepath.Join(binDir, "claude"), logPath)
	t.Setenv("PATH", binDir)

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"agent", "configure", "--mailcli-bin", "/usr/local/bin/mailcli", "--agent", "codex", "--agent", "claude"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected agent configure to run detected commands: %v\n%s", err, out.String())
	}

	raw, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("expected command log: %v", err)
	}
	log := string(raw)
	for _, want := range []string{
		"codex mcp add mailcli -- /usr/local/bin/mailcli mcp serve",
		"claude mcp add --scope user mailcli -- /usr/local/bin/mailcli mcp serve",
	} {
		if !strings.Contains(log, want) {
			t.Fatalf("expected log to contain %q, got:\n%s", want, log)
		}
	}
}

func writeExecutable(t *testing.T, path string) {
	t.Helper()

	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write executable: %v", err)
	}
}

func writeLoggingExecutable(t *testing.T, path, logPath string) {
	t.Helper()

	name := filepath.Base(path)
	script := "#!/bin/sh\nprintf '" + name + " %s\\n' \"$*\" >> " + shellQuote(logPath) + "\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write logging executable: %v", err)
	}
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
