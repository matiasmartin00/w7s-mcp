---
layout: default
title: Getting Started
nav_order: 2
---

# Getting Started
{: .no_toc }

<details open markdown="block">
  <summary>Table of contents</summary>
  {: .text-delta }
- TOC
{:toc}
</details>

---

## Requirements

- Go >= 1.22
- SQLite (handled automatically via the pure-Go `modernc.org/sqlite` driver — no `cgo` required)

---

## Run locally

```bash
go run ./cmd/w7s-mcp
```

The server starts on `http://localhost:4004/mcp` by default.

### Change the port

```bash
PORT=8080 go run ./cmd/w7s-mcp
```

### Health check

```bash
curl http://localhost:4004/healthz
# ok
```

---

## Environment variables

| Variable    | Default            | Description                              |
|-------------|--------------------|------------------------------------------|
| `W7S_DB_DIR`| `~/.config/w7s`    | Directory where `data.db` is created     |
| `PORT`      | `4004`             | Port the HTTP server listens on          |

---

## Register with an MCP client

Add this to your MCP client configuration:

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

## Test with curl

```bash
# Step 1 — initialize (save the Mcp-Session-Id header from the response)
curl -si -X POST http://localhost:4004/mcp \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","clientInfo":{"name":"curl","version":"0.0.1"}}}'

# Step 2 — list tools
curl -s -X POST http://localhost:4004/mcp \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -H "Mcp-Session-Id: <SESSION_ID>" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/list"}'
```

---

## Run tests

```bash
go test ./...
```

For end-to-end tests only:

```bash
go test ./internal/e2e/...
```

---

## Build

```bash
go build -o w7s-mcp ./cmd/w7s-mcp
```
