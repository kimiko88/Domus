package events_test

import (
	"context"
	"strings"
	"testing"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTraceContext_NewAndFormat(t *testing.T) {
	tc := events.NewTraceContext()
	require.NotNil(t, tc)

	assert.Len(t, tc.TraceID, 32)
	assert.Len(t, tc.SpanID, 16)
	assert.True(t, tc.Sampled)

	header := tc.String()
	assert.True(t, strings.HasPrefix(header, "00-"))
	assert.True(t, strings.HasSuffix(header, "-01"))

	// Parse it back
	parsed, err := events.ParseTraceParent(header)
	require.NoError(t, err)
	assert.Equal(t, tc.TraceID, parsed.TraceID)
	assert.Equal(t, tc.SpanID, parsed.SpanID)
	assert.True(t, parsed.Sampled)
}

func TestTraceContext_ParseW3C_ValidAndInvalid(t *testing.T) {
	// Standard W3C example
	validHeader := "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
	tc, err := events.ParseTraceParent(validHeader)
	require.NoError(t, err)
	assert.Equal(t, "4bf92f3577b34da6a3ce929d0e0e4736", tc.TraceID)
	assert.Equal(t, "00f067aa0ba902b7", tc.SpanID)
	assert.True(t, tc.Sampled)

	// Not sampled flag
	notSampled := "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-00"
	tc, err = events.ParseTraceParent(notSampled)
	require.NoError(t, err)
	assert.False(t, tc.Sampled)

	// Invalid versions
	_, err = events.ParseTraceParent("01-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	assert.Error(t, err)

	// All zeros trace ID
	_, err = events.ParseTraceParent("00-00000000000000000000000000000000-00f067aa0ba902b7-01")
	assert.Error(t, err)

	// All zeros span ID
	_, err = events.ParseTraceParent("00-4bf92f3577b34da6a3ce929d0e0e4736-0000000000000000-01")
	assert.Error(t, err)

	// Non-hex
	_, err = events.ParseTraceParent("00-4bf92f3577b34da6a3ce929d0e0e47zz-00f067aa0ba902b7-01")
	assert.Error(t, err)
}

func TestTraceContext_ChildSpan(t *testing.T) {
	parent := events.NewTraceContext()
	child := events.NewChildSpan(parent)

	assert.Equal(t, parent.TraceID, child.TraceID)
	assert.NotEqual(t, parent.SpanID, child.SpanID)
	assert.Equal(t, parent.SpanID, child.ParentSpanID)
	assert.Equal(t, parent.Sampled, child.Sampled)
}

func TestTraceContext_ContextInjection(t *testing.T) {
	ctx := context.Background()
	assert.Nil(t, events.TraceFromContext(ctx))

	tc := events.NewTraceContext()
	ctx = events.ContextWithTrace(ctx, tc)
	extracted := events.TraceFromContext(ctx)
	require.NotNil(t, extracted)
	assert.Equal(t, tc.TraceID, extracted.TraceID)
}
