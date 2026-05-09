---
layout: home
title: w7s-mcp
nav_order: 1
description: "Workflow Orchestrator MCP Server — deterministic multi-agent orchestration over MCP."
permalink: /
---

# w7s-mcp

**w7s-mcp** is an [MCP](https://modelcontextprotocol.io/) server that acts as an external orchestration engine for AI assistants.

It lets the Lead Agent (LLM) delegate all control logic to the server: step advancement, retries, escalation, and variable passing — all persisted in SQLite.

---

## Why w7s-mcp?

AI assistants lack a **deterministic and persistent** mechanism to orchestrate multi-agent workflows. Without an external orchestrator the agent decides order, retries, and state on its own — introducing non-determinism and state loss on context compactions.

w7s-mcp solves this by:

- Maintaining **execution state in SQLite** (no external infrastructure)
- Allowing workflow definitions in **YAML** with JSON Schema validation
- Supporting **retries** and **escalation** as declared in the workflow
- Operating as a standard **MCP HTTP server** (Streamable HTTP, JSON-RPC 2.0)

---

## Quick links

- [Getting Started](getting-started.md)
- [Architecture](architecture.md)
- [Tool Reference](tool-reference.md)
- [Workflow Authoring](workflow-authoring.md)
- [Contributing to Docs](contributing.md)
