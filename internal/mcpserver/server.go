// Package mcpserver constructs and configures the MCP server instance.
// It is transport-agnostic: callers decide how to expose the server
// (stdio, Streamable HTTP, etc.).
package mcpserver

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/matiasmartin00/w7s-mcp/internal/store"
	"github.com/matiasmartin00/w7s-mcp/internal/tools"
)

// New builds and returns a fully configured MCP server.
// All tool/resource/prompt registrations happen here; transport is not concerned.
func New() *server.MCPServer {
	s := server.NewMCPServer(
		"w7s-mcp",
		"0.1.0",
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true),
		server.WithHooks(buildHooks()),
		server.WithRecovery(),
	)

	dbDir := os.Getenv("W7S_DB_DIR")
	if dbDir == "" {
		home, _ := os.UserHomeDir()
		dbDir = filepath.Join(home, ".config", "w7s")
	}
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		slog.Error("failed to create db directory", "path", dbDir, "error", err)
	}
	st, err := store.Open(filepath.Join(dbDir, "data.db"))
	if err != nil {
		slog.Error("failed to open store", "error", err)
		panic(err)
	}

	tools.RegisterHelloWorld(s)
	tools.RegisterServerInfo(s)
	tools.RegisterStartRun(s, st)
	tools.RegisterGetRunStatus(s, st)
	tools.RegisterGetNextStep(s, st)
	tools.RegisterCompleteStep(s, st)
	tools.RegisterFailStep(s, st)

	registerResources(s)
	registerPrompts(s)

	return s
}

// buildHooks wires lifecycle observability into every MCP request.
func buildHooks() *server.Hooks {
	hooks := &server.Hooks{}

	hooks.AddBeforeAny(func(_ context.Context, id any, method mcp.MCPMethod, _ any) {
		slog.Debug("mcp request received", "method", method, "id", id)
	})
	hooks.AddOnSuccess(func(_ context.Context, id any, method mcp.MCPMethod, _ any, _ any) {
		slog.Debug("mcp request completed", "method", method, "id", id)
	})
	hooks.AddOnError(func(_ context.Context, id any, method mcp.MCPMethod, _ any, err error) {
		slog.Error("mcp request error", "method", method, "id", id, "error", err)
	})
	hooks.AddAfterInitialize(func(_ context.Context, _ any, msg *mcp.InitializeRequest, _ *mcp.InitializeResult) {
		slog.Info("client initialized",
			"client_name", msg.Params.ClientInfo.Name,
			"client_version", msg.Params.ClientInfo.Version,
			"protocol_version", msg.Params.ProtocolVersion,
		)
	})

	return hooks
}

// registerResources adds all MCP resources to s.
func registerResources(s *server.MCPServer) {
	aboutResource := mcp.NewResource(
		"w7s://about",
		"About w7s-mcp",
		mcp.WithResourceDescription("Static description of this MCP server"),
		mcp.WithMIMEType("text/plain"),
	)

	s.AddResource(aboutResource, func(_ context.Context, _ mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		slog.Debug("resource read", "uri", "w7s://about")
		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "w7s://about",
				MIMEType: "text/plain",
				Text: `w7s-mcp — workflow orchestrator MCP server
Version  : 0.1.0
Tools    : hello_world, server_info, start_run
Resources: w7s://about
Prompts  : greet
`,
			},
		}, nil
	})
}

// registerPrompts adds all MCP prompts to s.
func registerPrompts(s *server.MCPServer) {
	greetPrompt := mcp.NewPrompt("greet",
		mcp.WithPromptDescription("Generate a greeting message for the given person"),
		mcp.WithArgument("name",
			mcp.ArgumentDescription("Name of the person to greet"),
			mcp.RequiredArgument(),
		),
	)

	s.AddPrompt(greetPrompt, func(_ context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		name := req.Params.Arguments["name"]
		if name == "" {
			name = "friend"
		}
		slog.Debug("prompt get", "prompt", "greet", "name", name)
		return mcp.NewGetPromptResult(
			"Greeting prompt",
			[]mcp.PromptMessage{
				mcp.NewPromptMessage(
					mcp.RoleAssistant,
					mcp.NewTextContent("You are a friendly assistant. Greet "+name+" warmly and ask how you can help them today."),
				),
			},
		), nil
	})
}
