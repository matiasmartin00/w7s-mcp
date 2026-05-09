---
layout: default
title: Tool Reference
nav_order: 4
---

# Tool Reference
{: .no_toc }

<details open markdown="block">
  <summary>Table of contents</summary>
  {: .text-delta }
- TOC
{:toc}
</details>

---

## Overview

w7s-mcp exposes **5 public MCP tools**. The Lead Agent (LLM) is responsible for evaluating step output against the `expects` hint and calling either `complete_step` or `fail_step` accordingly. The server never auto-evaluates `expects`.

---

## `start_run`

Starts a new workflow execution.

**Input:**

| Param         | Type   | Required | Description                   |
|---------------|--------|----------|-------------------------------|
| `workflow_id` | string | Yes      | ID or path of the YAML file   |
| `task`        | string | Yes      | Task description               |

**Output:**

```json
{
  "run_id": "uuid",
  "workflow": "Feature Development",
  "steps": ["explore", "implement", "verify"],
  "message": "Run started. Call get_next_step with run_id: ..."
}
```

**Side effects:**
- Creates a record in `runs` with status `running`
- Creates records in `steps` for each step with status `pending`
- Persists `{{task}}` as a variable

---

## `get_next_step`

Returns the next step to execute with the interpolated prompt. Marks the step as `running`.

**Input:**

| Param    | Type   | Required | Description |
|----------|--------|----------|-------------|
| `run_id` | string | Yes      | Run ID      |

**Output (pending step available):**

```json
{
  "status": "next_step",
  "step_id": "explore",
  "agent": {
    "id": "explorer",
    "name": "Explorer",
    "files": { "AGENTS.md": "agents/explorer.md" }
  },
  "prompt": "Task: implement feature X\n\nExplore the codebase...",
  "attempt": 1,
  "expects": "STATUS:\\s*done",
  "instruction": "Execute the agent 'Explorer' with the prompt above. Evaluate its output against the 'expects' pattern. If it matches, call complete_step. If it does not match or the step failed, call fail_step."
}
```

**Output (workflow complete):**

```json
{ "status": "done", "message": "All steps completed. Workflow finished." }
```

**Output (run not active):**

```json
{ "status": "escalated", "message": "Run is not active" }
```

**Side effects:** Updates step to `running`, increments `attempt`. If no pending steps remain, marks run as `done`.

**Selection logic:** First step with status `pending` or `running` (insertion order).

---

## `complete_step`

Marks a step as `done`. Extracts variables from the output and persists them. Called by the Lead when the step output matches `expects`.

**Input:**

| Param     | Type   | Required | Description        |
|-----------|--------|----------|--------------------|
| `run_id`  | string | Yes      |                    |
| `step_id` | string | Yes      |                    |
| `output`  | string | Yes      | Full agent output  |

**Output:**

```json
{
  "status": "step_done",
  "step_id": "explore",
  "variables_extracted": {
    "scope": "authentication module",
    "files": "src/auth.go"
  },
  "message": "Step 'explore' completed. Call get_next_step to continue."
}
```

> `complete_step` does **not** evaluate `expects`. The Lead is responsible for that decision.

---

## `fail_step`

Marks a step as `failed`. Called by the Lead when the output does not satisfy `expects`. Handles retry or escalation according to `on_fail`.

**Input:**

| Param     | Type   | Required | Description                                           |
|-----------|--------|----------|-------------------------------------------------------|
| `run_id`  | string | Yes      |                                                       |
| `step_id` | string | Yes      |                                                       |
| `reason`  | string | No       | Failure description — stored in `feedback_var` if configured |

**Output (retry available):**

```json
{
  "status": "retry",
  "step_id": "verify",
  "retry_step": "implement",
  "attempts_used": 1,
  "attempts_remaining": 2,
  "message": "Step 'verify' failed. Retrying from 'implement'. Call get_next_step to continue."
}
```

**Output (retries exhausted):**

```json
{
  "status": "escalated",
  "step_id": "verify",
  "attempts": 3,
  "escalate_to": "human",
  "message": "Step 'verify' failed after 3 attempt(s). Escalating to human."
}
```

**Retry logic:**
1. If `attempt >= max_retries` or no `on_fail` configured → escalate, mark run as `escalated`
2. If retries remain: persist `reason` in `feedback_var`, reset `retry_step` (and intermediate steps) to `pending`

---

## `get_run_status`

Returns the full state of a run: status, steps, and accumulated variables.

**Input:** `run_id` or `workflow_id` (finds the most recent active run).

**Output:**

```json
{
  "run_id": "uuid",
  "workflow_id": "feature-dev",
  "task": "implement feature X",
  "status": "running",
  "steps": [
    { "step_id": "explore",   "status": "done",    "attempt": 1 },
    { "step_id": "implement", "status": "running", "attempt": 1 },
    { "step_id": "verify",    "status": "pending", "attempt": 0 }
  ],
  "variables": {
    "task":  "implement feature X",
    "scope": "authentication module",
    "files": "src/auth.go"
  }
}
```
