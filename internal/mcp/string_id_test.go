package mcp

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

// MCP requires the server to accept a string or integer request id. With the
// field typed *int, a client sending a string id failed to decode and the whole
// session died on a parse error rather than on something meaningful.
func TestStringID_RoundTripsUnchanged(t *testing.T) {
	s := NewServer("test", "1.0")
	var buf bytes.Buffer
	state := stateInitialized

	s.handleMessage(context.Background(),
		testMsg(`"call-abc-123"`, "tools/list", ``),
		&buf, &state)

	resp := readMCPResponse(t, &buf)
	if got := string(resp.ID); got != `"call-abc-123"` {
		t.Errorf("id = %s, want the same string back", got)
	}
	if resp.Error != nil {
		t.Errorf("a string id must not be an error: %v", resp.Error)
	}
	if !strings.Contains(string(resp.Result), "tools") {
		t.Errorf("tools/list result missing: %s", resp.Result)
	}
}

// Integer ids keep working — this is what Hermes actually sends, and a change
// here that broke it would be invisible until the next session.
func TestStringID_IntegerIDsStillWork(t *testing.T) {
	for _, id := range []string{"1", "42", "0"} {
		s := NewServer("test", "1.0")
		var buf bytes.Buffer
		state := stateInitialized

		s.handleMessage(context.Background(),
			testMsg(id, "tools/list", ``),
			&buf, &state)

		resp := readMCPResponse(t, &buf)
		if got := string(resp.ID); got != id {
			t.Errorf("id = %s, want %s", got, id)
		}
		if resp.Error != nil {
			t.Errorf("id %s must not error: %v", id, resp.Error)
		}
	}
}

// A notification has no id at all, and must stay silent. An absent id is a
// nil RawMessage — the distinction the old *int check relied on.
func TestStringID_NotificationsStaySilent(t *testing.T) {
	s := NewServer("test", "1.0")
	var buf bytes.Buffer
	state := stateInitialized

	// No id, unknown method: the old code treated this as a notification and
	// wrote nothing.
	s.handleMessage(context.Background(),
		jsonrpcMessage{JSONRPC: "2.0", Method: "nonsense/method"},
		&buf, &state)

	if buf.Len() != 0 {
		t.Errorf("a notification must not produce output, got: %s", buf.String())
	}
}

// An unknown method WITH an id must still get an error response carrying that
// id, string or not.
func TestStringID_UnknownMethodErrorsWithSameID(t *testing.T) {
	s := NewServer("test", "1.0")
	var buf bytes.Buffer
	state := stateInitialized

	s.handleMessage(context.Background(),
		testMsg(`"str-id-7"`, "does/not/exist", ``),
		&buf, &state)

	resp := readMCPResponse(t, &buf)
	if got := string(resp.ID); got != `"str-id-7"` {
		t.Errorf("error response id = %s, want the request's string id", got)
	}
	if resp.Error == nil {
		t.Fatal("expected an error for an unknown method")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("code = %d, want -32601", resp.Error.Code)
	}
}
