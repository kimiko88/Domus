package wyoming

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

type Config struct {
	WhisperAddr string
	PiperAddr   string
	WakeAddr    string
	Timeout     time.Duration
	Language    string
}

type AudioFormat struct {
	SampleRate  int
	SampleWidth int
	Channels    int
}

type Client struct {
	cfg Config
}

const (
	maxWyomingAudioBytes = 8 << 20
	maxWyomingTextBytes  = 8192
)

func NewClient(cfg Config) *Client {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	return &Client{cfg: cfg}
}

func (c *Client) dial(ctx context.Context, address string) (net.Conn, func(), error) {
	if strings.TrimSpace(address) == "" {
		return nil, nil, errors.New("wyoming: service address is not configured")
	}
	conn, err := (&net.Dialer{Timeout: c.cfg.Timeout}).DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, nil, fmt.Errorf("wyoming: connect to %s: %w", address, err)
	}
	deadline := time.Now().Add(c.cfg.Timeout)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	if err := conn.SetDeadline(deadline); err != nil {
		_ = conn.Close()
		return nil, nil, err
	}
	stopCancel := context.AfterFunc(ctx, func() { _ = conn.Close() })
	cleanup := func() {
		stopCancel()
		_ = conn.Close()
	}
	return conn, cleanup, nil
}

// Transcribe streams signed 16-bit little-endian mono PCM at 16 kHz to Whisper.
func (c *Client) Transcribe(ctx context.Context, pcmAudio []byte) (string, error) {
	if len(pcmAudio) == 0 || len(pcmAudio)%2 != 0 || len(pcmAudio) > maxWyomingAudioBytes {
		return "", fmt.Errorf("wyoming: audio must be non-empty, even-length 16-bit PCM up to %d bytes", maxWyomingAudioBytes)
	}
	conn, cleanup, err := c.dial(ctx, c.cfg.WhisperAddr)
	if err != nil {
		return "", err
	}
	defer cleanup()

	if err := WriteMessage(conn, Message{
		Type: EventAudioStart,
		Data: map[string]any{"rate": 16000, "width": 2, "channels": 1, "timestamp": 0},
	}); err != nil {
		return "", fmt.Errorf("wyoming: send audio-start: %w", err)
	}
	const chunkSize = 1024
	for offset := 0; offset < len(pcmAudio); offset += chunkSize {
		end := min(offset+chunkSize, len(pcmAudio))
		chunk := Message{
			Type: EventAudioChunk,
			Data: map[string]any{"rate": 16000, "width": 2, "channels": 1, "timestamp": offset / 2},
			Payload: pcmAudio[offset:end],
		}
		if err := WriteMessage(conn, chunk); err != nil {
			return "", fmt.Errorf("wyoming: send audio-chunk: %w", err)
		}
	}
	if err := WriteMessage(conn, Message{
		Type: EventAudioStop,
		Data: map[string]any{"timestamp": len(pcmAudio) / 2},
	}); err != nil {
		return "", fmt.Errorf("wyoming: send audio-stop: %w", err)
	}

	reader := bufio.NewReader(conn)
	for {
		msg, err := ReadMessage(reader)
		if err != nil {
			return "", fmt.Errorf("wyoming: wait for transcript: %w", err)
		}
		switch msg.Type {
		case EventTranscript:
			transcript, _ := msg.Data["text"].(string)
			transcript = strings.TrimSpace(transcript)
			if len(transcript) > maxWyomingTextBytes {
				return "", errors.New("wyoming: transcript exceeds maximum length")
			}
			return transcript, nil
		case "error":
			message, _ := msg.Data["text"].(string)
			if message == "" {
				message = "Whisper reported a transcription error"
			}
			return "", errors.New("wyoming: " + message)
		}
	}
}

func (c *Client) Synthesize(ctx context.Context, text, voice string) ([]byte, error) {
	pcm, _, err := c.SynthesizeWithFormat(ctx, text, voice)
	return pcm, err
}

// SynthesizeWithFormat retains the AudioStart metadata and bounds the returned audio.
func (c *Client) SynthesizeWithFormat(ctx context.Context, text, voice string) ([]byte, AudioFormat, error) {
	text = strings.TrimSpace(text)
	if text == "" || len(text) > maxWyomingTextBytes {
		return nil, AudioFormat{}, errors.New("wyoming: synthesis text is empty or too long")
	}
	conn, cleanup, err := c.dial(ctx, c.cfg.PiperAddr)
	if err != nil {
		return nil, AudioFormat{}, err
	}
	defer cleanup()

	data := map[string]any{"text": text}
	if strings.TrimSpace(voice) != "" {
		data["voice"] = map[string]any{"name": voice}
	}
	if err := WriteMessage(conn, Message{Type: EventSynthesize, Data: data}); err != nil {
		return nil, AudioFormat{}, fmt.Errorf("wyoming: send synthesize: %w", err)
	}

	reader := bufio.NewReader(conn)
	format := AudioFormat{SampleRate: 22050, SampleWidth: 2, Channels: 1}
	started := false
	var audio bytes.Buffer
	for {
		msg, err := ReadMessage(reader)
		if err != nil {
			return nil, AudioFormat{}, fmt.Errorf("wyoming: read synthesis response: %w", err)
		}
		switch msg.Type {
		case EventAudioStart:
			started = true
			if rate, ok := msg.Data["rate"].(float64); ok {
				format.SampleRate = int(rate)
			}
			if width, ok := msg.Data["width"].(float64); ok {
				format.SampleWidth = int(width)
			}
			if channels, ok := msg.Data["channels"].(float64); ok {
				format.Channels = int(channels)
			}
			if format.SampleRate < 8000 || format.SampleRate > 96000 ||
				format.SampleWidth != 2 || format.Channels < 1 || format.Channels > 2 {
				return nil, AudioFormat{}, fmt.Errorf("wyoming: unsupported Piper audio format: %+v", format)
			}
		case EventAudioChunk:
			if !started {
				return nil, AudioFormat{}, errors.New("wyoming: audio-chunk received before audio-start")
			}
			if len(msg.Payload) == 0 || audio.Len()+len(msg.Payload) > maxWyomingAudioBytes {
				return nil, AudioFormat{}, fmt.Errorf("wyoming: synthesized audio exceeds %d bytes or is empty", maxWyomingAudioBytes)
			}
			if _, err := audio.Write(msg.Payload); err != nil {
				return nil, AudioFormat{}, err
			}
		case EventAudioStop:
			if !started || audio.Len() == 0 {
				return nil, AudioFormat{}, errors.New("wyoming: Piper returned an empty audio stream")
			}
			align := format.SampleWidth * format.Channels
			if audio.Len()%align != 0 {
				return nil, AudioFormat{}, errors.New("wyoming: Piper returned an incomplete PCM sample frame")
			}
			return audio.Bytes(), format, nil
		case "error":
			message, _ := msg.Data["text"].(string)
			if message == "" {
				message = "Piper reported a synthesis error"
			}
			return nil, AudioFormat{}, errors.New("wyoming: " + message)
		}
	}
}

var _ io.Reader = (*bytes.Reader)(nil)
