// Package mcp implements a Model Context Protocol server for PM.
//
// It exposes PM's project memory functionality as MCP tools over stdio
// transport, enabling LLM agents to read, create, and update project data.
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

const protocolVersion = "2024-11-05"

// Server is an MCP server that exposes PM tools over stdio transport.
type Server struct {
	name    string
	version string
	tools   []Tool
}

// Tool defines an MCP tool: its schema and handler.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
	Handler     func(context.Context, json.RawMessage) (string, error) `json:"-"`
}

// NewServer creates a new MCP server.
func NewServer(name, version string) *Server {
	return &Server{name: name, version: version}
}

// AddTool registers an MCP tool.
func (s *Server) AddTool(t Tool) {
	s.tools = append(s.tools, t)
}

// Run starts the MCP stdio server loop. It reads JSON-RPC messages from
// stdin (Content-Length framed) and writes responses to stdout.
// The context controls server lifecycle: cancel it for graceful shutdown.
func (s *Server) Run(ctx context.Context) error {
	return s.runWithWriter(ctx, os.Stdout)
}

// runWithWriter is like Run but allows specifying the output writer (for testing).
func (s *Server) runWithWriter(ctx context.Context, w io.Writer) error {
	r := newMessageReader(os.Stdin)
	initialized := false

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		body, err := r.readMessage()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			slog.Error("mcp: read error, skipping", "error", err)
			continue
		}

		var req json.RawMessage
		if err := json.Unmarshal(body, &req); err != nil {
			sendError(w, nil, -32700, "Parse error", "")
			continue
		}

		s.handleMessage(ctx, req, w, &initialized)
	}
}

// jsonrpcMessage is the outer JSON-RPC 2.0 envelope.
type jsonrpcMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int            `json:"id"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonrpcError   `json:"error,omitempty"`
}

type jsonrpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

func (s *Server) handleMessage(ctx context.Context, body json.RawMessage, w io.Writer, initialized *bool) {
	var msg jsonrpcMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		sendError(w, nil, -32700, "Parse error", "")
		return
	}

	isNotification := msg.ID == nil

	switch msg.Method {
	case "initialize":
		s.handleInitialize(w, msg.ID)

	case "notifications/initialized":
		*initialized = true

	case "tools/list":
		if !*initialized {
			sendError(w, msg.ID, -32000, "Not initialized", "")
			return
		}
		s.handleToolsList(w, msg.ID)

	case "tools/call":
		if !*initialized {
			sendError(w, msg.ID, -32000, "Not initialized", "")
			return
		}
		s.handleToolCall(ctx, w, msg.ID, msg.Params)

	default:
		if !isNotification {
			sendError(w, msg.ID, -32601, fmt.Sprintf("Method not found: %s", msg.Method), "")
		}
	}
}

func (s *Server) handleInitialize(w io.Writer, id *int) {
	result := map[string]any{
		"protocolVersion": protocolVersion,
		"capabilities": map[string]any{
			"tools": map[string]any{},
		},
		"serverInfo": map[string]string{
			"name":    s.name,
			"version": s.version,
		},
	}
	sendResult(w, id, result)
}

func (s *Server) handleToolsList(w io.Writer, id *int) {
	tools := make([]Tool, len(s.tools))
	for i, t := range s.tools {
		tools[i] = Tool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.InputSchema,
		}
	}
	sendResult(w, id, map[string]any{"tools": tools})
}

func (s *Server) handleToolCall(ctx context.Context, w io.Writer, id *int, params json.RawMessage) {
	var call struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(params, &call); err != nil {
		sendError(w, id, -32602, "Invalid params", err.Error())
		return
	}

	for _, t := range s.tools {
		if t.Name == call.Name {
			text, err := t.Handler(ctx, call.Arguments)
			if err != nil {
				sendError(w, id, -32603, err.Error(), "")
				return
			}
			sendResult(w, id, map[string]any{
				"content": []map[string]string{
					{"type": "text", "text": text},
				},
			})
			return
		}
	}

	sendError(w, id, -32602, fmt.Sprintf("Unknown tool: %s", call.Name), "")
}

func sendResult(w io.Writer, id *int, result any) {
	resp := jsonrpcMessage{
		JSONRPC: "2.0",
		ID:      id,
	}
	respBytes, err := json.Marshal(result)
	if err != nil {
		slog.Error("mcp: marshal result", "error", err)
		return
	}
	resp.Result = respBytes
	writeMessage(w, resp)
}

func sendError(w io.Writer, id *int, code int, message, data string) {
	resp := jsonrpcMessage{
		JSONRPC: "2.0",
		ID:      id,
		Error: &jsonrpcError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	writeMessage(w, resp)
}

func writeMessage(w io.Writer, msg jsonrpcMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		slog.Error("mcp: marshal response", "error", err)
		return
	}
	fmt.Fprintf(w, "Content-Length: %d\r\n\r\n%s", len(data), data)
}

// messageReader reads MCP stdio messages with Content-Length framing.
type messageReader struct {
	reader *bufio.Reader
}

func newMessageReader(r io.Reader) *messageReader {
	return &messageReader{reader: bufio.NewReader(r)}
}

func (mr *messageReader) readMessage() ([]byte, error) {
	contentLength := 0
	for {
		line, err := mr.reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if strings.HasPrefix(line, "Content-Length:") {
			if _, err := fmt.Sscanf(line, "Content-Length: %d", &contentLength); err != nil {
				slog.Warn("mcp: bad Content-Length header", "line", line)
			}
		}
	}
	if contentLength == 0 {
		return nil, fmt.Errorf("mcp: empty content length")
	}
	body := make([]byte, contentLength)
	if _, err := io.ReadFull(mr.reader, body); err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	return body, nil
}
