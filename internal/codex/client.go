package codex

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const (
	defaultCommandTimeout  = 60 * time.Minute
	maxCommandTimeout      = 60 * time.Minute
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

// StreamEvent represents a single output event from Codex CLI
type StreamEvent struct {
	RawLine      string                 // Raw output line
	IsJSON       bool                   // Whether the line is valid JSON
	JSON         map[string]interface{} // Parsed JSON data (nil if not JSON)
	AgentMessage string                 // Extracted agent message text (if any)
}

// StreamHandler is a callback function for streaming output events
type StreamHandler func(StreamEvent)

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
	// Streaming and logging options
	StreamHandler StreamHandler // Callback for each output line
	Logger        *slog.Logger  // Logger for debug output
	Debug         bool          // Enable debug logging
}

// Result represents the parsed result from Codex CLI output
type Result struct {
	Success       bool
	SessionID     string
	AgentMessages string
	AllMessages   []map[string]interface{}
	Error         string
}

// Run executes the Codex CLI with the given options and returns the result.
// If ctx is canceled (e.g. client disconnects), the codex process is killed.
func Run(ctx context.Context, opts Options) (*Result, error) {
	if ctx == nil {
		ctx = context.Background()
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

	prompt := opts.Prompt
	if runtime.GOOS == "windows" {
		prompt = escapePrompt(prompt)
	}
	sandbox := opts.Sandbox
	if sandbox == "" {
		sandbox = SandboxReadOnly
	}
	if !IsValidSandbox(sandbox) {
		return nil, fmt.Errorf("invalid sandbox mode %q: must be one of %v", sandbox, ValidSandboxModes)
	}

	// Initialize logger
	logger := opts.Logger
	if logger == nil && opts.Debug {
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	}

	// Helper functions for logging and streaming
	logDebug := func(msg string, args ...any) {
		if logger != nil {
			logger.Debug(msg, args...)
		}
	}

	emitStream := func(evt StreamEvent) {
		if opts.StreamHandler == nil {
			return
		}
		defer func() {
			if r := recover(); r != nil {
				logDebug("stream handler panic", "panic", r)
			}
		}()
		opts.StreamHandler(evt)
	}

	codexPath, lookErr := exec.LookPath("codex")
	if lookErr != nil {
		return nil, fmt.Errorf("codex executable not found in PATH: %w", lookErr)
	}

	// Build the base command
	cmd := exec.CommandContext(ctx, codexPath, "exec", "--sandbox", sandbox, "--cd", opts.WorkingDir, "--json")

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

	// Log command start (hide prompt for security)
	if logger != nil {
		args := make([]string, len(cmd.Args))
		copy(args, cmd.Args)
		if len(args) > 0 {
			args[len(args)-1] = "<prompt>"
		}
		logger.Debug("starting codex", "args", args)
	}

	// Capture stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	// Combine stderr into stdout so we surface all messages
	cmd.Stderr = cmd.Stdout

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start codex command: %w", err)
	}

	// Parse the output
	result := &Result{
		Success:     true,
		AllMessages: make([]map[string]interface{}, 0),
	}

	agentMessages := make([]string, 0)
	recentLines := make([]string, 0)

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

			recentLines = append(recentLines, string(trimmed))
			if len(recentLines) > maxBufferedOutputLines {
				recentLines = recentLines[1:]
			}

			// Create stream event
			evt := StreamEvent{RawLine: string(trimmed)}

			var lineData map[string]interface{}
			if err := json.Unmarshal(trimmed, &lineData); err != nil {
				// Skip non-JSON lines instead of failing
				// Codex may output debug info or error messages that are not JSON
				logDebug("skipping non-json output", "error", err, "line", evt.RawLine)
				emitStream(evt)
				continue
			}

			evt.IsJSON = true
			evt.JSON = lineData

			// Collect all messages if requested
			if opts.ReturnAllMessages {
				result.AllMessages = append(result.AllMessages, lineData)
			}

			// Extract thread_id
			if threadID, ok := lineData["thread_id"].(string); ok && threadID != "" {
				result.SessionID = threadID
			}

			// Extract agent messages
			if item, ok := lineData["item"].(map[string]interface{}); ok {
				if itemType, ok := item["type"].(string); ok && itemType == "agent_message" {
					if text, ok := item["text"].(string); ok {
						agentMessages = append(agentMessages, text)
						evt.AgentMessage = text
					}
				}
			}

			// Emit stream event
			emitStream(evt)

			// Note: We don't check for intermediate errors here because Codex has
			// a retry mechanism (up to 5 reconnection attempts on network issues).
			// Setting Success=false on first error would incorrectly fail the whole
			// operation. Instead, we judge success at the end based on whether we
			// got valid SessionID and AgentMessages.
		case readErr, ok := <-readErrCh:
			if !ok {
				readErrCh = nil
				continue
			}
			if readErr != nil {
				result.Success = false
				if result.Error == "" {
					result.Error = fmt.Sprintf("failed to read codex output: %v", readErr)
				}
			}
			readErrCh = nil
		case <-noOutputCh:
			result.Success = false
			if result.Error == "" {
				result.Error = fmt.Sprintf("no output from codex for %s (last output at %s)", opts.NoOutputTimeout, lastOutput.Format(time.RFC3339))
				if len(recentLines) > 0 {
					result.Error += "\nrecent output:\n" + strings.Join(recentLines, "\n")
				}
			}
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
			break drainLoop
		case <-ctx.Done():
			result.Success = false
			if result.Error == "" {
				result.Error = fmt.Sprintf("codex command context canceled: %v", ctx.Err())
			}
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
			break drainLoop
		}
	}

	result.AgentMessages = strings.Join(agentMessages, "\n")

	// Wait for command to finish
	if err := cmd.Wait(); err != nil {
		result.Success = false
		if errors.Is(ctx.Err(), context.Canceled) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			if result.Error == "" {
				result.Error = fmt.Sprintf("codex command canceled: %v", ctx.Err())
			}
		} else if result.Error == "" {
			result.Error = fmt.Sprintf("codex command failed: %v", err)
			if len(recentLines) > 0 {
				result.Error += "\nrecent output:\n" + strings.Join(recentLines, "\n")
			}
		}
	}

	// Post-process validation
	if result.SessionID == "" {
		result.Success = false
		result.Error = "Failed to get SESSION_ID from the codex session. \n\n" + result.Error
	}

	if result.AgentMessages == "" {
		result.Success = false
		result.Error = "Failed to get agent_messages from the codex session. \n\n You can try to set return_all_messages to true to get the full reasoning information. \n\n " + result.Error
	}

	// Log completion
	if logger != nil {
		if result.Success {
			logger.Debug("codex completed", "session_id", result.SessionID)
		} else {
			logger.Debug("codex failed", "error", result.Error)
		}
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
