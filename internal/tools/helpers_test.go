package tools_test

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	mcpgoclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	mcpgoserver "github.com/mark3labs/mcp-go/server"
)

// newServerWith creates a minimal MCPServer with a single tool registered.
func newServerWith(register func(*mcpgoserver.MCPServer)) *mcpgoserver.MCPServer {
	s := mcpgoserver.NewMCPServer("test", "0.0.0", mcpgoserver.WithToolCapabilities(true))
	register(s)
	return s
}

// startClient creates, starts, and initializes an in-process client against s.
// It registers a cleanup to close the client when the test ends.
func startClient(t *testing.T, s *mcpgoserver.MCPServer) *mcpgoclient.Client {
	t.Helper()
	client, err := mcpgoclient.NewInProcessClient(s)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	if err := client.Start(context.Background()); err != nil {
		t.Fatalf("start client: %v", err)
	}
	t.Cleanup(func() { client.Close() })
	if _, err := client.Initialize(context.Background(), mcp.InitializeRequest{}); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	return client
}

// loaderTestdataPath returns the absolute path to a file in internal/loader/testdata.
func loaderTestdataPath(filename string) string {
	_, self, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(self), "..", "loader", "testdata", filename)
}
