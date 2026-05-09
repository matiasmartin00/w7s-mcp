package tools_test

import (
	"context"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/matiasmartin00/w7s-mcp/internal/tools"
)

func TestHelloWorld_Greets(t *testing.T) {
	client := startClient(t, newServerWith(tools.RegisterHelloWorld))

	result, err := client.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "hello_world",
			Arguments: map[string]any{"name": "Alice"},
		},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %v", result.Content)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected non-empty content")
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "Alice") {
		t.Errorf("expected 'Alice' in greeting, got: %s", text)
	}
}

func TestHelloWorld_Languages(t *testing.T) {
	client := startClient(t, newServerWith(tools.RegisterHelloWorld))

	cases := []struct {
		lang   string
		wantIn string
	}{
		{"en", "Hello"},
		{"es", "Hola"},
		{"fr", "Bonjour"},
	}
	for _, tc := range cases {
		t.Run(tc.lang, func(t *testing.T) {
			result, err := client.CallTool(context.Background(), mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name:      "hello_world",
					Arguments: map[string]any{"name": "Bob", "language": tc.lang},
				},
			})
			if err != nil {
				t.Fatalf("call tool: %v", err)
			}
			text := result.Content[0].(mcp.TextContent).Text
			if !strings.Contains(text, tc.wantIn) {
				t.Errorf("lang=%s: expected %q in %q", tc.lang, tc.wantIn, text)
			}
		})
	}
}

func TestHelloWorld_MissingName(t *testing.T) {
	client := startClient(t, newServerWith(tools.RegisterHelloWorld))

	result, err := client.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "hello_world",
			Arguments: map[string]any{},
		},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result when name is missing")
	}
}
