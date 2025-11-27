package codex

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

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
}

// Result represents the parsed result from Codex CLI output
type Result struct {
	Success       bool
	SessionID     string
	AgentMessages string
	AllMessages   []map[string]interface{}
	Error         string
}

// Run executes the Codex CLI with the given options and returns the result
func Run(opts Options) (*Result, error) {
	prompt := opts.Prompt
	if runtime.GOOS == "windows" {
		prompt = escapePrompt(prompt)
	}
	sandbox := opts.Sandbox
	if sandbox == "" {
		sandbox = "read-only"
	}

	// Build the base command
	cmd := exec.Command("codex", "exec", "--sandbox", sandbox, "--cd", opts.WorkingDir, "--json")

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

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var lineData map[string]interface{}
		if err := json.Unmarshal([]byte(line), &lineData); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("JSON parse error: %v. Line: %s", err, line)
			break
		}

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
					result.AgentMessages += text
				}
			}
		}

		// Check for errors
		if lineType, ok := lineData["type"].(string); ok {
			if strings.Contains(lineType, "fail") || strings.Contains(lineType, "error") {
				if result.AgentMessages == "" {
					result.Success = false
				}
				if errMsg, ok := lineData["error"].(map[string]interface{}); ok {
					if msg, ok := errMsg["message"].(string); ok {
						result.Error = "codex error: " + msg
					}
				} else if msg, ok := lineData["message"].(string); ok {
					result.Error = "codex error: " + msg
				}
			}
		}
	}

	if scanErr := scanner.Err(); scanErr != nil {
		result.Success = false
		if result.Error == "" {
			result.Error = fmt.Sprintf("failed to read codex output: %v", scanErr)
		}
	}

	// Wait for command to finish
	if err := cmd.Wait(); err != nil {
		if result.Error == "" {
			result.Error = fmt.Sprintf("codex command failed: %v", err)
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
