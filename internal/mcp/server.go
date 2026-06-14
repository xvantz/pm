// Package mcp implements a Model Context Protocol server for PM.
//
// It exposes PM's project memory functionality as MCP tools over stdio
// transport, enabling LLM agents to read, create, and update project data.
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
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
func (s *Server) Run() error {
	r := newMessageReader(os.Stdin)
	enc := json.NewEncoder(os.Stdout)

	initialized := false

	for {
		body, err := r.readMessage()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("read message: %w", err)
		}

		var req json.RawMessage
		if err := json.Unmarshal(body, &req); err != nil {
			// Not JSON — skip
			continue
		}

		s.handleMessage(req, enc, &initialized)
	}
}

// jsonrpcMessage is the outer JSON-RPC 2.0 envelope.
type jsonrpcMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int            `json:"id,omitempty"`
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

func (s *Server) handleMessage(body json.RawMessage, enc *json.Encoder, initialized *bool) {
	var msg jsonrpcMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		sendError(enc, nil, -32700, "Parse error", "")
		return
	}

	// Notifications have no ID
	isNotification := msg.ID == nil

	switch msg.Method {
	case "initialize":
		s.handleInitialize(enc, msg.ID)

	case "notifications/initialized":
		*initialized = true
		// Notifications don't have responses

	case "tools/list":
		if !*initialized {
			sendError(enc, msg.ID, -32000, "Not initialized", "")
			return
		}
		s.handleToolsList(enc, msg.ID)

	case "tools/call":
		if !*initialized {
			sendError(enc, msg.ID, -32000, "Not initialized", "")
			return
		}
		s.handleToolCall(enc, msg.ID, msg.Params)

	default:
		if !isNotification {
			sendError(enc, msg.ID, -32601, fmt.Sprintf("Method not found: %s", msg.Method), "")
		}
	}
}

func (s *Server) handleInitialize(enc *json.Encoder, id *int) {
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
	sendResult(enc, id, result)
}

func (s *Server) handleToolsList(enc *json.Encoder, id *int) {
	tools := make([]Tool, len(s.tools))
	for i, t := range s.tools {
		// Don't expose the handler in the JSON response
		tools[i] = Tool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.InputSchema,
		}
	}
	sendResult(enc, id, map[string]any{"tools": tools})
}

func (s *Server) handleToolCall(enc *json.Encoder, id *int, params json.RawMessage) {
	var call struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(params, &call); err != nil {
		sendError(enc, id, -32602, "Invalid params", err.Error())
		return
	}

	for _, t := range s.tools {
		if t.Name == call.Name {
			text, err := t.Handler(context.Background(), call.Arguments)
			if err != nil {
				sendError(enc, id, -32603, err.Error(), "")
				return
			}
			sendResult(enc, id, map[string]any{
				"content": []map[string]string{
					{"type": "text", "text": text},
				},
			})
			return
		}
	}

	sendError(enc, id, -32602, fmt.Sprintf("Unknown tool: %s", call.Name), "")
}

func sendResult(enc *json.Encoder, id *int, result any) {
	resp := jsonrpcMessage{
		JSONRPC: "2.0",
		ID:      id,
	}
	respBytes, _ := json.Marshal(result)
	resp.Result = respBytes
	writeMessage(enc, resp)
}

func sendError(enc *json.Encoder, id *int, code int, message, data string) {
	resp := jsonrpcMessage{
		JSONRPC: "2.0",
		ID:      id,
		Error: &jsonrpcError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	writeMessage(enc, resp)
}

func writeMessage(enc *json.Encoder, msg jsonrpcMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		slog.Error("mcp: marshal response", "error", err)
		return
	}
	// MCP stdio transport: Content-Length header + blank line + JSON body
	fmt.Fprintf(os.Stdout, "Content-Length: %d\r\n\r\n%s", len(data), data)
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
		// Ignore other headers (Content-Type etc.)
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
