package tools

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func RegisterHelloWorld(s *server.MCPServer) {
	tool := mcp.NewTool("hello_world",
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

	s.AddTool(tool, helloWorldHandler)
}

func helloWorldHandler(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	slog.Info("hello_world called", "name", name, "lang", lang)
	return mcp.NewToolResultText(fmt.Sprintf("%s, %s! 👋 — w7s-mcp is alive.", greeting, name)), nil
}
