package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nonozone/MailCli/internal/config"
	"github.com/nonozone/MailCli/pkg/driver"
	"github.com/nonozone/MailCli/pkg/schema"
)

type fakeSendDriver struct {
	lastRaw  []byte
	sendErr  error
	fetchErr error
}

func (f *fakeSendDriver) List(ctx context.Context, query schema.SearchQuery) ([]schema.MessageMetaSummary, error) {
	return nil, nil
}

func (f *fakeSendDriver) FetchRaw(ctx context.Context, id string) ([]byte, error) {
	return nil, f.fetchErr
}

func (f *fakeSendDriver) SendRaw(ctx context.Context, raw []byte) error {
	if f.sendErr != nil {
		return f.sendErr
	}
	f.lastRaw = append([]byte(nil), raw...)
	return nil
}

func TestSendCommandDryRunPrintsMIME(t *testing.T) {
	draftPath := writeTempFile(t, "draft.json", `{
  "from": {"name": "Nono", "address": "support@nono.im"},
  "to": [{"address": "user@example.com"}],
  "subject": "Welcome",
  "body_text": "Hello from MailCLI."
}`)

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"send", "--dry-run", draftPath})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected send dry-run to succeed: %v", err)
	}

	if !strings.Contains(out.String(), "Subject: Welcome") {
		t.Fatalf("expected MIME output")
	}
}

func TestReplyCommandDryRunPrintsThreadHeaders(t *testing.T) {
	replyPath := writeTempFile(t, "reply.json", `{
  "from": {"address": "support@nono.im"},
  "to": [{"address": "user@example.com"}],
  "subject": "Re: Question",
  "body_text": "Thanks for the email.",
  "reply_to_message_id": "<orig-123@example.com>",
  "references": ["<older-1@example.com>", "<orig-123@example.com>"]
}`)

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"reply", "--dry-run", replyPath})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected reply dry-run to succeed: %v", err)
	}

	if !strings.Contains(out.String(), "In-Reply-To: <orig-123@example.com>") {
		t.Fatalf("expected reply MIME output")
	}
}

func TestSendCommandUsesConfiguredDriver(t *testing.T) {
	restoreLoad := loadConfigFunc
	restoreDriver := driverFactoryFunc
	t.Cleanup(func() {
		loadConfigFunc = restoreLoad
		driverFactoryFunc = restoreDriver
	})

	configPath := writeTempFile(t, "config.yaml", "current_account: work\naccounts:\n  - name: work\n    driver: fake\n")
	loadConfigFunc = config.Load

	fake := &fakeSendDriver{}
	driverFactoryFunc = func(account config.AccountConfig) (driver.Driver, error) {
		return fake, nil
	}

	draftPath := writeTempFile(t, "draft.json", `{
  "account": "work",
  "from": {"address": "support@nono.im"},
  "to": [{"address": "user@example.com"}],
  "subject": "Welcome",
  "body_text": "Hello from MailCLI."
}`)

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"send", "--config", configPath, draftPath})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected send command to succeed: %v", err)
	}

	if len(fake.lastRaw) == 0 {
		t.Fatalf("expected send command to pass MIME to driver")
	}
	if !strings.Contains(out.String(), "\"ok\": true") {
		t.Fatalf("expected send result output")
	}
}

func TestSendPrepareWritesIntentWithoutSending(t *testing.T) {
	restoreLoad := loadConfigFunc
	restoreDriver := driverFactoryFunc
	t.Cleanup(func() {
		loadConfigFunc = restoreLoad
		driverFactoryFunc = restoreDriver
	})

	configPath := writeTempFile(t, "config.yaml", "current_account: work\naccounts:\n  - name: work\n    driver: fake\n    username: support@nono.im\n")
	loadConfigFunc = config.Load
	driverFactoryFunc = func(account config.AccountConfig) (driver.Driver, error) {
		t.Fatalf("prepare must not initialize or call the transport driver")
		return nil, nil
	}

	draftPath := writeTempFile(t, "draft.json", `{
  "account": "work",
  "to": [{"address": "user@example.com"}],
  "subject": "Welcome",
  "body_text": "Hello from MailCLI."
}`)
	operationsPath := filepath.Join(t.TempDir(), "operations.jsonl")

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"send", "prepare", "--config", configPath, "--operations", operationsPath, draftPath})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected send prepare to succeed: %v\n%s", err, out.String())
	}

	var result map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &result); err != nil {
		t.Fatalf("expected JSON prepare result: %v\n%s", err, out.String())
	}
	if result["status"] != "prepared" || result["operation"] != "send" || result["account"] != "work" {
		t.Fatalf("unexpected prepare result: %#v", result)
	}
	if strings.TrimSpace(result["intent_id"].(string)) == "" {
		t.Fatalf("expected intent id in prepare result: %#v", result)
	}
	confirmCommand, ok := result["confirm_command"].(string)
	if !ok || !strings.Contains(confirmCommand, "--config "+configPath) {
		t.Fatalf("expected confirm command to preserve config path, got %#v", result["confirm_command"])
	}

	rawLog, err := os.ReadFile(operationsPath)
	if err != nil {
		t.Fatalf("expected operations log to be written: %v", err)
	}
	if !strings.Contains(string(rawLog), `"status":"prepared"`) {
		t.Fatalf("expected prepared log entry, got %s", string(rawLog))
	}
	if strings.Contains(string(rawLog), "Hello from MailCLI.") {
		t.Fatalf("operation log must not store full draft body: %s", string(rawLog))
	}
}

func TestSendPrepareConfirmCommandQuotesPathsForCopyPaste(t *testing.T) {
	restoreLoad := loadConfigFunc
	restoreDriver := driverFactoryFunc
	t.Cleanup(func() {
		loadConfigFunc = restoreLoad
		driverFactoryFunc = restoreDriver
	})

	baseDir := t.TempDir()
	configDir := filepath.Join(baseDir, "config dir")
	operationsDir := filepath.Join(baseDir, "operations dir")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(operationsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("current_account: work\naccounts:\n  - name: work\n    driver: fake\n    username: support@nono.im\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	loadConfigFunc = config.Load
	driverFactoryFunc = func(account config.AccountConfig) (driver.Driver, error) {
		t.Fatalf("prepare must not initialize or call the transport driver")
		return nil, nil
	}

	draftPath := writeTempFile(t, "draft.json", `{
  "account": "work",
  "to": [{"address": "user@example.com"}],
  "subject": "Welcome",
  "body_text": "Hello from MailCLI."
}`)
	operationsPath := filepath.Join(operationsDir, "operations.jsonl")

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"send", "prepare", "--config", configPath, "--operations", operationsPath, draftPath})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected send prepare to succeed: %v\n%s", err, out.String())
	}

	var result map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &result); err != nil {
		t.Fatalf("expected JSON prepare result: %v\n%s", err, out.String())
	}
	confirmCommand := result["confirm_command"].(string)
	if !strings.Contains(confirmCommand, "--config '"+configPath+"'") {
		t.Fatalf("expected quoted config path in confirm command, got %q", confirmCommand)
	}
	if !strings.Contains(confirmCommand, "--operations '"+operationsPath+"'") {
		t.Fatalf("expected quoted operations path in confirm command, got %q", confirmCommand)
	}
}

func TestSendConfirmSendsPreparedIntentAndLogsOperation(t *testing.T) {
	restoreLoad := loadConfigFunc
	restoreDriver := driverFactoryFunc
	t.Cleanup(func() {
		loadConfigFunc = restoreLoad
		driverFactoryFunc = restoreDriver
	})

	configPath := writeTempFile(t, "config.yaml", "current_account: work\naccounts:\n  - name: work\n    driver: fake\n    username: support@nono.im\n")
	loadConfigFunc = config.Load

	draftPath := writeTempFile(t, "draft.json", `{
  "account": "work",
  "to": [{"address": "user@example.com"}],
  "subject": "Welcome",
  "body_text": "Hello from MailCLI."
}`)
	operationsPath := filepath.Join(t.TempDir(), "operations.jsonl")

	driverFactoryFunc = func(account config.AccountConfig) (driver.Driver, error) {
		t.Fatalf("prepare must not initialize or call the transport driver")
		return nil, nil
	}
	prepareCmd := NewRootCmd()
	var prepareOut bytes.Buffer
	prepareCmd.SetOut(&prepareOut)
	prepareCmd.SetErr(&prepareOut)
	prepareCmd.SetArgs([]string{"send", "prepare", "--config", configPath, "--operations", operationsPath, draftPath})
	if err := prepareCmd.Execute(); err != nil {
		t.Fatalf("expected send prepare to succeed: %v\n%s", err, prepareOut.String())
	}
	var prepareResult map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(prepareOut.Bytes()), &prepareResult); err != nil {
		t.Fatalf("expected JSON prepare result: %v\n%s", err, prepareOut.String())
	}
	intentID := prepareResult["intent_id"].(string)
	messageID := prepareResult["message_id"].(string)

	fake := &fakeSendDriver{}
	driverFactoryFunc = func(account config.AccountConfig) (driver.Driver, error) {
		return fake, nil
	}
	confirmCmd := NewRootCmd()
	var confirmOut bytes.Buffer
	confirmCmd.SetOut(&confirmOut)
	confirmCmd.SetErr(&confirmOut)
	confirmCmd.SetArgs([]string{"send", "confirm", "--config", configPath, "--operations", operationsPath, intentID})
	if err := confirmCmd.Execute(); err != nil {
		t.Fatalf("expected send confirm to succeed: %v\n%s", err, confirmOut.String())
	}

	if len(fake.lastRaw) == 0 {
		t.Fatalf("expected confirm to send prepared MIME")
	}
	rawMessage := string(fake.lastRaw)
	if !strings.Contains(rawMessage, "Hello from MailCLI.") || !strings.Contains(rawMessage, "Message-ID: "+messageID) {
		t.Fatalf("expected confirm to send the prepared intent body and message id, got %s", rawMessage)
	}

	var confirmResult map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(confirmOut.Bytes()), &confirmResult); err != nil {
		t.Fatalf("expected JSON confirm result: %v\n%s", err, confirmOut.String())
	}
	if confirmResult["ok"] != true || confirmResult["intent_id"] != intentID || strings.TrimSpace(confirmResult["operation_id"].(string)) == "" {
		t.Fatalf("unexpected confirm result: %#v", confirmResult)
	}

	rawLog, err := os.ReadFile(operationsPath)
	if err != nil {
		t.Fatalf("expected operations log to be written: %v", err)
	}
	logText := string(rawLog)
	if !strings.Contains(logText, `"status":"prepared"`) || !strings.Contains(logText, `"status":"sent"`) {
		t.Fatalf("expected prepared and sent log entries, got %s", logText)
	}
	if strings.Contains(logText, "Hello from MailCLI.") {
		t.Fatalf("operation log must not store full draft body: %s", logText)
	}
}

func TestSendConfirmRejectsAlreadySentIntent(t *testing.T) {
	restoreLoad := loadConfigFunc
	restoreDriver := driverFactoryFunc
	t.Cleanup(func() {
		loadConfigFunc = restoreLoad
		driverFactoryFunc = restoreDriver
	})

	configPath := writeTempFile(t, "config.yaml", "current_account: work\naccounts:\n  - name: work\n    driver: fake\n    username: support@nono.im\n")
	loadConfigFunc = config.Load

	draftPath := writeTempFile(t, "draft.json", `{
  "account": "work",
  "to": [{"address": "user@example.com"}],
  "subject": "Welcome",
  "body_text": "Hello from MailCLI."
}`)
	operationsPath := filepath.Join(t.TempDir(), "operations.jsonl")

	driverFactoryFunc = func(account config.AccountConfig) (driver.Driver, error) {
		t.Fatalf("prepare must not initialize or call the transport driver")
		return nil, nil
	}
	prepareCmd := NewRootCmd()
	var prepareOut bytes.Buffer
	prepareCmd.SetOut(&prepareOut)
	prepareCmd.SetErr(&prepareOut)
	prepareCmd.SetArgs([]string{"send", "prepare", "--config", configPath, "--operations", operationsPath, draftPath})
	if err := prepareCmd.Execute(); err != nil {
		t.Fatalf("expected send prepare to succeed: %v\n%s", err, prepareOut.String())
	}
	var prepareResult map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(prepareOut.Bytes()), &prepareResult); err != nil {
		t.Fatalf("expected JSON prepare result: %v\n%s", err, prepareOut.String())
	}
	intentID := prepareResult["intent_id"].(string)

	driverFactoryFunc = func(account config.AccountConfig) (driver.Driver, error) {
		return &fakeSendDriver{}, nil
	}
	firstConfirm := NewRootCmd()
	var firstOut bytes.Buffer
	firstConfirm.SetOut(&firstOut)
	firstConfirm.SetErr(&firstOut)
	firstConfirm.SetArgs([]string{"send", "confirm", "--config", configPath, "--operations", operationsPath, intentID})
	if err := firstConfirm.Execute(); err != nil {
		t.Fatalf("expected first confirm to succeed: %v\n%s", err, firstOut.String())
	}

	driverFactoryFunc = func(account config.AccountConfig) (driver.Driver, error) {
		t.Fatalf("second confirm must not call the transport driver")
		return nil, nil
	}
	secondConfirm := NewRootCmd()
	var secondOut bytes.Buffer
	var secondErr bytes.Buffer
	secondConfirm.SetOut(&secondOut)
	secondConfirm.SetErr(&secondErr)
	secondConfirm.SetArgs([]string{"send", "confirm", "--config", configPath, "--operations", operationsPath, intentID})
	err := secondConfirm.Execute()
	if !errors.Is(err, errSendFailure) {
		t.Fatalf("expected second confirm to fail with outbound sentinel, got %v\n%s", err, secondOut.String())
	}
	var secondResult map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(secondOut.Bytes()), &secondResult); err != nil {
		t.Fatalf("expected JSON duplicate-confirm failure: %v\n%s", err, secondOut.String())
	}
	if !strings.Contains(secondOut.String(), `"ok": false`) || !strings.Contains(secondOut.String(), `"code": "intent_already_sent"`) {
		t.Fatalf("expected structured already-sent failure, got %s", secondOut.String())
	}
	if secondResult["intent_id"] != intentID || strings.TrimSpace(secondResult["operation_id"].(string)) == "" {
		t.Fatalf("expected duplicate-confirm failure to include intent and operation ids, got %#v", secondResult)
	}

	rawLog, readErr := os.ReadFile(operationsPath)
	if readErr != nil {
		t.Fatalf("expected operations log to be written: %v", readErr)
	}
	logText := string(rawLog)
	if strings.Count(logText, `"status":"sent"`) != 1 {
		t.Fatalf("expected exactly one sent operation log entry, got %s", logText)
	}
	if !strings.Contains(logText, `"status":"failed"`) || !strings.Contains(logText, `"code":"intent_already_sent"`) {
		t.Fatalf("expected failed duplicate-confirm operation log entry, got %s", logText)
	}
}

func TestSendConfirmLogsFailedOperation(t *testing.T) {
	restoreLoad := loadConfigFunc
	restoreDriver := driverFactoryFunc
	t.Cleanup(func() {
		loadConfigFunc = restoreLoad
		driverFactoryFunc = restoreDriver
	})

	configPath := writeTempFile(t, "config.yaml", "current_account: work\naccounts:\n  - name: work\n    driver: fake\n    username: support@nono.im\n")
	loadConfigFunc = config.Load
	draftPath := writeTempFile(t, "draft.json", `{
  "account": "work",
  "to": [{"address": "user@example.com"}],
  "subject": "Welcome",
  "body_text": "Hello from MailCLI."
}`)
	operationsPath := filepath.Join(t.TempDir(), "operations.jsonl")

	driverFactoryFunc = func(account config.AccountConfig) (driver.Driver, error) {
		t.Fatalf("prepare must not initialize or call the transport driver")
		return nil, nil
	}
	prepareCmd := NewRootCmd()
	var prepareOut bytes.Buffer
	prepareCmd.SetOut(&prepareOut)
	prepareCmd.SetErr(&prepareOut)
	prepareCmd.SetArgs([]string{"send", "prepare", "--config", configPath, "--operations", operationsPath, draftPath})
	if err := prepareCmd.Execute(); err != nil {
		t.Fatalf("expected send prepare to succeed: %v\n%s", err, prepareOut.String())
	}
	var prepareResult map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(prepareOut.Bytes()), &prepareResult); err != nil {
		t.Fatalf("expected JSON prepare result: %v\n%s", err, prepareOut.String())
	}
	intentID := prepareResult["intent_id"].(string)

	driverFactoryFunc = func(account config.AccountConfig) (driver.Driver, error) {
		return nil, driver.ErrDriverConfigInvalid
	}
	confirmCmd := NewRootCmd()
	var confirmOut bytes.Buffer
	var confirmErr bytes.Buffer
	confirmCmd.SetOut(&confirmOut)
	confirmCmd.SetErr(&confirmErr)
	confirmCmd.SetArgs([]string{"send", "confirm", "--config", configPath, "--operations", operationsPath, intentID})
	err := confirmCmd.Execute()
	if !errors.Is(err, errSendFailure) {
		t.Fatalf("expected outbound failure sentinel, got %v\n%s", err, confirmOut.String())
	}
	var confirmResult map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(confirmOut.Bytes()), &confirmResult); err != nil {
		t.Fatalf("expected JSON confirm failure: %v\n%s", err, confirmOut.String())
	}
	if !strings.Contains(confirmOut.String(), `"ok": false`) || !strings.Contains(confirmOut.String(), `"code": "transport_failed"`) {
		t.Fatalf("expected structured confirm failure, got %s", confirmOut.String())
	}
	if confirmResult["intent_id"] != intentID || strings.TrimSpace(confirmResult["operation_id"].(string)) == "" {
		t.Fatalf("expected confirm failure to include intent and operation ids, got %#v", confirmResult)
	}

	rawLog, readErr := os.ReadFile(operationsPath)
	if readErr != nil {
		t.Fatalf("expected operations log to be written: %v", readErr)
	}
	if !strings.Contains(string(rawLog), `"status":"failed"`) || !strings.Contains(string(rawLog), `"code":"transport_failed"`) {
		t.Fatalf("expected failed operation log entry, got %s", string(rawLog))
	}
	if strings.Contains(string(rawLog), "Hello from MailCLI.") {
		t.Fatalf("operation log must not store full draft body: %s", string(rawLog))
	}
}

func TestOperationsListAndShowReadOperationLog(t *testing.T) {
	restoreLoad := loadConfigFunc
	restoreDriver := driverFactoryFunc
	t.Cleanup(func() {
		loadConfigFunc = restoreLoad
		driverFactoryFunc = restoreDriver
	})

	configPath := writeTempFile(t, "config.yaml", "current_account: work\naccounts:\n  - name: work\n    driver: fake\n    username: support@nono.im\n")
	loadConfigFunc = config.Load
	operationsPath := filepath.Join(t.TempDir(), "operations.jsonl")
	draftPath := writeTempFile(t, "draft.json", `{
  "account": "work",
  "to": [{"address": "user@example.com"}],
  "subject": "Welcome",
  "body_text": "Hello from MailCLI."
}`)

	driverFactoryFunc = func(account config.AccountConfig) (driver.Driver, error) {
		return nil, nil
	}
	prepareCmd := NewRootCmd()
	var prepareOut bytes.Buffer
	prepareCmd.SetOut(&prepareOut)
	prepareCmd.SetErr(&prepareOut)
	prepareCmd.SetArgs([]string{"send", "prepare", "--config", configPath, "--operations", operationsPath, draftPath})
	if err := prepareCmd.Execute(); err != nil {
		t.Fatalf("expected send prepare to succeed: %v\n%s", err, prepareOut.String())
	}
	var prepareResult map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(prepareOut.Bytes()), &prepareResult); err != nil {
		t.Fatalf("expected JSON prepare result: %v\n%s", err, prepareOut.String())
	}

	fake := &fakeSendDriver{}
	driverFactoryFunc = func(account config.AccountConfig) (driver.Driver, error) {
		return fake, nil
	}
	confirmCmd := NewRootCmd()
	var confirmOut bytes.Buffer
	confirmCmd.SetOut(&confirmOut)
	confirmCmd.SetErr(&confirmOut)
	confirmCmd.SetArgs([]string{"send", "confirm", "--config", configPath, "--operations", operationsPath, prepareResult["intent_id"].(string)})
	if err := confirmCmd.Execute(); err != nil {
		t.Fatalf("expected send confirm to succeed: %v\n%s", err, confirmOut.String())
	}
	var confirmResult map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(confirmOut.Bytes()), &confirmResult); err != nil {
		t.Fatalf("expected JSON confirm result: %v\n%s", err, confirmOut.String())
	}
	operationID := confirmResult["operation_id"].(string)

	listCmd := NewRootCmd()
	var listOut bytes.Buffer
	listCmd.SetOut(&listOut)
	listCmd.SetErr(&listOut)
	listCmd.SetArgs([]string{"operations", "list", "--operations", operationsPath})
	if err := listCmd.Execute(); err != nil {
		t.Fatalf("expected operations list to succeed: %v\n%s", err, listOut.String())
	}
	if !strings.Contains(listOut.String(), `"operations"`) || !strings.Contains(listOut.String(), `"status": "sent"`) {
		t.Fatalf("expected operations list to include sent entry, got %s", listOut.String())
	}
	if strings.Contains(listOut.String(), "Hello from MailCLI.") {
		t.Fatalf("operations list must not print full draft body: %s", listOut.String())
	}

	showCmd := NewRootCmd()
	var showOut bytes.Buffer
	showCmd.SetOut(&showOut)
	showCmd.SetErr(&showOut)
	showCmd.SetArgs([]string{"operations", "show", "--operations", operationsPath, operationID})
	if err := showCmd.Execute(); err != nil {
		t.Fatalf("expected operations show to succeed: %v\n%s", err, showOut.String())
	}
	if !strings.Contains(showOut.String(), `"id": "`+operationID+`"`) || !strings.Contains(showOut.String(), `"status": "sent"`) {
		t.Fatalf("expected operations show to return requested entry, got %s", showOut.String())
	}
}

func TestSendCommandDefaultsFromAddressFromAccountConfig(t *testing.T) {
	restoreLoad := loadConfigFunc
	restoreDriver := driverFactoryFunc
	t.Cleanup(func() {
		loadConfigFunc = restoreLoad
		driverFactoryFunc = restoreDriver
	})

	configPath := writeTempFile(t, "config.yaml", "current_account: work\naccounts:\n  - name: work\n    driver: fake\n    username: support@nono.im\n")
	loadConfigFunc = config.Load

	fake := &fakeSendDriver{}
	driverFactoryFunc = func(account config.AccountConfig) (driver.Driver, error) {
		return fake, nil
	}

	draftPath := writeTempFile(t, "draft.json", `{
  "account": "work",
  "to": [{"address": "user@example.com"}],
  "subject": "Welcome",
  "body_text": "Hello from MailCLI."
}`)

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"send", "--config", configPath, draftPath})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected send command with derived from address to succeed: %v", err)
	}

	if !strings.Contains(string(fake.lastRaw), "From: support@nono.im") {
		t.Fatalf("expected configured account username to fill missing From header, got %s", string(fake.lastRaw))
	}
}

func TestSendCommandReturnsStructuredAuthFailure(t *testing.T) {
	restoreLoad := loadConfigFunc
	restoreDriver := driverFactoryFunc
	t.Cleanup(func() {
		loadConfigFunc = restoreLoad
		driverFactoryFunc = restoreDriver
	})

	configPath := writeTempFile(t, "config.yaml", "current_account: work\naccounts:\n  - name: work\n    driver: fake\n")
	loadConfigFunc = config.Load

	fake := &fakeSendDriver{sendErr: assertErr("535 Authentication credentials invalid")}
	driverFactoryFunc = func(account config.AccountConfig) (driver.Driver, error) {
		return fake, nil
	}

	draftPath := writeTempFile(t, "draft.json", `{
  "account": "work",
  "from": {"address": "support@nono.im"},
  "to": [{"address": "user@example.com"}],
  "subject": "Welcome",
  "body_text": "Hello from MailCLI."
}`)

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"send", "--config", configPath, draftPath})

	err := cmd.Execute()
	if !errors.Is(err, errSendFailure) {
		t.Fatalf("expected outbound failure sentinel, got %v", err)
	}

	if !strings.Contains(out.String(), `"ok": false`) || !strings.Contains(out.String(), `"code": "auth_failed"`) {
		t.Fatalf("expected auth_failed send result, got %s", out.String())
	}
}

func TestSendCommandReturnsStructuredTypedAuthFailure(t *testing.T) {
	restoreLoad := loadConfigFunc
	restoreDriver := driverFactoryFunc
	t.Cleanup(func() {
		loadConfigFunc = restoreLoad
		driverFactoryFunc = restoreDriver
	})

	configPath := writeTempFile(t, "config.yaml", "current_account: work\naccounts:\n  - name: work\n    driver: fake\n")
	loadConfigFunc = config.Load

	fake := &fakeSendDriver{sendErr: driver.ErrAuthFailed}
	driverFactoryFunc = func(account config.AccountConfig) (driver.Driver, error) {
		return fake, nil
	}

	draftPath := writeTempFile(t, "draft.json", `{
  "account": "work",
  "from": {"address": "support@nono.im"},
  "to": [{"address": "user@example.com"}],
  "subject": "Welcome",
  "body_text": "Hello from MailCLI."
}`)

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"send", "--config", configPath, draftPath})

	err := cmd.Execute()
	if !errors.Is(err, errSendFailure) {
		t.Fatalf("expected outbound failure sentinel, got %v", err)
	}

	if !strings.Contains(out.String(), `"ok": false`) || !strings.Contains(out.String(), `"code": "auth_failed"`) {
		t.Fatalf("expected typed auth_failed send result, got %s", out.String())
	}
}

func TestSendCommandReturnsStructuredAccountNotFound(t *testing.T) {
	restoreLoad := loadConfigFunc
	restoreDriver := driverFactoryFunc
	t.Cleanup(func() {
		loadConfigFunc = restoreLoad
		driverFactoryFunc = restoreDriver
	})

	configPath := writeTempFile(t, "config.yaml", "current_account: work\naccounts:\n  - name: work\n    driver: fake\n")
	loadConfigFunc = config.Load
	driverFactoryFunc = func(account config.AccountConfig) (driver.Driver, error) {
		t.Fatalf("driver factory should not be called when account resolution fails")
		return nil, nil
	}

	draftPath := writeTempFile(t, "draft.json", `{
  "account": "missing",
  "from": {"address": "support@nono.im"},
  "to": [{"address": "user@example.com"}],
  "subject": "Welcome",
  "body_text": "Hello from MailCLI."
}`)

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"send", "--config", configPath, draftPath})

	err := cmd.Execute()
	if !errors.Is(err, errSendFailure) {
		t.Fatalf("expected outbound failure sentinel, got %v", err)
	}

	if !strings.Contains(out.String(), `"ok": false`) || !strings.Contains(out.String(), `"code": "account_not_found"`) {
		t.Fatalf("expected account_not_found send result, got %s", out.String())
	}
}

func TestReplyCommandUsesConfiguredDriver(t *testing.T) {
	restoreLoad := loadConfigFunc
	restoreDriver := driverFactoryFunc
	t.Cleanup(func() {
		loadConfigFunc = restoreLoad
		driverFactoryFunc = restoreDriver
	})

	configPath := writeTempFile(t, "config.yaml", "current_account: work\naccounts:\n  - name: work\n    driver: fake\n")
	loadConfigFunc = config.Load

	fake := &fakeSendDriver{}
	driverFactoryFunc = func(account config.AccountConfig) (driver.Driver, error) {
		return fake, nil
	}

	replyPath := writeTempFile(t, "reply.json", `{
  "account": "work",
  "from": {"address": "support@nono.im"},
  "to": [{"address": "user@example.com"}],
  "subject": "Question",
  "body_text": "Thanks for the email.",
  "reply_to_message_id": "<orig-123@example.com>",
  "references": ["<orig-123@example.com>"]
}`)

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"reply", "--config", configPath, replyPath})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected reply command to succeed: %v", err)
	}

	if len(fake.lastRaw) == 0 || !strings.Contains(string(fake.lastRaw), "In-Reply-To: <orig-123@example.com>") {
		t.Fatalf("expected reply command to send MIME with thread headers")
	}
	if !strings.Contains(out.String(), "\"ok\": true") {
		t.Fatalf("expected send result output")
	}
}

func TestReplyCommandReturnsStructuredMessageNotFound(t *testing.T) {
	restoreLoad := loadConfigFunc
	restoreDriver := driverFactoryFunc
	t.Cleanup(func() {
		loadConfigFunc = restoreLoad
		driverFactoryFunc = restoreDriver
	})

	configPath := writeTempFile(t, "config.yaml", "current_account: work\naccounts:\n  - name: work\n    driver: fake\n")
	loadConfigFunc = config.Load

	fake := &fakeSendDriver{fetchErr: assertErr("message not found: imap:uid:123")}
	driverFactoryFunc = func(account config.AccountConfig) (driver.Driver, error) {
		return fake, nil
	}

	replyPath := writeTempFile(t, "reply.json", `{
  "account": "work",
  "from": {"address": "support@nono.im"},
  "to": [{"address": "user@example.com"}],
  "body_text": "Thanks for the email.",
  "reply_to_id": "imap:uid:123"
}`)

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"reply", "--config", configPath, replyPath})

	err := cmd.Execute()
	if !errors.Is(err, errSendFailure) {
		t.Fatalf("expected outbound failure sentinel, got %v", err)
	}

	if !strings.Contains(out.String(), `"ok": false`) || !strings.Contains(out.String(), `"code": "message_not_found"`) {
		t.Fatalf("expected message_not_found reply result, got %s", out.String())
	}
}

func TestSendCommandReturnsStructuredTransportNotConfigured(t *testing.T) {
	restoreLoad := loadConfigFunc
	restoreDriver := driverFactoryFunc
	t.Cleanup(func() {
		loadConfigFunc = restoreLoad
		driverFactoryFunc = restoreDriver
	})

	configPath := writeTempFile(t, "config.yaml", "current_account: work\naccounts:\n  - name: work\n    driver: fake\n")
	loadConfigFunc = config.Load

	fake := &fakeSendDriver{sendErr: assertErr("smtp settings not configured for account")}
	driverFactoryFunc = func(account config.AccountConfig) (driver.Driver, error) {
		return fake, nil
	}

	draftPath := writeTempFile(t, "draft.json", `{
  "account": "work",
  "from": {"address": "support@nono.im"},
  "to": [{"address": "user@example.com"}],
  "subject": "Welcome",
  "body_text": "Hello from MailCLI."
}`)

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"send", "--config", configPath, draftPath})

	err := cmd.Execute()
	if !errors.Is(err, errSendFailure) {
		t.Fatalf("expected outbound failure sentinel, got %v", err)
	}

	if !strings.Contains(out.String(), `"ok": false`) || !strings.Contains(out.String(), `"code": "transport_not_configured"`) {
		t.Fatalf("expected transport_not_configured send result, got %s", out.String())
	}
}

func TestSendCommandReturnsStructuredTypedInvalidDriverConfig(t *testing.T) {
	restoreLoad := loadConfigFunc
	restoreDriver := driverFactoryFunc
	t.Cleanup(func() {
		loadConfigFunc = restoreLoad
		driverFactoryFunc = restoreDriver
	})

	configPath := writeTempFile(t, "config.yaml", "current_account: work\naccounts:\n  - name: work\n    driver: fake\n")
	loadConfigFunc = config.Load
	driverFactoryFunc = func(account config.AccountConfig) (driver.Driver, error) {
		return nil, driver.ErrDriverConfigInvalid
	}

	draftPath := writeTempFile(t, "draft.json", `{
  "account": "work",
  "from": {"address": "support@nono.im"},
  "to": [{"address": "user@example.com"}],
  "subject": "Welcome",
  "body_text": "Hello from MailCLI."
}`)

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"send", "--config", configPath, draftPath})

	err := cmd.Execute()
	if !errors.Is(err, errSendFailure) {
		t.Fatalf("expected outbound failure sentinel, got %v", err)
	}

	if !strings.Contains(out.String(), `"ok": false`) || !strings.Contains(out.String(), `"code": "transport_failed"`) {
		t.Fatalf("expected transport_failed result for typed invalid driver config, got %s", out.String())
	}
}

func TestReplyCommandResolvesReplyToIDViaFetch(t *testing.T) {
	restoreLoad := loadConfigFunc
	restoreDriver := driverFactoryFunc
	t.Cleanup(func() {
		loadConfigFunc = restoreLoad
		driverFactoryFunc = restoreDriver
	})

	configPath := writeTempFile(t, "config.yaml", "current_account: work\naccounts:\n  - name: work\n    driver: fake\n")
	loadConfigFunc = config.Load

	fake := &fakeSendDriver{}
	driverFactoryFunc = func(account config.AccountConfig) (driver.Driver, error) {
		return &struct {
			*fakeSendDriver
		}{fake}, nil
	}

	replyPath := writeTempFile(t, "reply.json", `{
  "account": "work",
  "from": {"address": "support@nono.im"},
  "to": [{"address": "user@example.com"}],
  "body_text": "Thanks for the email.",
  "reply_to_id": "imap:uid:123"
}`)

	drv := &replyResolveDriver{
		fakeSendDriver: fake,
		raw:            []byte("From: sender@example.com\r\nTo: support@nono.im\r\nSubject: Original subject\r\nMessage-ID: <orig-123@example.com>\r\nReferences: <older-1@example.com>\r\nDate: Wed, 26 Mar 2026 11:00:00 +0800\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nOriginal body"),
	}
	driverFactoryFunc = func(account config.AccountConfig) (driver.Driver, error) {
		return drv, nil
	}

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"reply", "--config", configPath, replyPath})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected reply command to succeed: %v", err)
	}

	raw := string(drv.lastRaw)
	if !strings.Contains(raw, "In-Reply-To: <orig-123@example.com>") {
		t.Fatalf("expected reply_to_id to resolve original message id")
	}
	if !strings.Contains(raw, "References: <older-1@example.com> <orig-123@example.com>") {
		t.Fatalf("expected references to include original thread")
	}
	if !strings.Contains(raw, "Subject: Re: Original subject") {
		t.Fatalf("expected subject to derive from original message")
	}
}

func TestReplyCommandDerivesRecipientAndSenderFromContext(t *testing.T) {
	restoreLoad := loadConfigFunc
	restoreDriver := driverFactoryFunc
	t.Cleanup(func() {
		loadConfigFunc = restoreLoad
		driverFactoryFunc = restoreDriver
	})

	configPath := writeTempFile(t, "config.yaml", "current_account: work\naccounts:\n  - name: work\n    driver: fake\n    username: support@nono.im\n")
	loadConfigFunc = config.Load

	drv := &replyResolveDriver{
		fakeSendDriver: &fakeSendDriver{},
		raw:            []byte("From: sender@example.com\r\nTo: support@nono.im\r\nSubject: Original subject\r\nMessage-ID: <orig-123@example.com>\r\nReferences: <older-1@example.com>\r\nDate: Wed, 26 Mar 2026 11:00:00 +0800\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nOriginal body"),
	}
	driverFactoryFunc = func(account config.AccountConfig) (driver.Driver, error) {
		return drv, nil
	}

	replyPath := writeTempFile(t, "reply.json", `{
  "account": "work",
  "body_text": "Thanks for the email.",
  "reply_to_id": "imap:uid:123"
}`)

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"reply", "--config", configPath, replyPath})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected reply command with derived recipients to succeed: %v", err)
	}

	raw := string(drv.lastRaw)
	if !strings.Contains(raw, "From: support@nono.im") {
		t.Fatalf("expected configured account username to fill reply From header, got %s", raw)
	}
	if !strings.Contains(raw, "To: sender@example.com") {
		t.Fatalf("expected original sender to fill reply recipient, got %s", raw)
	}
}

func TestReplyCommandDerivesRecipientFromOriginalToWhenOriginalFromIsSelf(t *testing.T) {
	restoreLoad := loadConfigFunc
	restoreDriver := driverFactoryFunc
	t.Cleanup(func() {
		loadConfigFunc = restoreLoad
		driverFactoryFunc = restoreDriver
	})

	configPath := writeTempFile(t, "config.yaml", "current_account: work\naccounts:\n  - name: work\n    driver: fake\n    username: support@nono.im\n")
	loadConfigFunc = config.Load

	drv := &replyResolveDriver{
		fakeSendDriver: &fakeSendDriver{},
		raw:            []byte("From: support@nono.im\r\nTo: sender@example.com, support@nono.im\r\nSubject: Original subject\r\nMessage-ID: <orig-123@example.com>\r\nReferences: <older-1@example.com>\r\nDate: Wed, 26 Mar 2026 11:00:00 +0800\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nOriginal body"),
	}
	driverFactoryFunc = func(account config.AccountConfig) (driver.Driver, error) {
		return drv, nil
	}

	replyPath := writeTempFile(t, "reply.json", `{
  "account": "work",
  "body_text": "Following up on the sent thread.",
  "reply_to_id": "imap:uid:123"
}`)

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"reply", "--config", configPath, replyPath})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected self-sent reply command to succeed: %v", err)
	}

	raw := string(drv.lastRaw)
	if !strings.Contains(raw, "To: sender@example.com") {
		t.Fatalf("expected original non-self recipient to be derived, got %s", raw)
	}
	if strings.Contains(raw, "To: support@nono.im,") || strings.Contains(raw, "To: sender@example.com, support@nono.im") {
		t.Fatalf("expected self address to be excluded from derived recipients, got %s", raw)
	}
}

func TestReplyCommandDoesNotFallbackToSelfWhenNoOtherRecipientExists(t *testing.T) {
	restoreLoad := loadConfigFunc
	restoreDriver := driverFactoryFunc
	t.Cleanup(func() {
		loadConfigFunc = restoreLoad
		driverFactoryFunc = restoreDriver
	})

	configPath := writeTempFile(t, "config.yaml", "current_account: work\naccounts:\n  - name: work\n    driver: fake\n    username: support@nono.im\n")
	loadConfigFunc = config.Load

	drv := &replyResolveDriver{
		fakeSendDriver: &fakeSendDriver{},
		raw:            []byte("From: support@nono.im\r\nTo: support@nono.im\r\nSubject: Original subject\r\nMessage-ID: <orig-123@example.com>\r\nDate: Wed, 26 Mar 2026 11:00:00 +0800\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nOriginal body"),
	}
	driverFactoryFunc = func(account config.AccountConfig) (driver.Driver, error) {
		return drv, nil
	}

	replyPath := writeTempFile(t, "reply.json", `{
  "account": "work",
  "body_text": "Following up on the sent thread.",
  "reply_to_id": "imap:uid:123"
}`)

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"reply", "--config", configPath, replyPath})

	err := cmd.Execute()
	if !errors.Is(err, errSendFailure) {
		t.Fatalf("expected invalid reply draft to return outbound failure sentinel, got %v", err)
	}

	if strings.Contains(string(drv.lastRaw), "To: support@nono.im") {
		t.Fatalf("expected reply flow not to fallback to self recipient, got %s", string(drv.lastRaw))
	}
	if !strings.Contains(out.String(), `"code": "invalid_draft"`) {
		t.Fatalf("expected invalid_draft result when no reply recipient can be derived, got %s", out.String())
	}
}

func TestSendCommandDryRunInvalidJSONStillReturnsRawError(t *testing.T) {
	draftPath := writeTempFile(t, "draft.json", `{`)

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"send", "--dry-run", draftPath})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected dry-run invalid json to fail")
	}
	if errors.Is(err, errSendFailure) {
		t.Fatalf("expected dry-run invalid json to keep raw error path")
	}
	if strings.Contains(out.String(), `"ok": false`) {
		t.Fatalf("expected no structured SendResult on dry-run parse failure")
	}
}

type replyResolveDriver struct {
	*fakeSendDriver
	raw []byte
}

func (d *replyResolveDriver) List(ctx context.Context, query schema.SearchQuery) ([]schema.MessageMetaSummary, error) {
	return nil, nil
}

func (d *replyResolveDriver) FetchRaw(ctx context.Context, id string) ([]byte, error) {
	return d.raw, nil
}

func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func assertErr(message string) error {
	return fakeErr(message)
}

type fakeErr string

func (e fakeErr) Error() string {
	return string(e)
}
