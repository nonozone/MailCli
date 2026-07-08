[中文](../../zh-CN/project/agent-harness-strategy.md) | English

# Agent Harness Strategy

This document defines how MailCLI should absorb agent harness patterns instead
of staying only "a command-line tool for AI agents."

Short version:

- Skills answer "how should an agent perform this kind of task?"
- A harness answers "how does a long-running task advance, pause, verify,
  hand off, audit, and involve a human?"
- MailCLI's product opportunity is not exposing mail protocols to AI. It is
  turning email handling into executable workflows with evidence and boundaries.

## External References

As of 2026-07-08, external agent and workflow systems point to a few consistent
patterns:

| System | Lesson | Meaning for MailCLI |
| --- | --- | --- |
| [Specability](https://specability.com/approach/) | Find invariants, write specs, ship the harness that runs rules, surfaces judgment calls, and records decisions | MailCLI should encode which mail actions require confirmation, which fields must not leak, and which outputs are stable |
| [Agent Skills](https://agentskills.io/) / [Claude Skills](https://docs.anthropic.com/en/docs/claude-code/skills) | Skills are on-demand packages of knowledge, workflows, scripts, and references | MailCLI can ship skills, but must not rely on prompt discipline for safety or stability |
| [LangGraph](https://docs.langchain.com/oss/python/langgraph/overview) | Long tasks need durable execution, persistence, human-in-the-loop, and stateful workflows | Email handling often spans messages, confirmations, and follow-up actions |
| [Deep Agents](https://docs.langchain.com/oss/python/deepagents/overview) | Agent harnesses include planning, file systems, subagents, long-term memory, and human approval | MailCLI contracts should support plans, candidate actions, approval, and audit results |
| [Temporal](https://docs.temporal.io/temporal) | Long workflows need event history, resumable state, and failure recovery | Mail automation needs operation logs and intent ids |
| [Microsoft Agent Framework](https://learn.microsoft.com/en-us/agent-framework/overview/) | Agents fit open-ended conversation; workflows fit explicit steps; HITL pauses and resumes through request / response | Dangerous MailCLI operations should become structured requests for human confirmation |
| [OpenAI Agents SDK](https://openai.github.io/openai-agents-python/) | Production agent runtimes need handoffs, guardrails, sessions, tracing, and human-in-the-loop | MailCLI does not need a model runtime, but it should expose boundaries those runtimes can call safely |
| [Google ADK](https://google.github.io/adk-docs/agents/workflow-agents/) | Complex systems need workflow agents, sessions, memory, artifacts, and confirmations | Email results should become artifacts and operations, not only chat transcript text |

## Harness Versus Skill

| Dimension | Skill | Harness |
| --- | --- | --- |
| Shape | Loadable instruction package | Runtime control layer |
| Job | Tell an agent how to do something | Decide the next action, whether it is allowed, and how completion is proven |
| State | Usually depends on conversation context | Has task, run, artifact, check, and log state |
| Failure handling | The agent explains and retries | Recovery state, failure reason, and next action are explicit |
| Human involvement | Asked when the agent remembers to ask | Pauses at explicit judgment points and resumes with a response |
| Verification | Mostly checklist or test suggestion | Gates, criteria, evidence, CI, or verifiers |
| Audit | Usually no durable trail | Records decisions, operations, evidence, and boundaries |

Skills are still valuable for provider setup notes, parser fixture workflows,
release checklists, and integration guides. They should not carry the whole
burden of multi-day development, destructive mail operations, or JSON contract
stability.

## Human Interaction Model

Harness-style interaction means fewer questions, asked at better points.

Start by clarifying the goal, not implementation details:

- Is the goal to connect a real mailbox, or to automate email handling?
- Which user step should disappear after this work?
- Which operations must stay human-confirmed?
- Which capabilities are explicitly out of scope?

For larger tasks, capture a goal card:

```text
Objective: what this achieves
User pain: which burden this reduces
Non-goals: what this will not do
Acceptance: how completion will be judged
Human decisions: where a person must decide
Evidence: which checks prove completion
```

During execution, ask only at judgment points:

- A direction changes product positioning.
- A default behavior carries safety risk.
- A public JSON field creates compatibility cost.

Avoid asking about routine file placement, test names, or implementation details
that the codebase already answers.

## Reducing User Burden

MailCLI users do not want to learn mail protocols, OAuth edge cases, MIME, IMAP
flags, or tool schemas. Their real burdens are:

- connecting their mailbox
- trusting that AI will not send, delete, or move mail accidentally
- knowing which messages matter
- knowing what automation executed
- recovering from failures

Product strategy:

1. Push setup complexity into the Go CLI.
2. Give humans short next steps and agents stable JSON.
3. Turn dangerous actions into auditable intents.
4. Make failures explainable with stable error codes and next-step guidance.

## Development Model

Development should follow workflows, not feature lists.

Current mainline:

1. Connect existing mailboxes.
2. Understand inboxes and threads.
3. Extract high-value information such as attachments, invoices, codes, and actions.
4. Confirm before execution and record after execution.

Each slice should declare:

- goal and non-goals
- affected CLI / schema / docs
- whether public JSON contracts change
- whether secrets, destructive actions, or stored user content are involved
- verification commands
- completion evidence

## Determining User Goals

Order of clarification:

1. Identify the friction to reduce.
2. Define a verifiable success state.
3. Define boundaries and non-goals.
4. Choose implementation details last.

For significant requests, the agent should restate:

```text
I think the goal is:
- ...

I will treat these as non-goals:
- ...

I will verify with:
- ...

I will ask you only if:
- ...
```

This reduces the user's burden of managing the agent.

## MailCLI Product Additions

Near-term additions:

1. Improve `mailcli account add` readiness output and provider-specific next steps.
2. Add `mailcli inbox triage` or an equivalent command with priority, needs-reply, todo candidates, and action candidates.
3. Add operation intents for `send`, `reply`, `delete`, `move`, and `mark`.
4. Add local operation logs.
5. Keep MCP write tools disabled by default unless explicitly enabled.

Defer:

- hosted Agent mailbox identities
- heavy OAuth provider platforms
- runtime plugin loading
- model providers inside Go core
- official paths that require users to understand Python or Node runtimes

## Current Small Slice

Current first implementation slice:

**Operation intent and logs for `send`.**

This directly addresses the user's strongest trust concern: AI should not send,
delete, or move mail without a confirmable intent and durable audit trail.

Implemented first-phase contract:

- `mailcli send prepare [--config] [--operations] draft.json`
- `mailcli send confirm [--config] [--operations] <intent-id>`
- `mailcli operations list [--operations]`
- `mailcli operations show [--operations] <operation-id|intent-id>`

Acceptance criteria:

- dry-run / prepare does not send mail
- confirm executes only the same intent
- operation logs do not store secrets
- JSON output is stable and covered by command-level tests
- docs say when agents must request human confirmation

Remaining extensions:

- add the same prepare / confirm / log contract for `reply`, `delete`, `move`, and `mark`
- add inbox triage signals before expanding automatic execution
- keep MCP write tools disabled by default unless a local policy explicitly enables them
