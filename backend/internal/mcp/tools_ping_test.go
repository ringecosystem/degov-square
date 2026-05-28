package mcp

import (
	"context"
	"testing"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestPingToolReturnsServiceStatus(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clientTransport, serverTransport := sdkmcp.NewInMemoryTransports()
	server := NewServer(Config{Name: "degov-square", Version: "test-version"})
	go func() {
		_ = server.Run(ctx, serverTransport)
	}()

	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "test-client"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer session.Close()

	result, err := session.CallTool(ctx, &sdkmcp.CallToolParams{Name: "ping"})
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}

	content, ok := result.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("StructuredContent type = %T, want map[string]any", result.StructuredContent)
	}
	if got, want := content["status"], "ok"; got != want {
		t.Fatalf("status = %v, want %v", got, want)
	}
	if got, want := content["service"], "degov-square"; got != want {
		t.Fatalf("service = %v, want %v", got, want)
	}
	if got, want := content["version"], "test-version"; got != want {
		t.Fatalf("version = %v, want %v", got, want)
	}
}
