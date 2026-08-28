// Package server implements the saka HTTP and MCP servers.
package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	saka "github.com/you/saka"
)

// ---- JSON-RPC 2.0 over stdio ----

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

// MCPServer serves saka over stdio for MCP clients (Claude Desktop, etc.).
type MCPServer struct {
	engine saka.Searcher
}

func NewMCP(engine saka.Searcher) *MCPServer {
	return &MCPServer{engine: engine}
}

// ServeStdio blocks reading line-delimited JSON-RPC requests from stdin
// and writing responses to stdout. Run with: saka serve --mcp
func (s *MCPServer) ServeStdio(ctx context.Context) error {
	return s.Serve(ctx, os.Stdin, os.Stdout)
}

func (s *MCPServer) Serve(ctx context.Context, in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 1<<20), 1<<20) // 1MB lines
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var req rpcRequest
		if err := json.Unmarshal(line, &req); err != nil {
			s.write(out, rpcResponse{JSONRPC: "2.0", Error: &rpcError{
				Code: -32700, Message: "parse error",
			}})
			continue
		}
		result, rpcErr := s.dispatch(ctx, req.Method, req.Params)
		resp := rpcResponse{JSONRPC: "2.0", ID: req.ID}
		if rpcErr != nil {
			resp.Error = rpcErr
		} else {
			b, _ := json.Marshal(result)
			resp.Result = b
		}
		s.write(out, resp)
	}
	return scanner.Err()
}

func (s *MCPServer) write(out io.Writer, resp rpcResponse) {
	b, _ := json.Marshal(resp)
	fmt.Fprintf(out, "%s\n", b)
}

type mcpTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

type mcpToolsResult struct {
	Tools []mcpTool `json:"tools"`
}

type mcpCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

func (s *MCPServer) dispatch(ctx context.Context, method string, params json.RawMessage) (any, *rpcError) {
	switch method {
	case "initialize":
		return map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "saka", "version": "0.1.0"},
		}, nil

	case "notifications/initialized":
		return nil, nil // notification: no response needed (id is null, tolerated)

	case "tools/list":
		return mcpToolsResult{Tools: []mcpTool{
			{
				Name:        "web_search",
				Description: "Search the web for free. Returns title, URL, snippet, position.",
				InputSchema: json.RawMessage(saka.SearchSchema()),
			},
			{
				Name:        "fetch_page",
				Description: "Fetch a URL and return extracted readable article text.",
				InputSchema: json.RawMessage(saka.FetchSchema()),
			},
		}}, nil

	case "tools/call":
		var p mcpCallParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &rpcError{Code: -32602, Message: "invalid params"}
		}
		text, err := saka.ExecuteTool(ctx, s.engine, p.Name, p.Arguments)
		if err != nil {
			// Tool execution errors are reported inside the result per MCP spec.
			return map[string]any{
				"content": []map[string]string{{"type": "text", "text": "error: " + err.Error()}},
				"isError": true,
			}, nil
		}
		return map[string]any{
			"content": []map[string]string{{"type": "text", "text": text}},
		}, nil

	default:
		return nil, &rpcError{Code: -32601, Message: "method not found: " + method}
	}
}
