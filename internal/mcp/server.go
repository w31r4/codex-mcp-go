package mcp

import (
	"context"
	"fmt"
	"os"
	"strings"

	"codex4kilomcp/internal/codex"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CodexInput represents the input parameters for the codex tool
type CodexInput struct {
	PROMPT            string   `json:"PROMPT" jsonschema:"Instruction for the task to send to codex."`
	Cd                string   `json:"cd" jsonschema:"Set the workspace root for codex before executing the task."`
	Sandbox           string   `json:"sandbox,omitempty" jsonschema:"Sandbox policy for model-generated commands. Defaults to 'read-only'."`
	SessionID         string   `json:"SESSION_ID,omitempty" jsonschema:"Resume the specified session of the codex. Defaults to None, start a new session."`
	SkipGitRepoCheck  *bool    `json:"skip_git_repo_check,omitempty" jsonschema:"Allow codex running outside a Git repository (useful for one-off directories)."`
	ReturnAllMessages bool     `json:"return_all_messages,omitempty" jsonschema:"Return all messages (e.g. reasoning, tool calls, etc.) from the codex session. Set to False by default, only the agent's final reply message is returned."`
	Image             []string `json:"image,omitempty" jsonschema:"Attach one or more image files to the initial prompt. Separate multiple paths with commas or repeat the flag."`
	Model             string   `json:"model,omitempty" jsonschema:"The model to use for the codex session. This parameter is strictly prohibited unless explicitly specified by the user."`
	Yolo              bool     `json:"yolo,omitempty" jsonschema:"Run every command without approvals or sandboxing. Only use when 'sandbox' couldn't be applied."`
	Profile           string   `json:"profile,omitempty" jsonschema:"Configuration profile name to load from '~/.codex/config.toml'. This parameter is strictly prohibited unless explicitly specified by the user."`
}

// CodexOutput represents the output from the codex tool
type CodexOutput struct {
	Success       bool                     `json:"success"`
	SessionID     string                   `json:"SESSION_ID"`
	AgentMessages string                   `json:"agent_messages"`
	AllMessages   []map[string]interface{} `json:"all_messages,omitempty"`
}

// NewServer creates and configures a new MCP server with the codex tool
func NewServer() *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{
		Name:    "Codex MCP Server-from guda.studio",
		Version: "0.0.0",
	}, nil)

	// Define the codex tool
	tool := &mcp.Tool{
		Name: "codex",
		Description: `Executes a non-interactive Codex session via CLI to perform AI-assisted coding tasks in a secure workspace.
This tool wraps the 'codex exec' command, enabling model-driven code generation, debugging, or automation based on natural language prompts.
It supports resuming ongoing sessions for continuity and enforces sandbox policies to prevent unsafe operations. Ideal for integrating Codex into MCP servers for agentic workflows, such as code reviews or repo modifications.

Key Features:
- Prompt-Driven Execution: Send task instructions to Codex for step-by-step code handling.
- Workspace Isolation: Operate within a specified directory, with optional Git repo skipping.
- Security Controls: Three sandbox levels balance functionality and safety.
- Session Persistence: Resume prior conversations via SESSION_ID for iterative tasks.

Edge Cases & Best Practices:
- Ensure 'cd' exists and is accessible; tool fails silently on invalid paths.
- For most repos, prefer "read-only" to avoid accidental changes.
- If needed, set 'return_all_messages' to True to parse "all_messages" for detailed tracing (e.g., reasoning, tool calls, etc.).`,
		Meta: mcp.Meta{
			"version": "0.0.0",
			"author":  "guda.studio",
		},
	}

	// Add the tool handler
	mcp.AddTool(s, tool, handleCodexTool)

	return s
}

// handleCodexTool processes the codex tool call
func handleCodexTool(ctx context.Context, req *mcp.CallToolRequest, input CodexInput) (*mcp.CallToolResult, CodexOutput, error) {
	// Validate required parameters
	if input.PROMPT == "" {
		return nil, CodexOutput{}, fmt.Errorf("PROMPT is required and must be a string")
	}

	if input.Cd == "" {
		return nil, CodexOutput{}, fmt.Errorf("cd is required and must be a string")
	}

	// Validate working directory exists
	if _, err := os.Stat(input.Cd); err != nil {
		return nil, CodexOutput{}, fmt.Errorf("working directory does not exist: %s", input.Cd)
	}

	// Set defaults
	if input.Sandbox == "" {
		input.Sandbox = "read-only"
	}
	input.SessionID = strings.TrimSpace(input.SessionID)
	skipGitRepoCheck := true
	if input.SkipGitRepoCheck != nil {
		skipGitRepoCheck = *input.SkipGitRepoCheck
	}

	// Validate image files exist
	for _, imgPath := range input.Image {
		if _, err := os.Stat(imgPath); err != nil {
			return nil, CodexOutput{}, fmt.Errorf("image file does not exist: %s", imgPath)
		}
	}

	// Create options for codex client
	opts := codex.Options{
		Prompt:            input.PROMPT,
		WorkingDir:        input.Cd,
		Sandbox:           input.Sandbox,
		SessionID:         input.SessionID,
		SkipGitRepoCheck:  skipGitRepoCheck,
		ReturnAllMessages: input.ReturnAllMessages,
		ImagePaths:        input.Image,
		Model:             input.Model,
		Yolo:              input.Yolo,
		Profile:           input.Profile,
	}

	// Execute codex
	result, err := codex.Run(opts)
	if err != nil {
		return nil, CodexOutput{}, fmt.Errorf("failed to execute codex: %v", err)
	}

	// Check if execution was successful
	if !result.Success {
		return nil, CodexOutput{}, fmt.Errorf("%s", result.Error)
	}

	// Prepare the response
	output := CodexOutput{
		Success:       true,
		SessionID:     result.SessionID,
		AgentMessages: result.AgentMessages,
	}

	if input.ReturnAllMessages {
		output.AllMessages = result.AllMessages
	}

	return nil, output, nil
}

// Run starts the MCP server over stdio transport.
func Run(ctx context.Context) error {
	server := NewServer()
	return server.Run(ctx, &mcp.StdioTransport{})
}
