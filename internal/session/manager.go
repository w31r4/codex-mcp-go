package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	cerrors "github.com/w31r4/codex-mcp-go/internal/errors"
	"github.com/w31r4/codex-mcp-go/internal/receipt"
)

type State string

const (
	StateRunning   State = "running"
	StateCompleted State = "completed"
	StateFailed    State = "failed"
	StateCancelled State = "cancelled"
)

type Options struct {
	MaxRunning int
	TTL        time.Duration

	DiagnosticsMaxEntries    int
	DiagnosticsMaxEntryBytes int
}

func DefaultOptions() Options {
	return Options{
		MaxRunning:               4,
		TTL:                      time.Hour,
		DiagnosticsMaxEntries:    200,
		DiagnosticsMaxEntryBytes: 2048,
	}
}

type Record struct {
	ID      string
	State   State
	WorkDir string
	Sandbox string

	StartedAt time.Time
	EndedAt   *time.Time

	ExecutionTimeMs int64
	ToolCallCount   int

	Error string

	cancel context.CancelFunc

	ChangeReceipt *receipt.ChangeReceipt

	diagNextSeq  uint64
	diagnostics  []DiagnosticEntry
	lastEventAt  *time.Time
	lastOutputAt *time.Time
}

type View struct {
	SessionID string `json:"SESSION_ID"`
	State     State  `json:"state"`
	WorkDir   string `json:"cd"`
	Sandbox   string `json:"sandbox"`
	StartedAt string `json:"started_at"`
	EndedAt   string `json:"ended_at,omitempty"`

	ExecutionTimeMs int64 `json:"execution_time_ms,omitempty"`
	ToolCallCount   int   `json:"tool_call_count,omitempty"`

	Error string `json:"error,omitempty"`
}

func (r *Record) View() View {
	if r == nil {
		return View{}
	}
	v := View{
		SessionID:       r.ID,
		State:           r.State,
		WorkDir:         r.WorkDir,
		Sandbox:         r.Sandbox,
		StartedAt:       r.StartedAt.UTC().Format(time.RFC3339),
		ExecutionTimeMs: r.ExecutionTimeMs,
		ToolCallCount:   r.ToolCallCount,
		Error:           r.Error,
	}
	if r.EndedAt != nil {
		v.EndedAt = r.EndedAt.UTC().Format(time.RFC3339)
	}
	return v
}

type DetailView struct {
	View

	LastEventAt   string                 `json:"last_event_at,omitempty"`
	LastOutputAt  string                 `json:"last_output_at,omitempty"`
	Recent        []DiagnosticEntryView  `json:"recent_entries,omitempty"`
	ChangeReceipt *receipt.ChangeReceipt `json:"change_receipt,omitempty"`
}

func (r *Record) DetailView(limit int) DetailView {
	if r == nil {
		return DetailView{}
	}
	if limit <= 0 {
		limit = 20
	}
	entries := recentViews(r.diagnostics, limit)
	dv := DetailView{
		View:          r.View(),
		Recent:        entries,
		ChangeReceipt: r.ChangeReceipt,
	}
	if r.lastEventAt != nil {
		dv.LastEventAt = r.lastEventAt.UTC().Format(time.RFC3339)
	}
	if r.lastOutputAt != nil {
		dv.LastOutputAt = r.lastOutputAt.UTC().Format(time.RFC3339)
	}
	return dv
}

type Manager struct {
	mu       sync.Mutex
	opts     Options
	sessions map[string]*Record
}

func NewManager(opts Options) *Manager {
	if opts.MaxRunning == 0 {
		opts.MaxRunning = DefaultOptions().MaxRunning
	}
	if opts.TTL == 0 {
		opts.TTL = DefaultOptions().TTL
	}
	if opts.DiagnosticsMaxEntries == 0 {
		opts.DiagnosticsMaxEntries = DefaultOptions().DiagnosticsMaxEntries
	}
	if opts.DiagnosticsMaxEntryBytes == 0 {
		opts.DiagnosticsMaxEntryBytes = DefaultOptions().DiagnosticsMaxEntryBytes
	}
	return &Manager{
		opts:     opts,
		sessions: make(map[string]*Record),
	}
}

func NewTemporaryID() string {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err == nil {
		return "tmp_" + hex.EncodeToString(buf)
	}
	return fmt.Sprintf("tmp_%d", time.Now().UnixNano())
}

func (m *Manager) Start(sessionID string, workDir string, sandbox string, cancel context.CancelFunc) (*Record, error) {
	if stringsTrim(sessionID) == "" {
		return nil, cerrors.ErrInvalidParams("SESSION_ID is required")
	}
	if cancel == nil {
		return nil, cerrors.ErrInvalidParams("cancel func is required")
	}

	now := time.Now()

	m.mu.Lock()
	defer m.mu.Unlock()

	m.cleanupExpiredLocked(now)

	if r, ok := m.sessions[sessionID]; ok && r.State == StateRunning {
		return nil, cerrors.ErrInvalidParams("session is already running")
	}

	running := 0
	for _, r := range m.sessions {
		if r.State == StateRunning {
			running++
		}
	}
	if m.opts.MaxRunning > 0 && running >= m.opts.MaxRunning {
		return nil, cerrors.New(cerrors.SessionLimitExceeded, "too many concurrent sessions").
			WithData("max_running", m.opts.MaxRunning).
			WithData("running", running)
	}

	rec := &Record{
		ID:        sessionID,
		State:     StateRunning,
		WorkDir:   workDir,
		Sandbox:   sandbox,
		StartedAt: now,
		cancel:    cancel,
	}
	m.sessions[sessionID] = rec
	return rec, nil
}

func (m *Manager) UpdateID(oldID string, newID string) (bool, error) {
	oldID = stringsTrim(oldID)
	newID = stringsTrim(newID)
	if oldID == "" || newID == "" || oldID == newID {
		return false, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	rec, ok := m.sessions[oldID]
	if !ok {
		return false, nil
	}
	if _, exists := m.sessions[newID]; exists {
		return false, cerrors.ErrInvalidParams("new SESSION_ID already exists")
	}

	delete(m.sessions, oldID)
	rec.ID = newID
	m.sessions[newID] = rec
	return true, nil
}

func (m *Manager) MarkCompleted(sessionID string, executionTimeMs int64, toolCallCount int) bool {
	return m.finish(sessionID, StateCompleted, "", executionTimeMs, toolCallCount)
}

func (m *Manager) MarkFailed(sessionID string, err error) bool {
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	return m.finish(sessionID, StateFailed, msg, 0, 0)
}

func (m *Manager) MarkCancelled(sessionID string, reason string) bool {
	return m.finish(sessionID, StateCancelled, reason, 0, 0)
}

func (m *Manager) SetChangeReceipt(sessionID string, receipt receipt.ChangeReceipt) bool {
	sessionID = stringsTrim(sessionID)
	if sessionID == "" {
		return false
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	rec, ok := m.sessions[sessionID]
	if !ok {
		return false
	}
	r := receipt
	rec.ChangeReceipt = &r
	return true
}

func (m *Manager) AppendDiagnostic(sessionID string, kind DiagnosticKind, message string) bool {
	sessionID = stringsTrim(sessionID)
	if sessionID == "" {
		return false
	}

	now := time.Now()

	m.mu.Lock()
	defer m.mu.Unlock()

	rec, ok := m.sessions[sessionID]
	if !ok {
		return false
	}

	rec.diagNextSeq++
	entry := DiagnosticEntry{
		Seq:     rec.diagNextSeq,
		At:      now,
		Kind:    kind,
		Message: truncate(message, m.opts.DiagnosticsMaxEntryBytes),
	}
	rec.diagnostics = append(rec.diagnostics, entry)
	if max := m.opts.DiagnosticsMaxEntries; max > 0 && len(rec.diagnostics) > max {
		rec.diagnostics = rec.diagnostics[len(rec.diagnostics)-max:]
	}
	rec.lastEventAt = &now
	if kind == DiagnosticOutput {
		rec.lastOutputAt = &now
	}
	return true
}

func (m *Manager) TailDiagnostics(sessionID string, cursor uint64, limit int) (entries []DiagnosticEntryView, nextCursor uint64, dropped bool, droppedBefore uint64, state State, found bool) {
	sessionID = stringsTrim(sessionID)
	if sessionID == "" {
		return nil, cursor, false, 0, "", false
	}

	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	rec, ok := m.sessions[sessionID]
	if !ok {
		return nil, cursor, false, 0, "", false
	}
	state = rec.State
	found = true

	if len(rec.diagnostics) == 0 {
		return nil, cursor, false, 0, state, true
	}

	oldest := rec.diagnostics[0].Seq
	if cursor != 0 && cursor < oldest {
		dropped = true
		droppedBefore = oldest
		// Reset cursor so the client can resume from what we still have.
		cursor = oldest - 1
	}

	entries = make([]DiagnosticEntryView, 0, limit)
	for _, e := range rec.diagnostics {
		if e.Seq <= cursor {
			continue
		}
		entries = append(entries, e.View())
		if len(entries) >= limit {
			break
		}
	}

	nextCursor = cursor
	if len(entries) > 0 {
		nextCursor = entries[len(entries)-1].Seq
	}
	return entries, nextCursor, dropped, droppedBefore, state, true
}

func (m *Manager) GetDetail(sessionID string, recentLimit int) (DetailView, bool) {
	sessionID = stringsTrim(sessionID)
	if sessionID == "" {
		return DetailView{}, false
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	rec, ok := m.sessions[sessionID]
	if !ok {
		return DetailView{}, false
	}
	return rec.DetailView(recentLimit), true
}

func (m *Manager) Cancel(sessionID string) (bool, error) {
	sessionID = stringsTrim(sessionID)
	if sessionID == "" {
		return false, cerrors.ErrInvalidParams("SESSION_ID is required")
	}

	now := time.Now()

	m.mu.Lock()
	defer m.mu.Unlock()

	rec, ok := m.sessions[sessionID]
	if !ok {
		return false, cerrors.New(cerrors.SessionNotFound, "session not found").WithData("SESSION_ID", sessionID)
	}
	if rec.State != StateRunning {
		return false, nil
	}

	rec.State = StateCancelled
	rec.Error = "cancel requested"
	rec.ExecutionTimeMs = 0
	rec.ToolCallCount = 0
	rec.EndedAt = &now
	if rec.cancel != nil {
		rec.cancel()
		rec.cancel = nil
	}
	return true, nil
}

func (m *Manager) Get(sessionID string) (View, bool) {
	sessionID = stringsTrim(sessionID)
	if sessionID == "" {
		return View{}, false
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	rec, ok := m.sessions[sessionID]
	if !ok {
		return View{}, false
	}
	return rec.View(), true
}

func (m *Manager) List() []View {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make([]View, 0, len(m.sessions))
	records := make([]*Record, 0, len(m.sessions))
	for _, r := range m.sessions {
		records = append(records, r)
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].StartedAt.After(records[j].StartedAt)
	})
	for _, r := range records {
		out = append(out, r.View())
	}
	return out
}

func (m *Manager) CleanupExpired(now time.Time) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cleanupExpiredLocked(now)
}

func (m *Manager) cleanupExpiredLocked(now time.Time) int {
	if m.opts.TTL < 0 {
		return 0
	}
	removed := 0
	for id, r := range m.sessions {
		if r.State == StateRunning || r.EndedAt == nil {
			continue
		}
		if now.Sub(*r.EndedAt) > m.opts.TTL {
			delete(m.sessions, id)
			removed++
		}
	}
	return removed
}

func (m *Manager) StartCleanup(ctx context.Context, interval time.Duration) {
	if m == nil {
		return
	}
	if interval <= 0 {
		interval = time.Minute
	}
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				m.CleanupExpired(time.Now())
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (m *Manager) finish(sessionID string, state State, errMsg string, executionTimeMs int64, toolCallCount int) bool {
	sessionID = stringsTrim(sessionID)
	if sessionID == "" {
		return false
	}

	now := time.Now()

	m.mu.Lock()
	defer m.mu.Unlock()

	rec, ok := m.sessions[sessionID]
	if !ok {
		return false
	}

	// Don't override an explicit cancellation (e.g. cancel_session).
	if rec.State == StateCancelled {
		return true
	}

	rec.State = state
	rec.Error = errMsg
	rec.ExecutionTimeMs = executionTimeMs
	rec.ToolCallCount = toolCallCount
	rec.EndedAt = &now
	rec.cancel = nil

	m.cleanupExpiredLocked(now)
	return true
}

func stringsTrim(s string) string {
	return strings.TrimSpace(s)
}
