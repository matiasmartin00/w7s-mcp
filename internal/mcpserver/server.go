// Package mcpserver constructs and configures the MCP server instance.
// It is transport-agnostic: callers decide how to expose the server
// (stdio, Streamable HTTP, etc.).
package mcpserver

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// startTime is captured at package init for uptime reporting.
var startTime = time.Now()

// New builds and returns a fully configured MCP server.
// All tool/resource/prompt registrations happen here; transport is not concerned.
func New() *server.MCPServer {
	hooks := buildHooks()

	s := server.NewMCPServer(
		"w7s-mcp",
		"0.1.0",
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true),
		server.WithHooks(hooks),
		server.WithRecovery(),
	)

	registerTools(s)
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

// registerTools adds all MCP tools to s.
func registerTools(s *server.MCPServer) {
	// ── Tool: hello_world ────────────────────────────────────────────────────
	helloTool := mcp.NewTool("hello_world",
		mcp.WithDescription("Greet someone by name"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the person to greet"),
		),
		mcp.WithString("language",
			mcp.Description("Language for the greeting (en, es, fr). Defaults to en."),
			mcp.Enum("en", "es", "fr"),
		),
	)

	s.AddTool(helloTool, func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := req.RequireString("name")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		lang := req.GetString("language", "en")

		greetings := map[string]string{"en": "Hello", "es": "Hola", "fr": "Bonjour"}
		greeting, ok := greetings[lang]
		if !ok {
			greeting = "Hello"
		}

		msg := fmt.Sprintf("%s, %s! 👋 — w7s-mcp is alive.", greeting, name)
		slog.Info("hello_world called", "name", name, "lang", lang)
		return mcp.NewToolResultText(msg), nil
	})

	// ── Tool: server_info ────────────────────────────────────────────────────
	infoTool := mcp.NewTool("server_info",
		mcp.WithDescription("Return runtime metadata about this MCP server"),
	)

	s.AddTool(infoTool, func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		info := fmt.Sprintf("name=w7s-mcp version=0.1.0 uptime=%s", time.Since(startTime).Round(time.Second))
		slog.Info("server_info called", "info", info)
		return mcp.NewToolResultText(info), nil
	})
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
				Text: `w7s-mcp — hello-world MCP server
Version  : 0.1.0
Tools    : hello_world, server_info
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
					mcp.NewTextContent(fmt.Sprintf(
						"You are a friendly assistant. Greet %s warmly and ask how you can help them today.",
						name,
					)),
				),
			},
		), nil
	})
}
