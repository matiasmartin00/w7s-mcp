package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	// Structured logger — writes to stderr so it doesn't pollute the MCP stdio transport.
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	slog.Info("starting w7s-mcp server")

	// ── Hooks ────────────────────────────────────────────────────────────────
	// Hooks provide lifecycle observability for every MCP request.
	hooks := &server.Hooks{}

	hooks.AddBeforeAny(func(ctx context.Context, id any, method mcp.MCPMethod, message any) {
		slog.Debug("mcp request received",
			"method", method,
			"id", id,
		)
	})

	hooks.AddOnSuccess(func(ctx context.Context, id any, method mcp.MCPMethod, message any, result any) {
		slog.Debug("mcp request completed",
			"method", method,
			"id", id,
		)
	})

	hooks.AddOnError(func(ctx context.Context, id any, method mcp.MCPMethod, message any, err error) {
		slog.Error("mcp request error",
			"method", method,
			"id", id,
			"error", err,
		)
	})

	hooks.AddAfterInitialize(func(ctx context.Context, id any, message *mcp.InitializeRequest, result *mcp.InitializeResult) {
		slog.Info("client initialized",
			"client_name", message.Params.ClientInfo.Name,
			"client_version", message.Params.ClientInfo.Version,
			"protocol_version", message.Params.ProtocolVersion,
		)
	})

	// ── Server ───────────────────────────────────────────────────────────────
	s := server.NewMCPServer(
		"w7s-mcp",
		"0.1.0",
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true), // subscribe + list-changed
		server.WithHooks(hooks),
		server.WithRecovery(), // recover from panics in handlers
	)

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

	s.AddTool(helloTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := req.RequireString("name")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		lang := req.GetString("language", "en")

		greetings := map[string]string{
			"en": "Hello",
			"es": "Hola",
			"fr": "Bonjour",
		}
		greeting, ok := greetings[lang]
		if !ok {
			greeting = "Hello"
		}

		msg := fmt.Sprintf("%s, %s! 👋 — w7s-mcp is alive.", greeting, name)

		slog.Info("hello_world tool called",
			"name", name,
			"language", lang,
			"response", msg,
		)

		// Send a notification to all connected clients after the tool runs.
		// In stdio mode this is a no-op (no separate sessions), but it demonstrates
		// the pattern for SSE/HTTP transports.
		go func() {
			time.Sleep(100 * time.Millisecond)
			s.SendNotificationToAllClients("notifications/message", map[string]any{
				"level":   "info",
				"message": fmt.Sprintf("Tool hello_world was called for %q", name),
				"ts":      time.Now().UTC().Format(time.RFC3339),
			})
		}()

		return mcp.NewToolResultText(msg), nil
	})

	// ── Tool: server_info ────────────────────────────────────────────────────
	infoTool := mcp.NewTool("server_info",
		mcp.WithDescription("Return runtime metadata about this MCP server"),
	)

	s.AddTool(infoTool, func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		info := fmt.Sprintf(
			"name=w7s-mcp version=0.1.0 uptime=%s",
			time.Since(startTime).Round(time.Second),
		)
		slog.Info("server_info tool called", "info", info)
		return mcp.NewToolResultText(info), nil
	})

	// ── Resource: about ──────────────────────────────────────────────────────
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
								Transport: stdio (default)
								Tools    : hello_world, server_info
								Resources: w7s://about
								Prompts  : greet
								`,
			},
		}, nil
	})

	// ── Prompt: greet ────────────────────────────────────────────────────────
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

	// ── Start ────────────────────────────────────────────────────────────────
	slog.Info("w7s-mcp server ready", "transport", "stdio")
	if err := server.ServeStdio(s); err != nil {
		slog.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}

// startTime is used by the server_info tool to report uptime.
var startTime = time.Now()
