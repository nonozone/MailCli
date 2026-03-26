# Agent Inbox Example Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a complete, minimal agent example that shows how an agent can call `mailcli` to read mail, reason over structured output, and generate a reply dry-run.

**Architecture:** Build one Python example script as the primary integration surface. It should support two input modes, local `.eml` via `mailcli parse` and configured inbox messages via `mailcli get`, then produce a JSON agent report with analysis, extracted codes/actions, and an optional reply dry-run block.

**Tech Stack:** Python 3 standard library, existing `mailcli` CLI, Go integration tests

---

**Follow-up enhancement:** The example now also supports an external provider mode, so developers can plug in their own LLM or agent runtime through a simple stdin/stdout JSON contract without adding SDK dependencies to the repository.

## Chunk 1: Failing Integration Tests

### Task 1: Define the example contract through tests

**Files:**
- Create: `examples/examples_test.go`

- [x] **Step 1: Write a failing integration test for verification-email analysis**
- [x] **Step 2: Write a failing integration test for reply dry-run generation**
- [x] **Step 3: Build a temporary `mailcli` binary during tests**
- [x] **Step 4: Run focused tests and confirm they fail**

## Chunk 2: Example Implementation

### Task 2: Implement the Python agent example

**Files:**
- Create: `examples/python/agent_inbox_assistant.py`

- [x] **Step 1: Add input selection for `--email` and `--message-id`**
- [x] **Step 2: Call `mailcli parse` or `mailcli get` and parse JSON output**
- [x] **Step 3: Generate a small agent analysis block**
- [x] **Step 4: Add optional reply-draft generation plus `mailcli reply --dry-run -`**
- [x] **Step 5: Run focused tests and confirm they pass**

## Chunk 3: Docs

### Task 3: Document the agent example

**Files:**
- Modify: `README.md`
- Modify: `README.zh-CN.md`
- Modify: `docs/en/agent-workflows.md`
- Modify: `docs/zh-CN/agent-workflows.md`
- Create: `docs/en/examples/agent-inbox-assistant.md`
- Create: `docs/zh-CN/examples/agent-inbox-assistant.md`

- [x] **Step 1: Add the example to README**
- [x] **Step 2: Add dedicated English and Chinese example docs**
- [x] **Step 3: Link the docs from agent-workflow pages**

## Chunk 4: Verification

### Task 4: Verify and publish

**Files:**
- Modify: `docs/superpowers/plans/2026-03-26-agent-inbox-example.md`

- [x] **Step 1: Run full repository tests**
- [x] **Step 2: Mark the plan complete**
- [ ] **Step 3: Commit**
- [ ] **Step 4: Push**
