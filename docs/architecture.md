---
layout: default
title: Architecture
nav_order: 3
---

# Architecture
{: .no_toc }

<details open markdown="block">
  <summary>Table of contents</summary>
  {: .text-delta }
- TOC
{:toc}
</details>

---

## Overview

```
┌─────────────────────────────────────────────────────┐
│              AI Assistant / MCP Client              │
│   Lead Agent (LLM)                                  │
│   └── invokes tools: start_run, get_next_step, etc. │
└────────────────────┬────────────────────────────────┘
                     │ Streamable HTTP (JSON-RPC 2.0)
                     ▼
┌─────────────────────────────────────────────────────┐
│               w7s-mcp (MCP Server)                  │
│                                                     │
│  Transport Layer  ──►  Tool Dispatcher              │
│                              │                      │
│              ┌───────────────┴──────────────┐       │
│              ▼                              ▼       │
│       Workflow Loader            Orchestration Engine│
│       (YAML + JSON Schema)       (step, retry, esc) │
│                                             │       │
│                                    State Store      │
│                                    (SQLite)         │
└─────────────────────────────────────────────────────┘
                                      │
                                      ▼
                               ~/.config/w7s/data.db
```

---

## Components

### Transport Layer

Manages the Streamable HTTP lifecycle: single `POST /mcp` endpoint accepts all JSON-RPC 2.0 messages. Implemented via `server.NewStreamableHTTPServer` from `mcp-go`.

### Tool Dispatcher

Routes each `tools/call` to its handler and serializes the response. Contains no business logic.

### Workflow Loader

Reads a YAML file from disk, validates it against the workflow JSON Schema (Draft 2020-12), and returns a typed data structure. Operates at `start_run` time, not on each step.

**Path resolution order:**

1. Absolute path (if the argument is an absolute path)
2. Assistant's **global** directory (e.g. `~/.config/opencode/workflows/`)
3. Assistant's **repo** directory (e.g. `./.opencode/workflows/`)

The assistant is identified by `clientInfo.Name` from the MCP `initialize` message.

### Orchestration Engine

Contains all business logic: step advancement, retry calculation, step reset for retry, and escalation. The server **never** evaluates `expects` — that responsibility belongs to the Lead Agent.

### State Store

Data access layer over SQLite. Exposes CRUD operations on `runs`, `steps`, and `variables`.

---

## Data Model

### runs

| Field         | Type    | Description                              |
|---------------|---------|------------------------------------------|
| `id`          | TEXT PK | UUID generated at `start_run`            |
| `workflow_id` | TEXT    | ID or path of the YAML workflow          |
| `task`        | TEXT    | Initial task description                 |
| `status`      | TEXT    | `running` \| `done` \| `escalated` \| `failed` |
| `created_at`  | INTEGER | Unix timestamp (ms)                      |

### steps

| Field       | Type    | Description                              |
|-------------|---------|------------------------------------------|
| `id`        | TEXT PK | `{run_id}:{step_id}`                     |
| `run_id`    | TEXT FK | Reference to `runs.id`                   |
| `step_id`   | TEXT    | Step ID as defined in workflow.yml       |
| `status`    | TEXT    | `pending` \| `running` \| `done` \| `failed` |
| `attempt`   | INTEGER | Times marked as `running`                |
| `output`    | TEXT    | Full agent output (nullable)             |
| `created_at`| INTEGER | Unix timestamp (ms)                      |

### variables

| Field    | Type    | Description                          |
|----------|---------|--------------------------------------|
| `run_id` | TEXT FK | Reference to `runs.id`               |
| `key`    | TEXT    | Lowercase key                        |
| `value`  | TEXT    | Value extracted from agent output    |
| PK       | —       | `(run_id, key)` — upsert on write    |

---

## Project structure

```
w7s-mcp/
├── cmd/w7s-mcp/main.go       — HTTP bootstrap, graceful shutdown
├── internal/
│   ├── domain/               — Canonical domain models and MCP tool DTOs
│   ├── e2e/                  — Black-box end-to-end tests
│   ├── loader/               — YAML loader + JSON Schema validation
│   ├── mcpserver/            — MCP server construction (tool registry)
│   ├── parser/               — {{variable}} interpolation + regex extraction
│   ├── store/                — SQLite state store (runs, steps, variables)
│   └── tools/                — MCP tool handlers
├── docs/                     — This documentation site
├── workflows/                — Example workflow YAML files
└── workflow.schema.json      — JSON Schema (Draft 2020-12) for workflows
```

---

## Communication protocol

The server implements **MCP over Streamable HTTP** (JSON-RPC 2.0, MCP spec 2025-03-26+):

- **Single endpoint:** `POST /mcp` — receives all client messages, streams responses
- Supported messages: `initialize`, `tools/list`, `tools/call`
- Session tracking via `Mcp-Session-Id` header
