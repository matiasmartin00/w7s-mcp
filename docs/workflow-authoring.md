---
layout: default
title: Workflow Authoring
nav_order: 5
---

# Workflow Authoring
{: .no_toc }

<details open markdown="block">
  <summary>Table of contents</summary>
  {: .text-delta }
- TOC
{:toc}
</details>

---

## Schema

Workflows are defined in YAML and validated against `workflow.schema.json` (JSON Schema Draft 2020-12). Add this comment to your file for autocompletion in VS Code with the Red Hat YAML extension:

```yaml
# yaml-language-server: $schema=../../workflow.schema.json
```

---

## Structure

```yaml
id: string           # Unique workflow identifier
name: string         # Human-readable name
version: string      # Semantic version (e.g. "1.0.0")
description: string  # Description (optional)

agents:
  - id: string       # Agent identifier (referenced from steps)
    name: string     # Human-readable name
    workspace:
      files:         # Map of filename → path relative to the agent
        AGENTS.md: agents/explorer.md
        SKILL.md:  skills/explorer.md

steps:
  - id: string       # Step identifier
    agent: string    # Reference to agents[].id
    input: string    # Prompt template (supports {{variable}})
    expects: string  # Regex hint for the Lead to evaluate success (default: "STATUS:\s*done")
    extract:         # Variables to capture from output (optional)
      var_name: string  # variable_name: regex with capture group
    on_fail:
      retry_step: string       # Step to roll back to (default: current step)
      feedback_var: string     # Variable to store the failure reason
      max_retries: integer     # Maximum retries (default: 0)
      on_exhausted:
        escalate_to: "human"   # Who to escalate to when retries are exhausted
```

---

## System variables

These variables are automatically available in all step prompts:

| Variable    | Description                             |
|-------------|-----------------------------------------|
| `{{task}}`  | The original task passed to `start_run` |

---

## Variable extraction

Variables captured from step output via `extract` are available for interpolation in subsequent steps.

```yaml
steps:
  - id: explore
    agent: explorer
    input: "Explore the codebase for: {{task}}"
    expects: "STATUS:\\s*done"
    extract:
      scope: "SCOPE:\\s*(.+)"   # captures value after "SCOPE:"
      files: "FILES:\\s*(.+)"   # captures value after "FILES:"

  - id: implement
    agent: implementor
    input: |
      Task: {{task}}
      Scope: {{scope}}
      Files: {{files}}
```

- If the output contains a match, the capture group value is persisted.
- If there is no match, the variable is not created (not an error).
- Extraction only happens inside `complete_step`, not `fail_step`.

---

## Retry and escalation

```yaml
steps:
  - id: implement
    agent: implementor
    input: "..."
    on_fail:
      retry_step: explore        # Roll back to this step
      feedback_var: impl_feedback  # Store failure reason here
      max_retries: 2
      on_exhausted:
        escalate_to: human
```

**Retry logic:**
1. If `attempt >= max_retries` or no `on_fail` → escalate, mark run `escalated`
2. If retries remain: persist reason in `feedback_var`, reset `retry_step` (and all intermediate steps) to `pending`

The failure reason is available in subsequent steps via `{{impl_feedback}}`.

---

## Full example

```yaml
# yaml-language-server: $schema=../../workflow.schema.json
id: feature-dev
name: Feature Development
version: "1.0.0"
description: Three-step workflow — explore, implement, verify.

agents:
  - id: explorer
    name: Explorer
    workspace:
      files:
        AGENTS.md: agents/explorer.md
        SKILL.md:  skills/sdd-explore.md

  - id: implementor
    name: Implementor
    workspace:
      files:
        AGENTS.md: agents/implementor.md
        SKILL.md:  skills/sdd-implement.md

  - id: verifier
    name: Verifier
    workspace:
      files:
        AGENTS.md: agents/verifier.md
        SKILL.md:  skills/sdd-verify.md

steps:
  - id: explore
    agent: explorer
    input: |
      Task: {{task}}

      Explore the codebase and identify the scope of the change.
      At the end, include:
        STATUS: done
        SCOPE: <scope>
        FILES: <affected files>
    expects: "STATUS:\\s*done"
    extract:
      scope: "SCOPE:\\s*(.+)"
      files: "FILES:\\s*(.+)"

  - id: implement
    agent: implementor
    input: |
      Task: {{task}}
      Scope: {{scope}}
      Files: {{files}}

      Implement the changes.
      At the end, include:
        STATUS: done
        CHANGES: <summary>
    expects: "STATUS:\\s*done"
    extract:
      changes: "CHANGES:\\s*(.+)"
    on_fail:
      retry_step: explore
      feedback_var: implement_feedback
      max_retries: 2
      on_exhausted:
        escalate_to: human

  - id: verify
    agent: verifier
    input: |
      Task: {{task}}
      Scope: {{scope}}
      Changes: {{changes}}

      Verify the implementation. At the end, include:
        STATUS: done
      If there are issues:
        STATUS: failed
        ISSUES: <description>
    expects: "STATUS:\\s*done"
    extract:
      issues: "ISSUES:\\s*(.+)"
    on_fail:
      retry_step: implement
      feedback_var: verify_feedback
      max_retries: 3
      on_exhausted:
        escalate_to: human
```

---

## Workflow file locations

The server resolves workflow files based on the connected assistant (`clientInfo.Name`):

| Assistant                      | Global directory              | Repo directory              |
|-------------------------------|-------------------------------|------------------------------|
| `opencode`                    | `~/.config/opencode/workflows/` | `./.opencode/workflows/`   |
| `github-copilot` / `copilot`  | `~/.copilot/workflows/`       | `./.github/workflows-mcp/`   |
| `claude`                      | `~/.claude/workflows/`        | `./.claude/workflows/`       |
| _(unknown)_                   | `~/.config/w7s/workflows/`    | `./.w7s/workflows/`          |

**Precedence:** global directory > repo directory (global overrides repo for same workflow ID).
