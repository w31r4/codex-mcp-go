package mcp

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/w31r4/codex-mcp-go/internal/codex"
	"github.com/w31r4/codex-mcp-go/internal/config"
	cerrors "github.com/w31r4/codex-mcp-go/internal/errors"
	"github.com/w31r4/codex-mcp-go/internal/logging"
	"github.com/w31r4/codex-mcp-go/internal/metrics"
	"github.com/w31r4/codex-mcp-go/internal/session"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	serverStartTime = time.Now()
	globalMetrics   = metrics.New()
	globalConfig    = config.Default()
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
	Model             string   `json:"model,omitempty" jsonschema:"The model to use for the codex session. This parameter is restricted by server allowlist (disabled by default)."`
	Yolo              *bool    `json:"yolo,omitempty" jsonschema:"Run every command without approvals or sandboxing. Defaults to false to avoid unsafe execution."`
	Profile           string   `json:"profile,omitempty" jsonschema:"Configuration profile name to load from '~/.codex/config.toml'. This parameter is restricted by server allowlist (disabled by default)."`
	TimeoutSeconds    *int     `json:"timeout_seconds,omitempty" jsonschema:"Total timeout (seconds) for the codex invocation. Defaults to 1800 (30 minutes) if not set; capped at 1800 (30 minutes)."`
	NoOutputSeconds   *int     `json:"no_output_seconds,omitempty" jsonschema:"No-output watchdog (seconds). Kill the run if no output for this duration. Defaults to 0 (disabled) if not set."`
}

// CodexOutput represents the output from the codex tool
type CodexOutput struct {
	Success         bool                     `json:"success"`
	SessionID       string                   `json:"SESSION_ID"`
	AgentMessages   string                   `json:"agent_messages"`
	AllMessages     []map[string]interface{} `json:"all_messages,omitempty"`
	ExecutionTimeMs int64                    `json:"execution_time_ms"`
	ToolCallCount   int                      `json:"tool_call_count"`
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
				Description: "The model to use for the codex session. This parameter is restricted by server allowlist (disabled by default).",
			},
			"yolo": {
				Type:        "boolean",
				Description: "Run every command without approvals or sandboxing. Defaults to false to avoid unsafe execution.",
			},
			"profile": {
				Type:        "string",
				Description: "Configuration profile name to load from '~/.codex/config.toml'. This parameter is restricted by server allowlist (disabled by default).",
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

func buildOutputSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"success": {
				Type:        "boolean",
				Description: "Whether the Codex invocation succeeded.",
			},
			"SESSION_ID": {
				Type:        "string",
				Description: "Codex session/thread identifier (thread_id).",
			},
			"agent_messages": {
				Type:        "string",
				Description: "The agent's final reply text (may contain multiple lines).",
			},
			"all_messages": {
				Type:        "array",
				Description: "Raw Codex CLI JSONL lines. Present only when return_all_messages=true.",
				Items:       &jsonschema.Schema{Type: "object"},
			},
			"execution_time_ms": {
				Type:        "number",
				Description: "Execution time for the Codex CLI invocation, in milliseconds.",
			},
			"tool_call_count": {
				Type:        "number",
				Description: "Best-effort count of tool calls observed in Codex JSONL output.",
			},
		},
		Required: []string{"success", "SESSION_ID", "agent_messages"},
	}
}

func buildStatsInputSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type:       "object",
		Properties: map[string]*jsonschema.Schema{},
	}
}

// NewServer creates and configures a new MCP server with the codex tool
func NewServer(cfg *config.Config) *mcp.Server {
	if cfg == nil {
		cfg = globalConfig
	}
	globalConfig = cfg
	globalSessions = session.NewManager(session.DefaultOptions())

	s := mcp.NewServer(&mcp.Implementation{
		Name:    cfg.Server.Name,
		Version: cfg.Server.Version,
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
		OutputSchema: buildOutputSchema(),
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

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_sessions",
		Title:       "List Sessions",
		Description: "Lists running and recent Codex sessions tracked by the server.",
		InputSchema: buildListSessionsInputSchema(),
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}, handleListSessions)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_session",
		Title:       "Get Session",
		Description: "Returns session metadata and state for the given SESSION_ID.",
		InputSchema: buildGetSessionInputSchema(),
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}, handleGetSession)

	destructive := true
	openWorld := false
	mcp.AddTool(s, &mcp.Tool{
		Name:        "cancel_session",
		Title:       "Cancel Session",
		Description: "Cancels a running session identified by SESSION_ID.",
		InputSchema: buildCancelSessionInputSchema(),
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    false,
			DestructiveHint: &destructive,
			IdempotentHint:  false,
			OpenWorldHint:   &openWorld,
		},
	}, handleCancelSession)

	return s
}

// handleCodexTool processes the codex tool call
func handleCodexTool(ctx context.Context, req *mcp.CallToolRequest, input CodexInput) (callResult *mcp.CallToolResult, out CodexOutput, err error) {
	cfg := globalConfig

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

	if cfg != nil && !cfg.Security.IsWorkDirAllowed(input.Cd) {
		return nil, CodexOutput{}, cerrors.New(cerrors.InvalidParams, "working directory is not allowed").
			WithData("path", input.Cd)
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
		if cfg != nil {
			input.Sandbox = cfg.Security.DefaultSandbox
		} else {
			input.Sandbox = "read-only"
		}
	}
	input.SessionID = strings.TrimSpace(input.SessionID)

	if cfg != nil && !cfg.Security.IsSandboxAllowed(input.Sandbox) {
		return nil, CodexOutput{}, cerrors.ErrInvalidSandboxMode(input.Sandbox, cfg.Security.AllowedSandboxModes)
	}

	skipGitRepoCheck := true
	if input.SkipGitRepoCheck != nil {
		skipGitRepoCheck = *input.SkipGitRepoCheck
	}

	yolo := false
	if input.Yolo != nil {
		yolo = *input.Yolo
	}

	if cfg != nil && cfg.Security.DisableYolo && yolo {
		return nil, CodexOutput{}, cerrors.ErrParameterProhibited("yolo", "yolo is disabled by server policy")
	}

	if input.Model != "" {
		if cfg == nil || !cfg.Security.IsModelAllowed(input.Model) {
			return nil, CodexOutput{}, cerrors.ErrParameterProhibited("model", "model is not allowlisted by server configuration")
		}
	}

	if input.Profile != "" {
		if cfg == nil || !cfg.Security.IsProfileAllowed(input.Profile) {
			return nil, CodexOutput{}, cerrors.ErrParameterProhibited("profile", "profile is not allowlisted by server configuration")
		}
	}

	var timeout time.Duration
	timeoutSeconds := 0
	if cfg != nil {
		timeoutSeconds = cfg.Codex.DefaultTimeoutSeconds
	}
	if input.TimeoutSeconds != nil && *input.TimeoutSeconds > 0 {
		timeoutSeconds = *input.TimeoutSeconds
	}
	if cfg != nil && timeoutSeconds > cfg.Codex.MaxTimeoutSeconds {
		timeoutSeconds = cfg.Codex.MaxTimeoutSeconds
	}
	if timeoutSeconds > 0 {
		timeout = time.Duration(timeoutSeconds) * time.Second
	}

	var noOutput time.Duration
	noOutputSeconds := 0
	if cfg != nil {
		noOutputSeconds = cfg.Codex.DefaultNoOutputTimeoutSeconds
	}
	if input.NoOutputSeconds != nil && *input.NoOutputSeconds > 0 {
		noOutputSeconds = *input.NoOutputSeconds
	}
	if noOutputSeconds > 0 {
		noOutput = time.Duration(noOutputSeconds) * time.Second
	}

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
		ExecutablePath:    cfg.Codex.ExecutablePath,
		MaxBufferedLines:  cfg.Codex.MaxBufferedLines,
	}

	// Track this execution as a session.
	trackingID := input.SessionID
	if trackingID == "" {
		trackingID = session.NewTemporaryID()
	}
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	if _, startErr := globalSessions.Start(trackingID, input.Cd, input.Sandbox, cancel); startErr != nil {
		return nil, CodexOutput{}, startErr
	}

	// Execute codex
	runStart := time.Now()
	codexResult, runErr := codex.Run(runCtx, opts)
	runDuration := time.Since(runStart)
	if runErr != nil {
		// Best-effort: if this was a new session, update the temporary tracking ID to the real thread_id when known.
		if input.SessionID == "" && codexResult != nil && strings.TrimSpace(codexResult.SessionID) != "" && codexResult.SessionID != trackingID {
			if ok, updateErr := globalSessions.UpdateID(trackingID, codexResult.SessionID); updateErr == nil && ok {
				trackingID = codexResult.SessionID
			}
		}
		if errors.Is(runCtx.Err(), context.Canceled) {
			globalSessions.MarkCancelled(trackingID, "cancelled")
		} else {
			globalSessions.MarkFailed(trackingID, runErr)
		}
		var cerr *cerrors.Error
		if errors.As(runErr, &cerr) {
			return nil, CodexOutput{}, cerr
		}
		return nil, CodexOutput{}, cerrors.ErrCodexExecutionFailed("failed to execute codex", runErr)
	}

	// Best-effort: if this was a new session, update the temporary tracking ID to the real thread_id when known.
	if input.SessionID == "" && codexResult != nil && strings.TrimSpace(codexResult.SessionID) != "" && codexResult.SessionID != trackingID {
		if ok, updateErr := globalSessions.UpdateID(trackingID, codexResult.SessionID); updateErr == nil && ok {
			trackingID = codexResult.SessionID
		}
	}

	// Check if execution was successful
	if !codexResult.Success {
		msg := strings.TrimSpace(codexResult.Error)
		if msg == "" {
			msg = "codex execution failed"
		}
		errOut := cerrors.New(cerrors.CodexExecutionFailed, msg)
		globalSessions.MarkFailed(trackingID, errOut)
		return nil, CodexOutput{}, errOut
	}

	globalSessions.MarkCompleted(trackingID, runDuration.Milliseconds(), codexResult.ToolCallCount)

	// Prepare the response
	out = CodexOutput{
		Success:         true,
		SessionID:       codexResult.SessionID,
		AgentMessages:   codexResult.AgentMessages,
		ExecutionTimeMs: runDuration.Milliseconds(),
		ToolCallCount:   codexResult.ToolCallCount,
	}

	if input.ReturnAllMessages {
		out.AllMessages = codexResult.AllMessages
	}

	callResult = &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: codexResult.AgentMessages},
		},
	}
	return callResult, out, nil
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
func Run(ctx context.Context, cfg *config.Config) error {
	server := NewServer(cfg)
	globalSessions.StartCleanup(ctx, time.Minute)
	return server.Run(ctx, &mcp.StdioTransport{})
}
