package mcpserver_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	mcpgoclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	mcpgoserver "github.com/mark3labs/mcp-go/server"
	"github.com/matiasmartin00/w7s-mcp/internal/mcpserver"
)

// sendRequest sends a JSON-RPC request to the streamable HTTP handler.
func sendRequest(t *testing.T, handler http.Handler, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w
}

// initRequest is the standard MCP initialize request payload.
var initRequest = map[string]any{
	"jsonrpc": "2.0",
	"id":      1,
	"method":  "initialize",
	"params": map[string]any{
		"protocolVersion": mcp.LATEST_PROTOCOL_VERSION,
		"clientInfo": map[string]any{
			"name":    "test-client",
			"version": "1.0.0",
		},
	},
}

// TestNew_Initialize verifies that the MCP server responds to initialize.
func TestNew_Initialize(t *testing.T) {
	s := mcpserver.New()
	handler := mcpgoserver.NewStreamableHTTPServer(s)

	w := sendRequest(t, handler, initRequest)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Result struct {
			ServerInfo struct {
				Name string `json:"name"`
			} `json:"serverInfo"`
		} `json:"result"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v: body=%s", err, w.Body.String())
	}
	if resp.Result.ServerInfo.Name == "" {
		t.Errorf("expected non-empty serverInfo.name; body=%s", w.Body.String())
	}
}

// TestNew_ToolsList verifies tools/list returns at least one tool.
func TestNew_ToolsList(t *testing.T) {
	s := mcpserver.New()
	handler := mcpgoserver.NewStreamableHTTPServer(s)

	initResp := sendRequest(t, handler, initRequest)
	if initResp.Code != http.StatusOK {
		t.Fatalf("initialize failed %d: %s", initResp.Code, initResp.Body.String())
	}
	sessionID := initResp.Header().Get("Mcp-Session-Id")

	toolsReq := map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
	}
	b, _ := json.Marshal(toolsReq)
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if sessionID != "" {
		req.Header.Set("Mcp-Session-Id", sessionID)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("tools/list: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal tools/list response: %v: body=%s", err, w.Body.String())
	}
	if len(resp.Result.Tools) == 0 {
		t.Errorf("expected at least one tool; body=%s", w.Body.String())
	}
}

// TestNew_AllToolsRegistered verifies that all expected tools are listed via in-process client.
func TestNew_AllToolsRegistered(t *testing.T) {
	s := mcpserver.New()
	client, err := mcpgoclient.NewInProcessClient(s)
	if err != nil {
		t.Fatalf("create in-process client: %v", err)
	}
	if err := client.Start(context.Background()); err != nil {
		t.Fatalf("start client: %v", err)
	}
	defer client.Close()
	if _, err := client.Initialize(context.Background(), mcp.InitializeRequest{}); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	result, err := client.ListTools(context.Background(), mcp.ListToolsRequest{})
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}

	want := []string{"hello_world", "server_info", "start_run"}
	toolNames := make(map[string]bool, len(result.Tools))
	for _, tool := range result.Tools {
		toolNames[tool.Name] = true
	}
	for _, name := range want {
		if !toolNames[name] {
			t.Errorf("tool %q not found in tools/list; got: %v", name, result.Tools)
		}
	}
}
