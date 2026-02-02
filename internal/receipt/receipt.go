package receipt

type FileChange struct {
	Path           string `json:"path"`
	IndexStatus    string `json:"index_status,omitempty"`
	WorktreeStatus string `json:"worktree_status,omitempty"`
}

type ChangeReceipt struct {
	ReceiptAvailable bool         `json:"receipt_available"`
	GitRoot          string       `json:"git_root,omitempty"`
	GitStatus        string       `json:"git_status,omitempty"`
	DiffStat         string       `json:"diff_stat,omitempty"`
	ChangedFiles     []FileChange `json:"changed_files,omitempty"`

	// Diff is included only when explicitly requested (e.g. return_diff=true),
	// and is always size-limited.
	Diff          string `json:"diff,omitempty"`
	DiffTruncated bool   `json:"diff_truncated,omitempty"`

	// ReceiptError contains best-effort diagnostics for why a receipt is unavailable.
	// It MUST NOT cause the parent tool call to fail.
	ReceiptError string `json:"receipt_error,omitempty"`
}
