package assistant

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVoicePipeline_RMSCalculation(t *testing.T) {
	silence := make([]byte, 320) // 10ms of silence
	rms := CalculateRMS(silence)
	if rms != 0.0 {
		t.Errorf("expected silence RMS to be 0.0, got %f", rms)
	}

	// Audio signal with samples
	audio := make([]byte, 320)
	for i := 0; i < len(audio)-1; i += 2 {
		audio[i] = 0x00
		audio[i+1] = 0x40 // Half scale amplitude ~16384
	}
	activeRMS := CalculateRMS(audio)
	if activeRMS <= 0.1 {
		t.Errorf("expected significant RMS on active audio, got %f", activeRMS)
	}
}

func TestVoicePipeline_BufferingAndProcessing(t *testing.T) {
	// Mock Whisper server
	whisperMux := http.NewServeMux()
	whisperMux.HandleFunc("/transcribe", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"text":"accendi la luce in cucina"}`))
	})
	whisperServer := httptest.NewServer(whisperMux)
	defer whisperServer.Close()

	// Mock Piper server
	piperMux := http.NewServeMux()
	piperMux.HandleFunc("/synthesize", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/wav")
		_, _ = w.Write([]byte("RIFF1234WAVEfmt mock_audio"))
	})
	piperServer := httptest.NewServer(piperMux)
	defer piperServer.Close()

	assistantSvc := NewService(Config{}, nil, nil)
	cfg := VoicePipelineConfig{
		WhisperEndpoint: whisperServer.URL,
		PiperEndpoint:   piperServer.URL,
		Language:        "it",
	}

	pipeline := NewVoicePipeline(cfg, assistantSvc)

	// Simulate streaming audio chunks
	chunk := make([]byte, 640) // 20ms at 16kHz 16-bit mono
	rms := pipeline.AppendChunk("sat_living", chunk)
	if rms < 0 {
		t.Errorf("expected non-negative RMS, got %f", rms)
	}

	// Process speech
	result, err := pipeline.ProcessBufferedSpeech(context.Background(), "sat_living")
	if err != nil {
		t.Fatalf("failed to process buffered speech: %v", err)
	}

	if result.TranscribedText != "accendi la luce in cucina" {
		t.Errorf("expected transcribed text 'accendi la luce in cucina', got '%s'", result.TranscribedText)
	}
	if len(result.AudioBytes) == 0 {
		t.Errorf("expected non-empty audio synthesis bytes")
	}
}
