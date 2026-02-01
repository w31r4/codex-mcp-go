package codex

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os/exec"
	"runtime"
	"strings"
	"time"

	cerrors "github.com/w31r4/codex-mcp-go/internal/errors"
	"github.com/w31r4/codex-mcp-go/internal/progress"
)

const (
	defaultCommandTimeout  = 30 * time.Minute
	maxCommandTimeout      = 30 * time.Minute
	defaultNoOutputTimeout = 0 // disabled by default
	maxBufferedOutputLines = 100

	// Sandbox mode constants
	SandboxReadOnly         = "read-only"
	SandboxWorkspaceWrite   = "workspace-write"
	SandboxDangerFullAccess = "danger-full-access"
)

// ValidSandboxModes contains all valid sandbox mode values
var ValidSandboxModes = []string{SandboxReadOnly, SandboxWorkspaceWrite, SandboxDangerFullAccess}

// IsValidSandbox checks if the given sandbox value is valid
func IsValidSandbox(sandbox string) bool {
	for _, valid := range ValidSandboxModes {
		if sandbox == valid {
			return true
		}
	}
	return false
}

// Options represents the parameters for a Codex CLI execution
type Options struct {
	Prompt            string
	WorkingDir        string
	Sandbox           string
	SessionID         string
	SkipGitRepoCheck  bool
	ReturnAllMessages bool
	ImagePaths        []string
	Model             string
	Yolo              bool
	Profile           string
	Timeout           time.Duration
	NoOutputTimeout   time.Duration
	ExecutablePath    string
	MaxBufferedLines  int
	Reporter          progress.Reporter
}

// Result represents the parsed result from Codex CLI output
type Result struct {
	Success       bool
	SessionID     string
	AgentMessages string
	AllMessages   []map[string]interface{}
	ToolCallCount int
	Error         string
}

// Run executes the Codex CLI with the given options and returns the result.
// If ctx is canceled (e.g. client disconnects), the codex process is killed.
func Run(ctx context.Context, opts Options) (*Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	reporter := opts.Reporter
	if reporter == nil {
		reporter = progress.Nop
	}
	if opts.Timeout <= 0 {
		opts.Timeout = defaultCommandTimeout
	}
	if opts.Timeout > maxCommandTimeout {
		opts.Timeout = maxCommandTimeout
	}
	if opts.NoOutputTimeout <= 0 {
		opts.NoOutputTimeout = defaultNoOutputTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()
	reporter.Report(ctx, "initializing")

	prompt := opts.Prompt
	if runtime.GOOS == "windows" {
		prompt = escapePrompt(prompt)
	}
	sandbox := opts.Sandbox
	if sandbox == "" {
		sandbox = SandboxReadOnly
	}
	if !IsValidSandbox(sandbox) {
		return nil, cerrors.ErrInvalidSandboxMode(sandbox, ValidSandboxModes)
	}

	codexPath := strings.TrimSpace(opts.ExecutablePath)
	if codexPath == "" {
		lookPath, lookErr := exec.LookPath("codex")
		if lookErr != nil {
			return nil, cerrors.ErrCodexNotFound(lookErr)
		}
		codexPath = lookPath
	}

	// Build the base command
	cmd := exec.CommandContext(ctx, codexPath, "exec", "--sandbox", sandbox, "--cd", opts.WorkingDir, "--json")
	reporter.Report(ctx, "starting codex")

	// Add optional flags
	if len(opts.ImagePaths) > 0 {
		cmd.Args = append(cmd.Args, "--image", strings.Join(opts.ImagePaths, ","))
	}
	if opts.Model != "" {
		cmd.Args = append(cmd.Args, "--model", opts.Model)
	}
	if opts.Profile != "" {
		cmd.Args = append(cmd.Args, "--profile", opts.Profile)
	}
	if opts.Yolo {
		cmd.Args = append(cmd.Args, "--yolo")
	}
	if opts.SkipGitRepoCheck {
		cmd.Args = append(cmd.Args, "--skip-git-repo-check")
	}

	// Add session resume or prompt
	if opts.SessionID != "" {
		cmd.Args = append(cmd.Args, "resume", opts.SessionID)
	}

	// Add the prompt at the end
	cmd.Args = append(cmd.Args, "--", prompt)

	// Capture stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, cerrors.ErrCodexExecutionFailed("failed to create stdout pipe", err)
	}
	// Combine stderr into stdout so we surface all messages
	cmd.Stderr = cmd.Stdout

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, cerrors.ErrCodexExecutionFailed("failed to start codex command", err)
	}
	reporter.Report(ctx, "codex running")

	// Parse the output
	result := &Result{
		Success:     true,
		AllMessages: make([]map[string]interface{}, 0),
	}
	var runErr *cerrors.Error

	agentMessages := make([]string, 0)
	recentLines := make([]string, 0)
	bufferLimit := maxBufferedOutputLines
	if opts.MaxBufferedLines > 0 {
		bufferLimit = opts.MaxBufferedLines
	}

	lineCh := make(chan []byte)
	readErrCh := make(chan error, 1)

	go func() {
		defer close(lineCh)
		defer close(readErrCh)
		reader := bufio.NewReader(stdout)
		for {
			line, err := reader.ReadBytes('\n')
			if len(line) > 0 {
				lineCh <- bytes.TrimSpace(line)
			}
			if err != nil {
				if !errors.Is(err, io.EOF) {
					readErrCh <- err
				}
				return
			}
		}
	}()

	var noOutputTimer *time.Timer
	var noOutputCh <-chan time.Time
	lastOutput := time.Now()

	resetNoOutputTimer := func() {}
	if opts.NoOutputTimeout > 0 {
		noOutputTimer = time.NewTimer(opts.NoOutputTimeout)
		noOutputCh = noOutputTimer.C
		resetNoOutputTimer = func() {
			if noOutputTimer == nil {
				return
			}
			if !noOutputTimer.Stop() {
				select {
				case <-noOutputTimer.C:
				default:
				}
			}
			noOutputTimer.Reset(opts.NoOutputTimeout)
		}
	}
	defer func() {
		if noOutputTimer != nil {
			noOutputTimer.Stop()
		}
	}()

	var progressTicker *time.Ticker
	if reporter != progress.Nop {
		progressTicker = time.NewTicker(5 * time.Second)
		defer progressTicker.Stop()
	}
	reportedFirstOutput := false
	reportedThreadID := false
	reportedAgentMessage := false

drainLoop:
	for {
		select {
		case trimmed, ok := <-lineCh:
			if !ok {
				break drainLoop
			}
			resetNoOutputTimer()
			lastOutput = time.Now()
			if len(trimmed) == 0 {
				continue
			}

			if !reportedFirstOutput {
				reportedFirstOutput = true
				reporter.Report(ctx, "received output")
			}

			recentLines = append(recentLines, string(trimmed))
			if len(recentLines) > bufferLimit {
				recentLines = recentLines[1:]
			}

			var lineData map[string]interface{}
			if err := json.Unmarshal(trimmed, &lineData); err != nil {
				result.Success = false
				if result.Error == "" {
					result.Error = "failed to parse codex output as JSON"
				}
				if runErr == nil {
					runErr = cerrors.ErrCodexExecutionFailed(result.Error, err).
						WithData("line", string(trimmed))
				}
				continue
			}

			// Collect all messages if requested
			if opts.ReturnAllMessages {
				result.AllMessages = append(result.AllMessages, lineData)
			}

			// Best-effort tool call counting (independent of opts.ReturnAllMessages).
			if item, ok := lineData["item"].(map[string]interface{}); ok {
				if itemType, ok := item["type"].(string); ok {
					switch itemType {
					case "tool_call", "tool_use":
						result.ToolCallCount++
					}
				}
			}

			// Extract thread_id
			if threadID, ok := lineData["thread_id"].(string); ok && threadID != "" {
				result.SessionID = threadID
				if !reportedThreadID {
					reportedThreadID = true
					reporter.Report(ctx, "received SESSION_ID")
				}
			}

			// Extract agent messages
			if item, ok := lineData["item"].(map[string]interface{}); ok {
				if itemType, ok := item["type"].(string); ok && itemType == "agent_message" {
					if text, ok := item["text"].(string); ok {
						agentMessages = append(agentMessages, text)
						if !reportedAgentMessage {
							reportedAgentMessage = true
							reporter.Report(ctx, "received agent message")
						}
					}
				}
			}

			// Check for errors
			if lineType, ok := lineData["type"].(string); ok {
				if strings.Contains(lineType, "fail") || strings.Contains(lineType, "error") {
					result.Success = false
					if errMsg, ok := lineData["error"].(map[string]interface{}); ok {
						if msg, ok := errMsg["message"].(string); ok {
							result.Error = "codex error: " + msg
							if runErr == nil {
								runErr = cerrors.New(cerrors.CodexExecutionFailed, result.Error)
							}
						}
					} else if msg, ok := lineData["message"].(string); ok {
						result.Error = "codex error: " + msg
						if runErr == nil {
							runErr = cerrors.New(cerrors.CodexExecutionFailed, result.Error)
						}
					}
				}
			}
		case readErr, ok := <-readErrCh:
			if !ok {
				readErrCh = nil
				continue
			}
			if readErr != nil {
				result.Success = false
				if result.Error == "" {
					result.Error = "failed to read codex output"
				}
				if runErr == nil {
					runErr = cerrors.ErrCodexExecutionFailed(result.Error, readErr)
				}
			}
			readErrCh = nil
		case <-noOutputCh:
			result.Success = false
			if result.Error == "" {
				result.Error = "no output from codex"
			}
			if runErr == nil {
				runErr = cerrors.ErrNoOutputTimeout(opts.NoOutputTimeout).
					WithData("last_output_at", lastOutput.Format(time.RFC3339))
			}
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
			break drainLoop
		case <-ctx.Done():
			result.Success = false
			if result.Error == "" {
				result.Error = "codex execution canceled"
			}
			if runErr == nil {
				if errors.Is(ctx.Err(), context.DeadlineExceeded) {
					runErr = cerrors.ErrCodexTimeout(opts.Timeout)
				} else {
					runErr = cerrors.ErrCodexExecutionFailed(result.Error, ctx.Err())
				}
			}
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
			break drainLoop
		case <-func() <-chan time.Time {
			if progressTicker == nil {
				return nil
			}
			return progressTicker.C
		}():
			reporter.Report(ctx, "running")
		}
	}

	result.AgentMessages = strings.Join(agentMessages, "\n")
	reporter.Report(ctx, "finalizing")

	// Wait for command to finish
	if err := cmd.Wait(); err != nil {
		result.Success = false
		if result.Error == "" {
			result.Error = "codex command failed"
		}
		if runErr == nil {
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				runErr = cerrors.ErrCodexTimeout(opts.Timeout)
			} else if errors.Is(ctx.Err(), context.Canceled) {
				runErr = cerrors.ErrCodexExecutionFailed("codex execution canceled", ctx.Err())
			} else {
				runErr = cerrors.ErrCodexExecutionFailed(result.Error, err)
			}
		}
	}

	// Post-process validation
	if result.SessionID == "" {
		result.Success = false
		if result.Error == "" {
			result.Error = "failed to get SESSION_ID from the codex session"
		}
		if runErr == nil {
			runErr = cerrors.New(cerrors.CodexExecutionFailed, result.Error)
		}
	}

	if result.AgentMessages == "" {
		result.Success = false
		if result.Error == "" {
			result.Error = "failed to get agent_messages from the codex session"
		}
		if runErr == nil {
			runErr = cerrors.New(cerrors.CodexExecutionFailed, result.Error)
		}
	}

	if runErr != nil {
		if len(recentLines) > 0 {
			runErr.WithData("recent_output", recentLines)
		}
		return result, runErr
	}
	return result, nil
}

// escapePrompt mirrors the Python implementation to avoid Windows shell quoting issues.
func escapePrompt(prompt string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"\"", "\\\"",
		"\n", "\\n",
		"\r", "\\r",
		"\t", "\\t",
		"\b", "\\b",
		"\f", "\\f",
		"'", "\\'",
	)
	return replacer.Replace(prompt)
}
