package session

import (
	"strings"
	"time"
	"unicode/utf8"
)

type DiagnosticKind string

const (
	DiagnosticSystem   DiagnosticKind = "system"
	DiagnosticProgress DiagnosticKind = "progress"
	DiagnosticOutput   DiagnosticKind = "output"
)

type DiagnosticEntry struct {
	Seq     uint64
	At      time.Time
	Kind    DiagnosticKind
	Message string
}

type DiagnosticEntryView struct {
	Seq     uint64         `json:"seq"`
	At      string         `json:"ts"`
	Kind    DiagnosticKind `json:"kind"`
	Message string         `json:"message"`
}

func (e DiagnosticEntry) View() DiagnosticEntryView {
	return DiagnosticEntryView{
		Seq:     e.Seq,
		At:      e.At.UTC().Format(time.RFC3339),
		Kind:    e.Kind,
		Message: e.Message,
	}
}

func recentViews(entries []DiagnosticEntry, limit int) []DiagnosticEntryView {
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}
	if len(entries) == 0 {
		return nil
	}
	start := 0
	if len(entries) > limit {
		start = len(entries) - limit
	}
	out := make([]DiagnosticEntryView, 0, len(entries)-start)
	for _, e := range entries[start:] {
		out = append(out, e.View())
	}
	return out
}

func truncate(s string, maxBytes int) string {
	if maxBytes <= 0 || s == "" {
		return s
	}
	if len(s) <= maxBytes {
		return s
	}
	// Avoid cutting in the middle of a multi-byte rune by backing off
	// to the nearest valid boundary (best-effort).
	out := s[:maxBytes]
	for !utf8.ValidString(out) && maxBytes > 0 {
		maxBytes--
		out = s[:maxBytes]
	}
	return strings.TrimRight(out, "\n")
}
