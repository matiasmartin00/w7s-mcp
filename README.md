# w7s-mcp

A [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) server written in Go that acts as a **deterministic workflow orchestrator** for AI agents.

📚 **[Full documentation → matiasmartin00.github.io/w7s-mcp](https://matiasmartin00.github.io/w7s-mcp)**

---

## What it does

AI assistants lack a persistent mechanism to orchestrate multi-agent workflows. `w7s-mcp` solves this by maintaining execution state in SQLite and exposing workflow control via MCP tools:

- Define workflows in **YAML** with JSON Schema validation
- Support **retries** and **escalation** as declared in the workflow
- Operate as an **MCP HTTP server** (Streamable HTTP, JSON-RPC 2.0)

---

## Quick Start

### Requirements

- Go >= 1.22

### Run locally

```bash
go run ./cmd/w7s-mcp
```

Server starts at `http://localhost:4004/mcp`.

### Environment variables

| Variable     | Default          | Description                          |
|--------------|------------------|--------------------------------------|
| `W7S_DB_DIR` | `~/.config/w7s`  | Directory where `data.db` is created |
| `PORT`       | `4004`           | HTTP port                            |

### Register with an MCP client

```json
{
  "mcp": {
    "w7s": {
      "type": "remote",
      "url": "http://localhost:4004/mcp"
    }
  }
}
```

---

## MCP Tools

| Tool             | Description                                                  |
|------------------|--------------------------------------------------------------|
| `start_run`      | Start a new workflow execution                               |
| `get_next_step`  | Get the next step to execute (interpolated prompt + agent)   |
| `complete_step`  | Mark a step as done, extract variables from agent output     |
| `fail_step`      | Mark a step as failed, trigger retry or escalation           |
| `get_run_status` | Inspect full run state: steps, variables, status             |

---

## Development

```bash
# Run tests
go test ./...

# Build
go build ./cmd/w7s-mcp

# Vet
go vet ./...
```

---

## Documentation

- [Getting Started](https://matiasmartin00.github.io/w7s-mcp/getting-started)
- [Architecture](https://matiasmartin00.github.io/w7s-mcp/architecture)
- [Tool Reference](https://matiasmartin00.github.io/w7s-mcp/tool-reference)
- [Workflow Authoring](https://matiasmartin00.github.io/w7s-mcp/workflow-authoring)

---

## License

MIT
