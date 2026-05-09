# Skill Registry

## Project Conventions
- Go MCP server using `github.com/mark3labs/mcp-go`
- Entry point: `main.go`
- Structured logging via `log/slog`
- stdio transport via `server.ServeStdio`

## Detected User Skills
- `sdd-init` — project bootstrap and SDD initialization
- `golang-cli` — Go CLI/server shape and entrypoint conventions
- `golang-code-style` — Go formatting and conventions
- `golang-error-handling` — error handling patterns in Go
- `golang-testing` — Go test patterns and coverage
- `golang-lint` — linting and static checks

## Project Guidance
- Prefer Go-native tooling (`go test`, `gofmt`, `go vet`)
- Keep stdio transport output clean; logs go to stderr
