package mcp

import (
	"context"

	"github.com/wzhongyou/baize/core/tool"
)

// BaizeMCPServer exposes Baize's tools as an MCP server, allowing external
// MCP clients to discover and invoke Baize's capabilities
// through the Model Context Protocol.
//
// Full implementation will be completed in a future phase. The current stub
// defines the structure and will be connected to the MCP server protocol
// when the server subsystem is built out.
type BaizeMCPServer struct {
	name    string
	version string
	tools   *tool.ToolRegistry
}

// NewBaizeMCPServer creates a new BaizeMCPServer with the given tools.
func NewBaizeMCPServer(tools *tool.ToolRegistry) *BaizeMCPServer {
	return &BaizeMCPServer{
		name:    "baize",
		version: "1.0.0",
		tools:   tools,
	}
}

// Serve starts the MCP server. This is a stub — full stdio-based MCP server
// implementation will be completed in a future phase.
func (s *BaizeMCPServer) Serve(ctx context.Context) error {
	// TODO: Implement full MCP server over stdio using the MCP protocol.
	// The server will:
	//   1. Read JSON-RPC messages from stdin
	//   2. Handle Initialize, ListTools, CallTool requests
	//   3. Write JSON-RPC responses to stdout
	<-ctx.Done()
	return ctx.Err()
}

// Name returns the server identifier.
func (s *BaizeMCPServer) Name() string { return s.name }

// Version returns the server version string.
func (s *BaizeMCPServer) Version() string { return s.version }
