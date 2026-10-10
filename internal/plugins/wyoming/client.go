package wyoming

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"time"
)

// Config holds connection parameters for Wyoming services.
type Config struct {
	WhisperAddr string // e.g. "127.0.0.1:10300"
	PiperAddr   string // e.g. "127.0.0.1:10200"
	WakeAddr    string // e.g. "127.0.0.1:10400"
	Timeout     time.Duration
}

// Client interacts with Wyoming STT, TTS, and Wake Word microservices.
type Client struct {
	cfg Config
}

// NewClient returns a new Wyoming client.
func NewClient(cfg Config) *Client {
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}
	return &Client{cfg: cfg}
}

// Transcribe streams PCM 16kHz mono audio to Whisper and returns the recognized text.
func (c *Client) Transcribe(ctx context.Context, pcmAudio []byte) (string, error) {
	if c.cfg.WhisperAddr == "" {
		return "", fmt.Errorf("wyoming: whisper address not configured")
	}

	dialer := net.Dialer{Timeout: c.cfg.Timeout}
	conn, err := dialer.DialContext(ctx, "tcp", c.cfg.WhisperAddr)
	if err != nil {
		return "", fmt.Errorf("wyoming: failed to connect to whisper at %s: %w", c.cfg.WhisperAddr, err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// 1. Send audio-start
	startMsg := Message{
		Type: EventAudioStart,
		Data: map[string]any{
			"rate":      16000,
			"width":     2,
			"channels":  1,
			"timestamp": 0,
		},
	}
	if err := WriteMessage(conn, startMsg); err != nil {
		return "", fmt.Errorf("wyoming: failed sending audio-start: %w", err)
	}

	// 2. Send audio chunks (split into 1024-byte frames)
	chunkSize := 1024
	for i := 0; i < len(pcmAudio); i += chunkSize {
		end := i + chunkSize
		if end > len(pcmAudio) {
			end = len(pcmAudio)
		}
		chunkMsg := Message{
			Type:    EventAudioChunk,
			Payload: pcmAudio[i:end],
			Data: map[string]any{
				"rate":      16000,
				"width":     2,
				"channels":  1,
				"timestamp": i / 2,
			},
		}
		if err := WriteMessage(conn, chunkMsg); err != nil {
			return "", fmt.Errorf("wyoming: failed sending audio-chunk: %w", err)
		}
	}

	// 3. Send audio-stop
	stopMsg := Message{
		Type: EventAudioStop,
		Data: map[string]any{
			"timestamp": len(pcmAudio) / 2,
		},
	}
	if err := WriteMessage(conn, stopMsg); err != nil {
		return "", fmt.Errorf("wyoming: failed sending audio-stop: %w", err)
	}

	// 4. Wait for transcript
	for {
		msg, err := ReadMessage(reader)
		if err != nil {
			return "", fmt.Errorf("wyoming: error waiting for transcript: %w", err)
		}
		if msg.Type == EventTranscript {
			if text, ok := msg.Data["text"].(string); ok {
				return text, nil
			}
			return "", nil
		}
	}
}

// Synthesize sends text to Piper and returns synthesized PCM audio.
func (c *Client) Synthesize(ctx context.Context, text string, voice string) ([]byte, error) {
	if c.cfg.PiperAddr == "" {
		return nil, fmt.Errorf("wyoming: piper address not configured")
	}

	dialer := net.Dialer{Timeout: c.cfg.Timeout}
	conn, err := dialer.DialContext(ctx, "tcp", c.cfg.PiperAddr)
	if err != nil {
		return nil, fmt.Errorf("wyoming: failed to connect to piper at %s: %w", c.cfg.PiperAddr, err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// Send synthesize event
	synthMsg := Message{
		Type: EventSynthesize,
		Data: map[string]any{
			"text": text,
			"voice": map[string]any{
				"name": voice,
			},
		},
	}
	if err := WriteMessage(conn, synthMsg); err != nil {
		return nil, fmt.Errorf("wyoming: failed sending synthesize: %w", err)
	}

	var audioBuf bytes.Buffer

	// Read audio-chunks until audio-stop
	for {
		msg, err := ReadMessage(reader)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("wyoming: read error: %w", err)
		}
		switch msg.Type {
		case EventAudioChunk:
			audioBuf.Write(msg.Payload)
		case EventAudioStop:
			return audioBuf.Bytes(), nil
		}
	}

	return audioBuf.Bytes(), nil
}
