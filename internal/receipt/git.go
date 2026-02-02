package receipt

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

type CollectOptions struct {
	ReturnDiff   bool
	MaxDiffBytes int

	// Timeout bounds the total time spent in Git collection.
	Timeout time.Duration
}

const (
	defaultCollectTimeout = 3 * time.Second
	defaultMaxDiffBytes   = 64 * 1024
)

func Collect(ctx context.Context, cd string, opts CollectOptions) ChangeReceipt {
	if ctx == nil {
		ctx = context.Background()
	}
	if opts.Timeout <= 0 {
		opts.Timeout = defaultCollectTimeout
	}
	if opts.MaxDiffBytes <= 0 {
		opts.MaxDiffBytes = defaultMaxDiffBytes
	}

	receipt := ChangeReceipt{}

	if _, err := exec.LookPath("git"); err != nil {
		receipt.ReceiptAvailable = false
		receipt.ReceiptError = "git not found"
		return receipt
	}

	gitRoot, ok, err := GitRoot(ctx, cd, opts.Timeout)
	if err != nil {
		receipt.ReceiptAvailable = false
		receipt.ReceiptError = err.Error()
		return receipt
	}
	if !ok || strings.TrimSpace(gitRoot) == "" {
		receipt.ReceiptAvailable = false
		return receipt
	}

	receipt.ReceiptAvailable = true
	receipt.GitRoot = gitRoot

	status, err := runGit(ctx, cd, opts.Timeout, "status", "--porcelain=v1")
	if err != nil {
		receipt.ReceiptError = fmt.Sprintf("git status failed: %v", err)
		return receipt
	}
	receipt.GitStatus = status
	receipt.ChangedFiles = parsePorcelainV1(status)

	diffStat, err := runGit(ctx, cd, opts.Timeout, "diff", "--stat")
	if err != nil {
		receipt.ReceiptError = fmt.Sprintf("git diff --stat failed: %v", err)
		return receipt
	}
	receipt.DiffStat = diffStat

	if opts.ReturnDiff {
		diff, truncated, err := runGitTruncated(ctx, cd, opts.Timeout, opts.MaxDiffBytes, "diff")
		if err != nil {
			receipt.ReceiptError = fmt.Sprintf("git diff failed: %v", err)
			return receipt
		}
		receipt.Diff = diff
		receipt.DiffTruncated = truncated
	}

	return receipt
}

// GitRoot returns the repository root for cd if it is inside a Git work tree.
func GitRoot(ctx context.Context, cd string, timeout time.Duration) (root string, ok bool, err error) {
	out, err := runGit(ctx, cd, timeout, "rev-parse", "--show-toplevel")
	if err != nil {
		// Treat common "not a repository" as "not ok".
		if isNotGitRepo(err) {
			return "", false, nil
		}
		return "", false, err
	}
	root = strings.TrimSpace(out)
	if root == "" {
		return "", false, nil
	}
	return root, true, nil
}

func runGit(ctx context.Context, cd string, timeout time.Duration, args ...string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if timeout <= 0 {
		timeout = defaultCollectTimeout
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, "git", append([]string{"-C", cd}, args...)...)
	out, err := cmd.CombinedOutput()
	if runCtx.Err() != nil {
		return "", runCtx.Err()
	}
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return "", err
		}
		return "", fmt.Errorf("%w: %s", err, msg)
	}
	return strings.TrimRight(string(out), "\n"), nil
}

func runGitTruncated(ctx context.Context, cd string, timeout time.Duration, maxBytes int, args ...string) (out string, truncated bool, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if timeout <= 0 {
		timeout = defaultCollectTimeout
	}
	if maxBytes <= 0 {
		maxBytes = defaultMaxDiffBytes
	}

	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, "git", append([]string{"-C", cd}, args...)...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", false, err
	}
	// Merge stderr into stdout so callers see useful error context.
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return "", false, err
	}

	var b strings.Builder
	b.Grow(min(maxBytes, 8*1024))

	reader := bufio.NewReader(stdout)
	buf := make([]byte, 4096)
	written := 0
	for {
		n, readErr := reader.Read(buf)
		if n > 0 {
			remaining := maxBytes - written
			if remaining > 0 {
				if n > remaining {
					b.Write(buf[:remaining])
					written += remaining
					truncated = true
				} else {
					b.Write(buf[:n])
					written += n
				}
			} else {
				truncated = true
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			return "", false, readErr
		}
	}

	if err := cmd.Wait(); err != nil {
		if runCtx.Err() != nil {
			return "", false, runCtx.Err()
		}
		msg := strings.TrimSpace(b.String())
		if msg == "" {
			return "", false, err
		}
		return "", false, fmt.Errorf("%w: %s", err, msg)
	}

	out = strings.TrimRight(b.String(), "\n")
	return out, truncated, nil
}

func parsePorcelainV1(status string) []FileChange {
	lines := strings.Split(status, "\n")
	out := make([]FileChange, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		if strings.HasPrefix(line, "?? ") {
			path := strings.TrimSpace(strings.TrimPrefix(line, "?? "))
			if path == "" {
				continue
			}
			out = append(out, FileChange{
				Path:           path,
				IndexStatus:    "?",
				WorktreeStatus: "?",
			})
			continue
		}
		if len(line) < 3 {
			continue
		}
		indexStatus := string(line[0])
		worktreeStatus := string(line[1])
		path := strings.TrimSpace(line[3:])
		if path == "" {
			continue
		}
		out = append(out, FileChange{
			Path:           path,
			IndexStatus:    strings.TrimSpace(indexStatus),
			WorktreeStatus: strings.TrimSpace(worktreeStatus),
		})
	}
	return out
}

func isNotGitRepo(err error) bool {
	if err == nil {
		return false
	}
	// "git -C <dir> rev-parse" typically exits with status 128 and prints:
	// "fatal: not a git repository ..."
	return errors.Is(err, exec.ErrNotFound) || strings.Contains(err.Error(), "not a git repository")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
