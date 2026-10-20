package camera

import (
	"context"
	"math"
	"os"
	"testing"
)

func TestInferenceEngine_DetectHardwareBackend(t *testing.T) {
	// 1. Default fallback to CPU SIMD
	_ = os.Unsetenv("DOMUS_AI_BACKEND")
	_ = os.Unsetenv("DOMUS_FORCE_CORAL")
	_ = os.Unsetenv("DOMUS_FORCE_HAILO")

	backend := DetectHardwareBackend()
	if backend == nil {
		t.Fatalf("expected non-nil default backend")
	}
	if backend.Type() != BackendCPUSIMD {
		t.Errorf("expected default backend %s, got %s", BackendCPUSIMD, backend.Type())
	}
	if !backend.IsAvailable() {
		t.Errorf("expected cpu backend to be available")
	}

	// 2. Forced Coral TPU override
	t.Setenv("DOMUS_AI_BACKEND", "coral")
	coral := DetectHardwareBackend()
	if coral.Type() != BackendCoralTPU {
		t.Errorf("expected %s with DOMUS_AI_BACKEND=coral, got %s", BackendCoralTPU, coral.Type())
	}

	// 3. Forced Hailo-8 override
	t.Setenv("DOMUS_AI_BACKEND", "hailo")
	hailo := DetectHardwareBackend()
	if hailo.Type() != BackendHailo8 {
		t.Errorf("expected %s with DOMUS_AI_BACKEND=hailo, got %s", BackendHailo8, hailo.Type())
	}
}

func TestInferenceEngine_VectorNormalizationAndEmbedding(t *testing.T) {
	ctx := context.Background()
	backend := NewCPUSIMDBackend()

	// 1. Generate 128-D embedding from sample image payload
	img := []byte("sample_jpeg_face_frame_binary_stream_marco_rossi")
	emb, err := backend.ExtractFaceEmbedding(ctx, img)
	if err != nil {
		t.Fatalf("ExtractFaceEmbedding failed: %v", err)
	}
	if len(emb) != 128 {
		t.Fatalf("expected 128-D vector, got length %d", len(emb))
	}

	// 2. Verify L2 Euclidean norm is approximately 1.0
	var sumSquares float64
	for _, val := range emb {
		sumSquares += float64(val) * float64(val)
	}
	norm := math.Sqrt(sumSquares)
	if math.Abs(norm-1.0) > 1e-4 {
		t.Errorf("expected L2 norm 1.0, got %f", norm)
	}

	// 3. Determinism: identical input produces identical embedding
	emb2, _ := backend.ExtractFaceEmbedding(ctx, img)
	for i := range emb {
		if emb[i] != emb2[i] {
			t.Errorf("expected deterministic embedding, mismatch at index %d: %f vs %f", i, emb[i], emb2[i])
			break
		}
	}

	// 4. Empty payload error
	_, errEmpty := backend.ExtractFaceEmbedding(ctx, nil)
	if errEmpty == nil {
		t.Errorf("expected error on empty image payload")
	}
}

func TestInferenceEngine_ObjectDetectionAndALPR(t *testing.T) {
	ctx := context.Background()

	backends := []InferenceBackend{
		NewCoralTPUBackend(""),
		NewHailo8Backend(""),
		NewCPUSIMDBackend(),
	}

	for _, b := range backends {
		t.Run(string(b.Type()), func(t *testing.T) {
			frame := []byte("sample_h264_frame_data_for_bounding_box")

			// Object Detection
			detections, err := b.DetectObjects(ctx, frame)
			if err != nil {
				t.Fatalf("DetectObjects failed on %s: %v", b.Name(), err)
			}
			if len(detections) == 0 {
				t.Fatalf("expected detections on %s", b.Name())
			}
			det := detections[0]
			if det.Confidence <= 0.5 {
				t.Errorf("expected confidence > 0.5, got %f", det.Confidence)
			}
			if det.Box.Width <= 0 || det.Box.Height <= 0 {
				t.Errorf("invalid bounding box: %+v", det.Box)
			}

			// ALPR Plate Recognition
			plate, score, err := b.RecognizePlate(ctx, frame)
			if err != nil {
				t.Fatalf("RecognizePlate failed on %s: %v", b.Name(), err)
			}
			if plate == "" || score < 0.65 {
				t.Errorf("expected valid recognized plate on %s, got plate=%s, score=%f", b.Name(), plate, score)
			}
		})
	}
}
