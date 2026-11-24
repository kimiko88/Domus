package assistant

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"
)

// AudioProcessResult contains the outcome of speech processing.
type AudioProcessResult struct {
	TranscribedText string `json:"transcribed_text"`
	ResponseText    string `json:"response_text"`
	AudioFormat     string `json:"audio_format"` // "audio/wav" or "audio/pcm"
	AudioBytes      []byte `json:"-"`            // Synthesized audio payload
	AudioBase64     string `json:"audio_base64"` // For JSON transmission
	ActionTaken     bool   `json:"action_taken"`
	DurationMs      int64  `json:"duration_ms"`
}

// VoicePipelineConfig defines endpoints and parameters for local Whisper and Piper.
type VoicePipelineConfig struct {
	WhisperEndpoint string  `json:"whisper_endpoint"`  // e.g. "http://localhost:10300" (Faster-Whisper/Wyoming)
	PiperEndpoint   string  `json:"piper_endpoint"`    // e.g. "http://localhost:10200" (Piper TTS/Wyoming)
	Language        string  `json:"language"`          // "it", "en", etc.
	VoiceModel      string  `json:"voice_model"`       // e.g. "it_IT-paola-medium"
	VADThresholdRMS float64 `json:"vad_threshold_rms"` // RMS energy threshold to consider speech (e.g. 0.015)
	SampleRate      int     `json:"sample_rate"`       // 16000 Hz standard
}

// VoicePipeline manages bounded PCM buffers, transcription and speech synthesis.
const maxSatelliteAudioBytes = 2 << 20
const maxVoiceResponseBytes = 2 << 20

type VoicePipeline struct {
	mu           sync.Mutex
	cfg          VoicePipelineConfig
	assistantSvc *Service
	httpClient   *http.Client
	audioBuffers map[string]*bytes.Buffer // satelliteID -> accumulated PCM
}

// NewVoicePipeline constructs an offline local voice pipeline.
func NewVoicePipeline(cfg VoicePipelineConfig, assistantSvc *Service) *VoicePipeline {
	if cfg.WhisperEndpoint == "" {
		cfg.WhisperEndpoint = "http://localhost:10300"
	}
	if cfg.PiperEndpoint == "" {
		cfg.PiperEndpoint = "http://localhost:10200"
	}
	if cfg.Language == "" {
		cfg.Language = "it"
	}
	if cfg.VoiceModel == "" {
		cfg.VoiceModel = "it_IT-paola-medium"
	}
	if cfg.VADThresholdRMS <= 0 {
		cfg.VADThresholdRMS = 0.015
	}
	if cfg.SampleRate <= 0 {
		cfg.SampleRate = 16000
	}

	return &VoicePipeline{
		cfg:          cfg,
		assistantSvc: assistantSvc,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		audioBuffers: make(map[string]*bytes.Buffer),
	}
}

// CalculateRMS computes Root Mean Square volume of 16-bit signed PCM audio.
func CalculateRMS(pcmData []byte) float64 {
	if len(pcmData) < 2 {
		return 0
	}
	sampleCount := len(pcmData) / 2
	var sumSquares float64

	for i := 0; i < len(pcmData)-1; i += 2 {
		val := int16(binary.LittleEndian.Uint16(pcmData[i : i+2]))
		norm := float64(val) / 32768.0
		sumSquares += norm * norm
	}

	return math.Sqrt(sumSquares / float64(sampleCount))
}

// AppendChunk buffers incoming audio chunks from a satellite.
func (p *VoicePipeline) AppendChunk(satelliteID string, pcmChunk []byte) float64 {
	level, _ := p.AppendChunkChecked(satelliteID, pcmChunk)
	return level
}

// AppendChunkChecked appends signed 16-bit little-endian mono PCM with a per-satellite cap.
func (p *VoicePipeline) AppendChunkChecked(satelliteID string, pcmChunk []byte) (float64, error) {
	if strings.TrimSpace(satelliteID) == "" || len(pcmChunk) == 0 || len(pcmChunk)%2 != 0 {
		return 0, fmt.Errorf("voice_pipeline: invalid satellite ID or PCM chunk")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	buf, exists := p.audioBuffers[satelliteID]
	if !exists {
		buf = bytes.NewBuffer(make([]byte, 0, 16000*2*3))
		p.audioBuffers[satelliteID] = buf
	}
	if len(pcmChunk) > maxSatelliteAudioBytes-buf.Len() {
		return CalculateRMS(pcmChunk), fmt.Errorf("voice_pipeline: audio buffer limit exceeded")
	}
	buf.Write(pcmChunk)
	return CalculateRMS(pcmChunk), nil
}

// ClearBuffer clears the audio buffer for a given satellite.
func (p *VoicePipeline) ClearBuffer(satelliteID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.audioBuffers, satelliteID)
}

// ProcessBufferedSpeech transcribes accumulated audio, resolves the intent, and synthesizes the reply.
func (p *VoicePipeline) ProcessBufferedSpeech(ctx context.Context, satelliteID string) (*AudioProcessResult, error) {
	p.mu.Lock()
	buf, exists := p.audioBuffers[satelliteID]
	if !exists || buf.Len() == 0 {
		p.mu.Unlock()
		return nil, fmt.Errorf("voice_pipeline: empty audio buffer for %s", satelliteID)
	}
	pcmBytes := make([]byte, buf.Len())
	copy(pcmBytes, buf.Bytes())
	delete(p.audioBuffers, satelliteID)
	p.mu.Unlock()

	startTime := time.Now()

	// Do not execute a guessed command if speech recognition is unavailable.
	transcription, err := p.TranscribeAudio(ctx, pcmBytes)
	if err != nil { return nil, fmt.Errorf("voice_pipeline: transcription failed: %w", err) }
	if strings.TrimSpace(transcription) == "" { return nil, fmt.Errorf("voice_pipeline: no speech recognized") }
	return p.ProcessTextPrompt(ctx, transcription, time.Since(startTime))
}

// ProcessTextPrompt sends a typed or transcribed prompt through Domus Assistant
// and synthesizes a WAV reply when the configured Piper server is reachable.
func (p *VoicePipeline) ProcessTextPrompt(ctx context.Context, prompt string, elapsed ...time.Duration) (*AudioProcessResult, error) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" || len(prompt) > 4096 { return nil, fmt.Errorf("voice_pipeline: text prompt is empty or too long") }
	if p.assistantSvc == nil { return nil, fmt.Errorf("voice_pipeline: assistant service is not configured") }
	start := time.Now()
	chatResp, err := p.assistantSvc.ProcessPrompt(ctx, prompt)
	if err != nil { return nil, fmt.Errorf("voice_pipeline: assistant processing failed: %w", err) }
	if chatResp == nil { return nil, fmt.Errorf("voice_pipeline: assistant returned an empty response") }
	responseText := strings.TrimSpace(chatResp.ResponseText)
	if responseText == "" { responseText = "Non ho una risposta disponibile." }
	audioOut, synthErr := p.SynthesizeSpeech(ctx, responseText)
	if synthErr != nil { audioOut = nil }
	duration := time.Since(start)
	if len(elapsed) > 0 && elapsed[0] > duration { duration = elapsed[0] }
	audioFormat := ""
	if len(audioOut) > 0 { audioFormat = "audio/wav" }
	return &AudioProcessResult{
		TranscribedText: prompt,
		ResponseText: responseText,
		AudioFormat: audioFormat,
		AudioBytes: audioOut,
		ActionTaken: len(chatResp.ActionsExecuted) > 0,
		DurationMs: duration.Milliseconds(),
	}, nil
}

// TranscribeAudio calls local Whisper server or decodes speech.
func (p *VoicePipeline) TranscribeAudio(ctx context.Context, pcmData []byte) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.WhisperEndpoint+"/transcribe", bytes.NewReader(pcmData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("whisper returned status %d", resp.StatusCode)
	}

	var res struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}
	return strings.TrimSpace(res.Text), nil
}

// SynthesizeSpeech calls local Piper TTS server to generate WAV audio for the response.
func (p *VoicePipeline) SynthesizeSpeech(ctx context.Context, text string) ([]byte, error) {
	payload, _ := json.Marshal(map[string]string{
		"text":  text,
		"voice": p.cfg.VoiceModel,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.PiperEndpoint+"/synthesize", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("piper returned status %d", resp.StatusCode)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, io.LimitReader(resp.Body, maxVoiceResponseBytes+1))
	if err != nil { return nil, err }
	if buf.Len() > maxVoiceResponseBytes { return nil, fmt.Errorf("piper response exceeds %d bytes", maxVoiceResponseBytes) }
	return buf.Bytes(), nil
}

// generateBeepWAV generates a small in-memory valid WAV sine wave for audio feedback.
func (p *VoicePipeline) generateBeepWAV(freqHz float64, durationMs int) []byte {
	sampleRate := 16000
	numSamples := sampleRate * durationMs / 1000
	pcmData := make([]byte, numSamples*2)

	for i := 0; i < numSamples; i++ {
		t := float64(i) / float64(sampleRate)
		val := int16(math.Sin(2*math.Pi*freqHz*t) * 16000.0)
		binary.LittleEndian.PutUint16(pcmData[i*2:i*2+2], uint16(val))
	}

	wavHeader := make([]byte, 44)
	copy(wavHeader[0:4], "RIFF")
	binary.LittleEndian.PutUint32(wavHeader[4:8], uint32(36+len(pcmData)))
	copy(wavHeader[8:12], "WAVE")
	copy(wavHeader[12:16], "fmt ")
	binary.LittleEndian.PutUint32(wavHeader[16:20], 16) // Subchunk1Size
	binary.LittleEndian.PutUint16(wavHeader[20:22], 1)  // PCM
	binary.LittleEndian.PutUint16(wavHeader[22:24], 1)  // Mono
	binary.LittleEndian.PutUint32(wavHeader[24:28], uint32(sampleRate))
	binary.LittleEndian.PutUint32(wavHeader[28:32], uint32(sampleRate*2))
	binary.LittleEndian.PutUint16(wavHeader[32:34], 2)  // BlockAlign
	binary.LittleEndian.PutUint16(wavHeader[34:36], 16) // BitsPerSample
	copy(wavHeader[36:40], "data")
	binary.LittleEndian.PutUint32(wavHeader[40:44], uint32(len(pcmData)))

	return append(wavHeader, pcmData...)
}
