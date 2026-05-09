package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	mcpgoserver "github.com/mark3labs/mcp-go/server"
	"github.com/matiasmartin00/w7s-mcp/internal/mcpserver"
)

// buildHandler creates the Streamable HTTP handler used by main.
func buildHTTPHandler() http.Handler {
	s := mcpserver.New()
	return mcpgoserver.NewStreamableHTTPServer(s)
}

func sendJSONRPC(t *testing.T, handler http.Handler, body map[string]any, sessionID string) *httptest.ResponseRecorder {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if sessionID != "" {
		req.Header.Set("Mcp-Session-Id", sessionID)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w
}

// TestHTTP_Initialize verifies POST /mcp responds to initialize with 200.
func TestHTTP_Initialize(t *testing.T) {
	handler := buildHTTPHandler()
	w := sendJSONRPC(t, handler, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": mcp.LATEST_PROTOCOL_VERSION,
			"clientInfo":      map[string]any{"name": "test", "version": "0.0.1"},
		},
	}, "")

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
		t.Fatalf("unmarshal: %v; body=%s", err, w.Body.String())
	}
	if resp.Result.ServerInfo.Name != "w7s-mcp" {
		t.Errorf("expected serverInfo.name=w7s-mcp, got %q", resp.Result.ServerInfo.Name)
	}
}

// TestHTTP_ToolsList verifies tools/list returns at least one tool after initialize.
func TestHTTP_ToolsList(t *testing.T) {
	handler := buildHTTPHandler()

	initW := sendJSONRPC(t, handler, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": mcp.LATEST_PROTOCOL_VERSION,
			"clientInfo":      map[string]any{"name": "test", "version": "0.0.1"},
		},
	}, "")
	sessionID := initW.Header().Get("Mcp-Session-Id")

	w := sendJSONRPC(t, handler, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
	}, sessionID)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v; body=%s", err, w.Body.String())
	}
	if len(resp.Result.Tools) == 0 {
		t.Errorf("expected at least one tool, got zero; body=%s", w.Body.String())
	}
}

// TestHTTP_PortDefault verifies that defaultPort() returns "4004".
func TestHTTP_PortDefault(t *testing.T) {
	got := defaultPort()
	if got != "4004" {
		t.Errorf("expected defaultPort()=4004, got %q", got)
	}
}
