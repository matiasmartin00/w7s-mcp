package mcpserver_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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
		t.Errorf("expected non-empty serverInfo.name, got empty; body=%s", w.Body.String())
	}
}

// TestNew_ToolsList verifies tools/list returns the declared tools.
func TestNew_ToolsList(t *testing.T) {
	s := mcpserver.New()
	handler := mcpgoserver.NewStreamableHTTPServer(s)

	// Must initialize first to establish a session.
	initResp := sendRequest(t, handler, initRequest)
	if initResp.Code != http.StatusOK {
		t.Fatalf("initialize failed %d: %s", initResp.Code, initResp.Body.String())
	}

	// Extract session ID from Mcp-Session-Id header.
	sessionID := initResp.Header().Get("Mcp-Session-Id")

	// Build tools/list request.
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
		t.Errorf("expected at least one tool, got zero; body=%s", w.Body.String())
	}
}

// TestNew_HelloWorldTool verifies that hello_world tool is declared.
func TestNew_HelloWorldTool(t *testing.T) {
	s := mcpserver.New()
	handler := mcpgoserver.NewStreamableHTTPServer(s)

	initResp := sendRequest(t, handler, initRequest)
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

	var resp struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp) //nolint:errcheck

	found := false
	for _, tool := range resp.Result.Tools {
		if tool.Name == "hello_world" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("hello_world tool not found in tools/list; body=%s", w.Body.String())
	}
}

// TestNew_HelloWorldTool_Call verifies that the hello_world tool can be called.
func TestNew_HelloWorldTool_Call(t *testing.T) {
	s := mcpserver.New()
	handler := mcpgoserver.NewStreamableHTTPServer(s)

	initResp := sendRequest(t, handler, initRequest)
	sessionID := initResp.Header().Get("Mcp-Session-Id")

	callReq := map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "hello_world",
			"arguments": map[string]any{
				"name": "TDD",
			},
		},
	}
	b, _ := json.Marshal(callReq)
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if sessionID != "" {
		req.Header.Set("Mcp-Session-Id", sessionID)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("tools/call: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// We just need a non-error result with content.
	var resp struct {
		Result struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
		Error *json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal call response: %v", err)
	}
	if resp.Error != nil {
		t.Errorf("expected no error, got: %s", *resp.Error)
	}
	if len(resp.Result.Content) == 0 {
		t.Errorf("expected non-empty content in tool result; body=%s", w.Body.String())
	}
}

// TestNew_StartRunTool_Listed verifies that start_run tool is present in tools/list.
// RED: start_run is not yet registered — this test will fail until server.go is updated.
func TestNew_StartRunTool_Listed(t *testing.T) {
	s := mcpserver.New()
	client, err := mcpgoclient.NewInProcessClient(s)
	if err != nil {
		t.Fatalf("create in-process client: %v", err)
	}
	if err := client.Start(context.Background()); err != nil {
		t.Fatalf("start in-process client: %v", err)
	}
	defer client.Close()

	if _, err := client.Initialize(context.Background(), mcp.InitializeRequest{}); err != nil {
		t.Fatalf("initialize in-process client: %v", err)
	}

	result, err := client.ListTools(context.Background(), mcp.ListToolsRequest{})
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}

	found := false
	for _, tool := range result.Tools {
		if tool.Name == "start_run" {
			found = true
			break
		}
	}
	if !found {
		names := make([]string, 0, len(result.Tools))
		for _, tool := range result.Tools {
			names = append(names, tool.Name)
		}
		t.Errorf("start_run not found in tools list; tools=%v", names)
	}
}

// TestNew_StartRunTool_Call_ValidWorkflow verifies start_run tool returns success for a valid workflow file.
func TestNew_StartRunTool_Call_ValidWorkflow(t *testing.T) {
	s := mcpserver.New()
	client, err := mcpgoclient.NewInProcessClient(s)
	if err != nil {
		t.Fatalf("create in-process client: %v", err)
	}
	if err := client.Start(context.Background()); err != nil {
		t.Fatalf("start in-process client: %v", err)
	}
	defer client.Close()

	if _, err := client.Initialize(context.Background(), mcp.InitializeRequest{}); err != nil {
		t.Fatalf("initialize in-process client: %v", err)
	}

	result, err := client.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "start_run",
			Arguments: map[string]any{
				"workflow_path": "../../internal/loader/testdata/valid.yaml",
			},
		},
	})
	if err != nil {
		t.Fatalf("call start_run: %v", err)
	}
	if result.IsError {
		t.Errorf("expected success, got error result: %v", result.Content)
	}
	if len(result.Content) == 0 {
		t.Errorf("expected non-empty content in tool result")
	}
	// Verify the response mentions the workflow name.
	text := result.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "greet-user") {
		t.Errorf("expected response to mention workflow name 'greet-user', got: %s", text)
	}
}

// TestNew_StartRunTool_Call_InvalidPath verifies start_run returns an error result for missing file.
func TestNew_StartRunTool_Call_InvalidPath(t *testing.T) {
	s := mcpserver.New()
	client, err := mcpgoclient.NewInProcessClient(s)
	if err != nil {
		t.Fatalf("create in-process client: %v", err)
	}
	if err := client.Start(context.Background()); err != nil {
		t.Fatalf("start in-process client: %v", err)
	}
	defer client.Close()

	if _, err := client.Initialize(context.Background(), mcp.InitializeRequest{}); err != nil {
		t.Fatalf("initialize in-process client: %v", err)
	}

	result, err := client.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "start_run",
			Arguments: map[string]any{
				"workflow_path": "/nonexistent/path/workflow.yaml",
			},
		},
	})
	if err != nil {
		t.Fatalf("call start_run: %v", err)
	}
	if !result.IsError {
		t.Errorf("expected error result for missing file, got success: %v", result.Content)
	}
}

// TestNew_ServerInfoTool verifies server_info tool is present via direct in-process call.
func TestNew_ServerInfoTool(t *testing.T) {
	s := mcpserver.New()

	// Use in-process client for a pure unit test on the server.
	client, err := mcpgoclient.NewInProcessClient(s)
	if err != nil {
		t.Fatalf("create in-process client: %v", err)
	}
	if err := client.Start(context.Background()); err != nil {
		t.Fatalf("start in-process client: %v", err)
	}
	defer client.Close()

	if _, err := client.Initialize(context.Background(), mcp.InitializeRequest{}); err != nil {
		t.Fatalf("initialize in-process client: %v", err)
	}

	result, err := client.ListTools(context.Background(), mcp.ListToolsRequest{})
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}

	found := false
	for _, tool := range result.Tools {
		if tool.Name == "server_info" {
			found = true
			break
		}
	}
	if !found {
		names := make([]string, 0, len(result.Tools))
		for _, tool := range result.Tools {
			names = append(names, tool.Name)
		}
		t.Errorf("server_info not found; tools=%v", names)
	}
}
