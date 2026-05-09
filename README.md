# w7s-mcp

A minimal [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) server written in Go, using [mcp-go](https://github.com/mark3labs/mcp-go) and the **Streamable HTTP** transport.

## Quick Start

### Requirements

- Go 1.22+

### Run locally

```bash
go run ./cmd/w7s-mcp
```

The server listens on `http://localhost:4004/mcp` by default.

### Change the port

```bash
PORT=8080 go run ./cmd/w7s-mcp
```

### Health check

```bash
curl http://localhost:4004/healthz
# ok
```

## MCP Endpoint

| Method | Path   | Description                          |
|--------|--------|--------------------------------------|
| POST   | `/mcp` | Streamable HTTP MCP transport        |

### Test with curl

```bash
# initialize
curl -s -X POST http://localhost:4004/mcp \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","clientInfo":{"name":"curl","version":"0.0.1"}}}'

# list tools (replace <SESSION_ID> with the Mcp-Session-Id from the initialize response header)
curl -s -X POST http://localhost:4004/mcp \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -H "Mcp-Session-Id: <SESSION_ID>" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/list"}'
```

## Tools

| Tool          | Description                            |
|---------------|----------------------------------------|
| `hello_world` | Greet someone by name (en / es / fr)   |
| `server_info` | Return runtime metadata of the server  |

## Development

```bash
# Run tests
go test ./...

# Build
go build ./cmd/w7s-mcp

# Vet
go vet ./...
```

## Architecture

```
cmd/w7s-mcp/main.go             — HTTP bootstrap, graceful shutdown
internal/mcpserver/server.go    — MCP server construction (transport-agnostic)
```

Business logic lives in `internal/mcpserver`; the transport layer (`cmd/w7s-mcp/main.go`) only wires HTTP.
