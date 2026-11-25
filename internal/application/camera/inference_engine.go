package camera

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
)

type BackendType string

const (
	BackendCoralTPU BackendType = "coral_tpu"
	BackendHailo8 BackendType = "hailo_8"
	BackendCPUSIMD BackendType = "cpu_simd"
	BackendUnavailable BackendType = "unavailable"
)

var ErrInferenceModelNotConfigured = errors.New("camera: no real inference model/runtime is configured")

type InferenceBackend interface {
	Name() string
	Type() BackendType
	IsAvailable() bool
	ExtractFaceEmbedding(ctx context.Context, imgData []byte) ([]float32, error)
	DetectObjects(ctx context.Context, frame []byte) ([]AIDetection, error)
	RecognizePlate(ctx context.Context, frame []byte) (string, float64, error)
}

type BaseBackend struct { mu sync.RWMutex }

func NormalizeVector(v []float32) []float32 {
	if len(v) == 0 { return v }
	var sumSquares float64
	for _, val := range v { sumSquares += float64(val) * float64(val) }
	if sumSquares == 0 { return v }
	norm := float32(math.Sqrt(sumSquares))
	res := make([]float32, len(v))
	for i, val := range v { res[i] = val / norm }
	return res
}

func inferenceUnavailable(name string) error {
	return fmt.Errorf("%w: %s model runtime is not configured", ErrInferenceModelNotConfigured, name)
}

type UnavailableBackend struct{}
func (u *UnavailableBackend) Name() string { return "Inference not configured" }
func (u *UnavailableBackend) Type() BackendType { return BackendUnavailable }
func (u *UnavailableBackend) IsAvailable() bool { return false }
func (u *UnavailableBackend) ExtractFaceEmbedding(ctx context.Context, _ []byte) ([]float32, error) {
	if err := ctx.Err(); err != nil { return nil, err }; return nil, inferenceUnavailable("unavailable")
}
func (u *UnavailableBackend) DetectObjects(ctx context.Context, _ []byte) ([]AIDetection, error) {
	if err := ctx.Err(); err != nil { return nil, err }; return nil, inferenceUnavailable("unavailable")
}
func (u *UnavailableBackend) RecognizePlate(ctx context.Context, _ []byte) (string, float64, error) {
	if err := ctx.Err(); err != nil { return "", 0, err }; return "", 0, inferenceUnavailable("unavailable")
}

// These backend descriptors remain available for future integrations. They
// are not active drivers until a real runtime and compatible models are loaded.
type CoralTPUBackend struct { BaseBackend; devicePath string }
func NewCoralTPUBackend(devicePath string) *CoralTPUBackend {
	if devicePath == "" { devicePath = "/dev/apex_0" }
	return &CoralTPUBackend{devicePath: devicePath}
}
func (c *CoralTPUBackend) Name() string { return "Coral Edge TPU (runtime not configured)" }
func (c *CoralTPUBackend) Type() BackendType { return BackendCoralTPU }
func (c *CoralTPUBackend) IsAvailable() bool { return false }
func (c *CoralTPUBackend) ExtractFaceEmbedding(ctx context.Context, _ []byte) ([]float32, error) {
	if err := ctx.Err(); err != nil { return nil, err }; return nil, inferenceUnavailable("coral")
}
func (c *CoralTPUBackend) DetectObjects(ctx context.Context, _ []byte) ([]AIDetection, error) {
	if err := ctx.Err(); err != nil { return nil, err }; return nil, inferenceUnavailable("coral")
}
func (c *CoralTPUBackend) RecognizePlate(ctx context.Context, _ []byte) (string, float64, error) {
	if err := ctx.Err(); err != nil { return "", 0, err }; return "", 0, inferenceUnavailable("coral")
}

type Hailo8Backend struct { BaseBackend; devicePath string }
func NewHailo8Backend(devicePath string) *Hailo8Backend {
	if devicePath == "" { devicePath = "/dev/hailo0" }
	return &Hailo8Backend{devicePath: devicePath}
}
func (h *Hailo8Backend) Name() string { return "Hailo-8 (runtime not configured)" }
func (h *Hailo8Backend) Type() BackendType { return BackendHailo8 }
func (h *Hailo8Backend) IsAvailable() bool { return false }
func (h *Hailo8Backend) ExtractFaceEmbedding(ctx context.Context, _ []byte) ([]float32, error) {
	if err := ctx.Err(); err != nil { return nil, err }; return nil, inferenceUnavailable("hailo")
}
func (h *Hailo8Backend) DetectObjects(ctx context.Context, _ []byte) ([]AIDetection, error) {
	if err := ctx.Err(); err != nil { return nil, err }; return nil, inferenceUnavailable("hailo")
}
func (h *Hailo8Backend) RecognizePlate(ctx context.Context, _ []byte) (string, float64, error) {
	if err := ctx.Err(); err != nil { return "", 0, err }; return "", 0, inferenceUnavailable("hailo")
}

type CPUSIMDBackend struct { BaseBackend }
func NewCPUSIMDBackend() *CPUSIMDBackend { return &CPUSIMDBackend{} }
func (s *CPUSIMDBackend) Name() string { return "CPU inference (model runtime not configured)" }
func (s *CPUSIMDBackend) Type() BackendType { return BackendCPUSIMD }
func (s *CPUSIMDBackend) IsAvailable() bool { return false }
func (s *CPUSIMDBackend) ExtractFaceEmbedding(ctx context.Context, _ []byte) ([]float32, error) {
	if err := ctx.Err(); err != nil { return nil, err }; return nil, inferenceUnavailable("cpu")
}
func (s *CPUSIMDBackend) DetectObjects(ctx context.Context, _ []byte) ([]AIDetection, error) {
	if err := ctx.Err(); err != nil { return nil, err }; return nil, inferenceUnavailable("cpu")
}
func (s *CPUSIMDBackend) RecognizePlate(ctx context.Context, _ []byte) (string, float64, error) {
	if err := ctx.Err(); err != nil { return "", 0, err }; return "", 0, inferenceUnavailable("cpu")
}

// DetectHardwareBackend fails closed. An accelerator device node or force flag
// does not prove that a usable inference runtime or model is initialized.
func DetectHardwareBackend() InferenceBackend { return &UnavailableBackend{} }
