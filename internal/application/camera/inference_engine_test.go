package camera

import (
	"context"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInferenceEngine_FailsClosedWithoutRealModels(t *testing.T) {
	t.Setenv("DOMUS_AI_BACKEND", "coral")
	t.Setenv("DOMUS_FORCE_CORAL", "true")
	t.Setenv("DOMUS_FORCE_HAILO", "true")
	backends := []InferenceBackend{DetectHardwareBackend(), NewCoralTPUBackend(""), NewHailo8Backend(""), NewCPUSIMDBackend()}
	for _, backend := range backends {
		require.False(t, backend.IsAvailable())
		_, err := backend.DetectObjects(context.Background(), []byte("frame"))
		require.ErrorIs(t, err, ErrInferenceModelNotConfigured)
		_, _, err = backend.RecognizePlate(context.Background(), []byte("frame"))
		require.ErrorIs(t, err, ErrInferenceModelNotConfigured)
		_, err = backend.ExtractFaceEmbedding(context.Background(), []byte("image"))
		require.ErrorIs(t, err, ErrInferenceModelNotConfigured)
	}
}

func TestInferenceEngine_NormalizeVector(t *testing.T) {
	input := []float32{3, 4}
	normalized := NormalizeVector(input)
	require.Len(t, normalized, 2)
	norm := math.Sqrt(float64(normalized[0]*normalized[0] + normalized[1]*normalized[1]))
	require.InDelta(t, 1.0, norm, 1e-6)
	require.Equal(t, []float32{3, 4}, input)
}
