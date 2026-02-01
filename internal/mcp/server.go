package mcp

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/w31r4/codex-mcp-go/internal/codex"
	cerrors "github.com/w31r4/codex-mcp-go/internal/errors"
	"github.com/w31r4/codex-mcp-go/internal/logging"
	"github.com/w31r4/codex-mcp-go/internal/metrics"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	serverStartTime = time.Now()
	globalMetrics   = metrics.New()
)

// CodexInput represents the input parameters for the codex tool
type CodexInput struct {
	PROMPT            string   `json:"PROMPT" jsonschema:"Instruction for the task to send to codex."`
	Cd                string   `json:"cd" jsonschema:"Set the workspace root for codex before executing the task."`
	Sandbox           string   `json:"sandbox,omitempty" jsonschema:"enum=read-only,enum=workspace-write,enum=danger-full-access,description=Sandbox policy for model-generated commands. Valid values: read-only (default) workspace-write danger-full-access."`
	SessionID         string   `json:"SESSION_ID,omitempty" jsonschema:"Resume the specified session of the codex. Defaults to None, start a new session."`
	SkipGitRepoCheck  *bool    `json:"skip_git_repo_check,omitempty" jsonschema:"Allow codex running outside a Git repository (useful for one-off directories)."`
	ReturnAllMessages bool     `json:"return_all_messages,omitempty" jsonschema:"Return all messages (e.g. reasoning, tool calls, etc.) from the codex session. Set to False by default, only the agent's final reply message is returned."`
	Image             []string `json:"image,omitempty" jsonschema:"Attach one or more image files to the initial prompt. Separate multiple paths with commas or repeat the flag."`
	Model             string   `json:"model,omitempty" jsonschema:"The model to use for the codex session. This parameter is strictly prohibited unless explicitly specified by the user."`
	Yolo              *bool    `json:"yolo,omitempty" jsonschema:"Run every command without approvals or sandboxing. Defaults to false to avoid unsafe execution."`
	Profile           string   `json:"profile,omitempty" jsonschema:"Configuration profile name to load from '~/.codex/config.toml'. This parameter is strictly prohibited unless explicitly specified by the user."`
	TimeoutSeconds    *int     `json:"timeout_seconds,omitempty" jsonschema:"Total timeout (seconds) for the codex invocation. Defaults to 1800 (30 minutes) if not set; capped at 1800 (30 minutes)."`
	NoOutputSeconds   *int     `json:"no_output_seconds,omitempty" jsonschema:"No-output watchdog (seconds). Kill the run if no output for this duration. Defaults to 0 (disabled) if not set."`
}

// CodexOutput represents the output from the codex tool
type CodexOutput struct {
	Success       bool                     `json:"success"`
	SessionID     string                   `json:"SESSION_ID"`
	AgentMessages string                   `json:"agent_messages"`
	AllMessages   []map[string]interface{} `json:"all_messages,omitempty"`
}

type StatsInput struct{}

type StatsOutput struct {
	Uptime  string           `json:"uptime"`
	Metrics metrics.Snapshot `json:"metrics"`
}

// buildInputSchema creates an explicit JSON Schema for CodexInput.
// This ensures all fields have explicit "type" fields, which is required
// by providers like Gemini/Vertex AI that strictly validate function declarations.
func buildInputSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"PROMPT": {
				Type:        "string",
				Description: "Instruction for the task to send to codex.",
			},
			"cd": {
				Type:        "string",
				Description: "Set the workspace root for codex before executing the task.",
			},
			"sandbox": {
				Type:        "string",
				Description: "Sandbox policy for model-generated commands. Valid values: read-only (default), workspace-write, danger-full-access.",
				Enum:        []any{"read-only", "workspace-write", "danger-full-access"},
			},
			"SESSION_ID": {
				Type:        "string",
				Description: "Resume the specified session of the codex. Defaults to None, start a new session.",
			},
			"skip_git_repo_check": {
				Type:        "boolean",
				Description: "Allow codex running outside a Git repository (useful for one-off directories).",
			},
			"return_all_messages": {
				Type:        "boolean",
				Description: "Return all messages (e.g. reasoning, tool calls, etc.) from the codex session. Set to False by default, only the agent's final reply message is returned.",
			},
			"image": {
				Type:        "array",
				Description: "Attach one or more image files to the initial prompt. Separate multiple paths with commas or repeat the flag.",
				Items:       &jsonschema.Schema{Type: "string"},
			},
			"model": {
				Type:        "string",
				Description: "The model to use for the codex session. This parameter is strictly prohibited unless explicitly specified by the user.",
			},
			"yolo": {
				Type:        "boolean",
				Description: "Run every command without approvals or sandboxing. Defaults to false to avoid unsafe execution.",
			},
			"profile": {
				Type:        "string",
				Description: "Configuration profile name to load from '~/.codex/config.toml'. This parameter is strictly prohibited unless explicitly specified by the user.",
			},
			"timeout_seconds": {
				Type:        "number",
				Description: "Total timeout (seconds) for the codex invocation. Defaults to 1800 (30 minutes) if not set; capped at 1800 (30 minutes).",
			},
			"no_output_seconds": {
				Type:        "number",
				Description: "No-output watchdog (seconds). Kill the run if no output for this duration. Defaults to 0 (disabled) if not set.",
			},
		},
		Required: []string{"PROMPT", "cd"},
	}
}

func buildStatsInputSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type:       "object",
		Properties: map[string]*jsonschema.Schema{},
	}
}

// NewServer creates and configures a new MCP server with the codex tool
func NewServer() *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{
		Name:    "Codex MCP Server-from guda.studio",
		Version: "0.0.9",
	}, nil)

	// Define the codex tool with explicit InputSchema
	// This ensures compatibility with strict schema validators like Gemini/Vertex AI
	tool := &mcp.Tool{
		Name: "codex",
		Description: `Executes a non-interactive Codex session via CLI to perform AI-assisted coding tasks in a secure workspace.
This tool wraps the 'codex exec' command, enabling model-driven code generation, debugging, or automation based on natural language prompts.
It supports resuming ongoing sessions for continuity and enforces sandbox policies to prevent unsafe operations. Ideal for integrating Codex into MCP servers for agentic workflows, such as code reviews or repo modifications.

Key Features:
- Prompt-Driven Execution: Send task instructions to Codex for step-by-step code handling.
- Workspace Isolation: Operate within a specified directory, with optional Git repo skipping.
- Security Controls: Three sandbox levels (read-only, workspace-write, danger-full-access) balance functionality and safety.
- Session Persistence: Resume prior conversations via SESSION_ID for iterative tasks.

Edge Cases & Best Practices:
- Ensure 'cd' exists and is accessible; tool fails silently on invalid paths.
- Defaults to "read-only" sandbox. Valid sandbox values: read-only, workspace-write, danger-full-access.
- Disables "yolo" (auto-confirmation) by default; enable write/yolo explicitly if your workflow requires it.
- If needed, set 'return_all_messages' to True to parse "all_messages" for detailed tracing (e.g., reasoning, tool calls, etc.).`,
		InputSchema: buildInputSchema(),
		Meta: mcp.Meta{
			"version": "0.0.9",
			"author":  "guda.studio",
		},
	}

	// Add the tool handler
	mcp.AddTool(s, tool, handleCodexTool)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "stats",
		Title:       "Server Stats",
		Description: "Returns server uptime and aggregate request metrics.",
		InputSchema: buildStatsInputSchema(),
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}, handleStats)

	return s
}

// handleCodexTool processes the codex tool call
func handleCodexTool(ctx context.Context, req *mcp.CallToolRequest, input CodexInput) (callResult *mcp.CallToolResult, out CodexOutput, err error) {
	ctx, rc := logging.NewRequestContext(ctx, "codex")
	logging.LogRequest(ctx, map[string]any{
		"cd":                  input.Cd,
		"sandbox":             input.Sandbox,
		"session_id":          strings.TrimSpace(input.SessionID),
		"prompt_chars":        len(input.PROMPT),
		"image_count":         len(input.Image),
		"return_all_messages": input.ReturnAllMessages,
	})
	defer func() {
		success := err == nil && out.Success
		globalMetrics.RecordRequest("codex", success, time.Since(rc.StartTime))
		if err != nil {
			var cerr *cerrors.Error
			if errors.As(err, &cerr) {
				globalMetrics.RecordError(cerr.Code.Name())
			}
		}
		logging.LogResponse(ctx, map[string]any{
			"success":    success,
			"session_id": out.SessionID,
		}, err)
	}()

	// Validate required parameters
	if input.PROMPT == "" {
		return nil, CodexOutput{}, cerrors.ErrInvalidParams("PROMPT is required and must be a non-empty string")
	}

	if input.Cd == "" {
		return nil, CodexOutput{}, cerrors.ErrInvalidParams("cd is required and must be a non-empty string")
	}

	// Validate working directory exists
	info, err := os.Stat(input.Cd)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, CodexOutput{}, cerrors.ErrWorkdirNotFound(input.Cd)
		}
		return nil, CodexOutput{}, cerrors.Wrap(cerrors.InternalError, "failed to stat working directory", err).
			WithData("path", input.Cd)
	}
	if !info.IsDir() {
		return nil, CodexOutput{}, cerrors.ErrWorkdirNotDirectory(input.Cd)
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

	yolo := false
	if input.Yolo != nil {
		yolo = *input.Yolo
	}

	if input.Model != "" {
		return nil, CodexOutput{}, cerrors.ErrParameterProhibited("model", "model selection is disabled by server policy")
	}

	if input.Profile != "" {
		return nil, CodexOutput{}, cerrors.ErrParameterProhibited("profile", "profile selection is disabled by server policy")
	}

	var timeout time.Duration
	if input.TimeoutSeconds != nil && *input.TimeoutSeconds > 0 {
		timeout = time.Duration(*input.TimeoutSeconds) * time.Second
	}
	if timeout > 30*time.Minute {
		timeout = 30 * time.Minute
	}

	var noOutput time.Duration
	if input.NoOutputSeconds != nil && *input.NoOutputSeconds > 0 {
		noOutput = time.Duration(*input.NoOutputSeconds) * time.Second
	} // nil or <=0 keeps default (disabled)

	// Validate image files exist
	for _, imgPath := range input.Image {
		if _, err := os.Stat(imgPath); err != nil {
			if os.IsNotExist(err) {
				return nil, CodexOutput{}, cerrors.ErrImageNotFound(imgPath)
			}
			return nil, CodexOutput{}, cerrors.Wrap(cerrors.InternalError, "failed to stat image file", err).
				WithData("path", imgPath)
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
		Yolo:              yolo,
		Profile:           input.Profile,
		Timeout:           timeout,
		NoOutputTimeout:   noOutput,
	}

	// Execute codex
	codexResult, runErr := codex.Run(ctx, opts)
	if runErr != nil {
		var cerr *cerrors.Error
		if errors.As(runErr, &cerr) {
			return nil, CodexOutput{}, cerr
		}
		return nil, CodexOutput{}, cerrors.ErrCodexExecutionFailed("failed to execute codex", runErr)
	}

	// Check if execution was successful
	if !codexResult.Success {
		if codexResult.Error == "" {
			return nil, CodexOutput{}, cerrors.New(cerrors.CodexExecutionFailed, "codex execution failed")
		}
		return nil, CodexOutput{}, cerrors.New(cerrors.CodexExecutionFailed, codexResult.Error)
	}

	// Prepare the response
	out = CodexOutput{
		Success:       true,
		SessionID:     codexResult.SessionID,
		AgentMessages: codexResult.AgentMessages,
	}

	if input.ReturnAllMessages {
		out.AllMessages = codexResult.AllMessages
	}

	return nil, out, nil
}

func handleStats(ctx context.Context, req *mcp.CallToolRequest, input StatsInput) (result *mcp.CallToolResult, output StatsOutput, err error) {
	ctx, rc := logging.NewRequestContext(ctx, "stats")
	logging.LogRequest(ctx, map[string]any{})
	defer func() {
		success := err == nil
		globalMetrics.RecordRequest("stats", success, time.Since(rc.StartTime))
		if err != nil {
			var cerr *cerrors.Error
			if errors.As(err, &cerr) {
				globalMetrics.RecordError(cerr.Code.Name())
			}
		}
		logging.LogResponse(ctx, map[string]any{"success": success}, err)
	}()

	output = StatsOutput{
		Uptime:  time.Since(serverStartTime).String(),
		Metrics: globalMetrics.Snapshot(),
	}
	return nil, output, nil
}

// Run starts the MCP server over stdio transport.
func Run(ctx context.Context) error {
	server := NewServer()
	return server.Run(ctx, &mcp.StdioTransport{})
}
