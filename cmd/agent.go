package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

type agentDoctorResult struct {
	Status     string              `json:"status"`
	MailCLIBin string              `json:"mailcli_bin"`
	Agents     []agentInstallState `json:"agents"`
}

type agentInstallState struct {
	Name             string   `json:"name"`
	Detected         bool     `json:"detected"`
	Path             string   `json:"path,omitempty"`
	MCPServerName    string   `json:"mcp_server_name"`
	ConfigureCommand []string `json:"configure_command,omitempty"`
	Status           string   `json:"status"`
	Message          string   `json:"message,omitempty"`
}

type agentConfigureResult struct {
	Status string              `json:"status"`
	Agents []agentInstallState `json:"agents"`
}

var execCommandContextFunc = exec.CommandContext

func newAgentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Detect and configure local AI agent integrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newAgentDoctorCmd())
	cmd.AddCommand(newAgentConfigureCmd())
	return cmd
}

func newAgentDoctorCmd() *cobra.Command {
	var mailcliBin string

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Detect installed AI agents and show MailCLI MCP setup commands",
		RunE: func(cmd *cobra.Command, args []string) error {
			bin, err := resolveMailCLIBin(mailcliBin)
			if err != nil {
				return err
			}
			result := agentDoctorResult{
				Status:     "ready",
				MailCLIBin: bin,
				Agents:     detectAgents(bin),
			}
			if countDetectedAgents(result.Agents) == 0 {
				result.Status = "no_agents_detected"
			}
			return writeJSON(cmd.OutOrStdout(), result)
		},
	}

	cmd.Flags().StringVar(&mailcliBin, "mailcli-bin", "", "mailcli binary path used in generated agent commands")
	return cmd
}

func newAgentConfigureCmd() *cobra.Command {
	var (
		mailcliBin string
		agents     []string
		dryRun     bool
	)

	cmd := &cobra.Command{
		Use:   "configure",
		Short: "Register MailCLI as an MCP server in detected local AI agents",
		RunE: func(cmd *cobra.Command, args []string) error {
			bin, err := resolveMailCLIBin(mailcliBin)
			if err != nil {
				return err
			}

			detected := filterAgentStates(detectAgents(bin), agents)
			result := agentConfigureResult{
				Status: "configured",
				Agents: detected,
			}
			if len(detected) == 0 {
				result.Status = "no_agents_selected"
				return writeJSON(cmd.OutOrStdout(), result)
			}

			for i := range detected {
				if !detected[i].Detected {
					detected[i].Status = "skipped"
					detected[i].Message = "agent command not found"
					result.Status = "partial"
					continue
				}
				if dryRun {
					detected[i].Status = "dry_run"
					continue
				}
				if err := runConfigureCommand(cmd.Context(), detected[i].ConfigureCommand); err != nil {
					detected[i].Status = "error"
					detected[i].Message = err.Error()
					result.Status = "partial"
					continue
				}
				detected[i].Status = "configured"
			}
			result.Agents = detected
			return writeJSON(cmd.OutOrStdout(), result)
		},
	}

	cmd.Flags().StringVar(&mailcliBin, "mailcli-bin", "", "mailcli binary path to register")
	cmd.Flags().StringArrayVar(&agents, "agent", nil, "agent to configure: codex or claude (repeatable; default: all detected)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print planned configuration without changing agent settings")
	return cmd
}

func resolveMailCLIBin(explicit string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		return strings.TrimSpace(explicit), nil
	}
	if path, err := exec.LookPath("mailcli"); err == nil {
		return path, nil
	}
	path, err := os.Executable()
	if err == nil {
		return path, nil
	}
	return "", fmt.Errorf("mailcli binary not found; pass --mailcli-bin")
}

func detectAgents(mailcliBin string) []agentInstallState {
	return []agentInstallState{
		detectCodex(mailcliBin),
		detectClaude(mailcliBin),
	}
}

func detectCodex(mailcliBin string) agentInstallState {
	state := agentInstallState{
		Name:          "codex",
		MCPServerName: "mailcli",
		Status:        "missing",
	}
	if path, err := exec.LookPath("codex"); err == nil {
		state.Detected = true
		state.Path = path
		state.Status = "ready"
		state.ConfigureCommand = []string{"codex", "mcp", "add", "mailcli", "--", mailcliBin, "mcp", "serve"}
	}
	return state
}

func detectClaude(mailcliBin string) agentInstallState {
	state := agentInstallState{
		Name:          "claude",
		MCPServerName: "mailcli",
		Status:        "missing",
	}
	if path, err := exec.LookPath("claude"); err == nil {
		state.Detected = true
		state.Path = path
		state.Status = "ready"
		state.ConfigureCommand = []string{"claude", "mcp", "add", "--scope", "user", "mailcli", "--", mailcliBin, "mcp", "serve"}
	}
	return state
}

func countDetectedAgents(states []agentInstallState) int {
	count := 0
	for _, state := range states {
		if state.Detected {
			count++
		}
	}
	return count
}

func filterAgentStates(states []agentInstallState, names []string) []agentInstallState {
	if len(names) == 0 {
		filtered := make([]agentInstallState, 0, len(states))
		for _, state := range states {
			if state.Detected {
				filtered = append(filtered, state)
			}
		}
		return filtered
	}

	wanted := make(map[string]bool, len(names))
	for _, name := range names {
		wanted[strings.ToLower(strings.TrimSpace(name))] = true
	}

	filtered := make([]agentInstallState, 0, len(states))
	for _, state := range states {
		if wanted[state.Name] {
			filtered = append(filtered, state)
		}
	}
	return filtered
}

func runConfigureCommand(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("empty configure command")
	}
	cmd := execCommandContextFunc(ctx, args[0], args[1:]...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		text := strings.TrimSpace(string(out))
		if text == "" {
			return err
		}
		return fmt.Errorf("%w: %s", err, text)
	}
	return nil
}
