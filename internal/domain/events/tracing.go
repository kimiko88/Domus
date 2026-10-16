package events

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

// W3C Trace Context constants
const (
	TraceVersion = "00"
	FlagSampled  = "01"
)

// TraceContext represents a W3C Trace Context (traceparent).
type TraceContext struct {
	TraceID      string // 16 bytes (32 hex characters)
	SpanID       string // 8 bytes (16 hex characters)
	ParentSpanID string // 8 bytes (16 hex characters), if child span
	Sampled      bool   // true if recorded/sampled
}

// String returns the canonical W3C traceparent header string (e.g. 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01).
func (tc *TraceContext) String() string {
	if tc == nil || tc.TraceID == "" || tc.SpanID == "" {
		return ""
	}
	flag := "00"
	if tc.Sampled {
		flag = FlagSampled
	}
	return fmt.Sprintf("%s-%s-%s-%s", TraceVersion, tc.TraceID, tc.SpanID, flag)
}

// ParseTraceParent parses a W3C traceparent string according to the W3C Trace Context Level 1 specification.
func ParseTraceParent(raw string) (*TraceContext, error) {
	raw = strings.TrimSpace(raw)
	parts := strings.Split(raw, "-")
	if len(parts) < 4 {
		return nil, errors.New("traceparent: invalid format, expected 4 hyphen-separated fields")
	}

	version := parts[0]
	traceID := parts[1]
	spanID := parts[2]
	flags := parts[3]

	if version != TraceVersion {
		return nil, fmt.Errorf("traceparent: unsupported version %q", version)
	}

	if len(traceID) != 32 || traceID == "00000000000000000000000000000000" {
		return nil, errors.New("traceparent: invalid trace-id (must be 32 non-zero hex chars)")
	}
	if _, err := hex.DecodeString(traceID); err != nil {
		return nil, fmt.Errorf("traceparent: trace-id is not hex: %w", err)
	}

	if len(spanID) != 16 || spanID == "0000000000000000" {
		return nil, errors.New("traceparent: invalid parent-id (must be 16 non-zero hex chars)")
	}
	if _, err := hex.DecodeString(spanID); err != nil {
		return nil, fmt.Errorf("traceparent: parent-id is not hex: %w", err)
	}

	sampled := len(flags) == 2 && (flags[1]&0x01) != 0

	return &TraceContext{
		TraceID: strings.ToLower(traceID),
		SpanID:  strings.ToLower(spanID),
		Sampled: sampled,
	}, nil
}

// NewTraceContext generates a new root W3C TraceContext with cryptographically secure random IDs.
func NewTraceContext() *TraceContext {
	traceBytes := make([]byte, 16)
	_, _ = rand.Read(traceBytes)
	spanBytes := make([]byte, 8)
	_, _ = rand.Read(spanBytes)

	return &TraceContext{
		TraceID: hex.EncodeToString(traceBytes),
		SpanID:  hex.EncodeToString(spanBytes),
		Sampled: true,
	}
}

// NewChildSpan creates a child span preserving the trace-id and setting the current span as parent.
func NewChildSpan(parent *TraceContext) *TraceContext {
	if parent == nil {
		return NewTraceContext()
	}

	childSpanBytes := make([]byte, 8)
	_, _ = rand.Read(childSpanBytes)

	return &TraceContext{
		TraceID:      parent.TraceID,
		SpanID:       hex.EncodeToString(childSpanBytes),
		ParentSpanID: parent.SpanID,
		Sampled:      parent.Sampled,
	}
}

type traceCtxKey struct{}

// ContextWithTrace returns a new Context containing the provided TraceContext.
func ContextWithTrace(ctx context.Context, tc *TraceContext) context.Context {
	return context.WithValue(ctx, traceCtxKey{}, tc)
}

// TraceFromContext returns the TraceContext stored in the context, or nil if none is present.
func TraceFromContext(ctx context.Context) *TraceContext {
	if ctx == nil {
		return nil
	}
	if v, ok := ctx.Value(traceCtxKey{}).(*TraceContext); ok {
		return v
	}
	return nil
}
