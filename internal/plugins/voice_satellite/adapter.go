// Package voice_satellite implements a Linux microphone/speaker voice satellite.
package voice_satellite

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"sync"
	"time"

	appassistant "github.com/domus-hub/domus/internal/application/assistant"
	"github.com/domus-hub/domus/internal/plugins/wyoming"
)

const (
	defaultSampleRate = 16000
	maxCommandAudioBytes = 2 << 20
	maxVoiceResponseBytes = 2 << 20
)

// Config controls ALSA capture, Wyoming wake detection, and the authenticated
// Domus voice gateway. BearerToken is deliberately never logged.
type Config struct {
	SatelliteID       string
	ServerURL         string
	BearerToken       string
	WakeAddress       string
	WakeWords         []string
	AudioDevice       string
	SpeakerDevice     string
	SampleRate        int
	FrameDuration     time.Duration
	SpeechThreshold   float64
	SilenceDuration   time.Duration
	MaxCommandDuration time.Duration
	HTTPTimeout       time.Duration
}

type VoiceSatelliteAdapter struct {
	cfg        Config
	endpoint   string
	client     *http.Client
	logger     *slog.Logger
	mu         sync.RWMutex
	running    bool
	lastError  string
	cancel     context.CancelFunc
	done       chan struct{}
}

var _ interface {
	ID() string
	Type() string
	Start(context.Context) error
	Stop(context.Context) error
	HealthCheck(context.Context) error
} = (*VoiceSatelliteAdapter)(nil)

func NewAdapter(cfg Config, logger *slog.Logger) (*VoiceSatelliteAdapter, error) {
	cfg.SatelliteID = strings.TrimSpace(cfg.SatelliteID)
	cfg.BearerToken = strings.TrimSpace(cfg.BearerToken)
	cfg.ServerURL = strings.TrimSpace(cfg.ServerURL)
	cfg.WakeAddress = strings.TrimSpace(cfg.WakeAddress)
	if !validSatelliteID(cfg.SatelliteID) {
		return nil, errors.New("voice satellite: satellite ID must contain 1-64 letters, digits, '_' or '-'")
	}
	u, err := url.Parse(cfg.ServerURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, errors.New("voice satellite: server URL must be an absolute HTTP(S) URL without credentials, query or fragment")
	}
	if cfg.BearerToken == "" || strings.ContainsAny(cfg.BearerToken, "\r\n") {
		return nil, errors.New("voice satellite: a bearer token is required")
	}
	if _, _, err := netSplitHostPort(cfg.WakeAddress); err != nil {
		return nil, fmt.Errorf("voice satellite: invalid Wyoming wake address: %w", err)
	}
	if cfg.AudioDevice == "" {
		cfg.AudioDevice = "default"
	}
	if cfg.SampleRate == 0 {
		cfg.SampleRate = defaultSampleRate
	}
	if cfg.SampleRate != defaultSampleRate {
		return nil, fmt.Errorf("voice satellite: only %d Hz PCM is supported by the Domus gateway", defaultSampleRate)
	}
	if cfg.FrameDuration == 0 {
		cfg.FrameDuration = 100 * time.Millisecond
	}
	if cfg.FrameDuration < 20*time.Millisecond || cfg.FrameDuration > 500*time.Millisecond {
		return nil, errors.New("voice satellite: frame duration must be between 20ms and 500ms")
	}
	if cfg.SpeechThreshold <= 0 {
		cfg.SpeechThreshold = 0.015
	}
	if cfg.SpeechThreshold > 1 {
		return nil, errors.New("voice satellite: speech threshold must be in the range (0, 1]")
	}
	if cfg.SilenceDuration == 0 {
		cfg.SilenceDuration = 1200 * time.Millisecond
	}
	if cfg.SilenceDuration < 300*time.Millisecond || cfg.SilenceDuration > 5*time.Second {
		return nil, errors.New("voice satellite: silence duration must be between 300ms and 5s")
	}
	if cfg.MaxCommandDuration == 0 {
		cfg.MaxCommandDuration = 10 * time.Second
	}
	if cfg.MaxCommandDuration < time.Second || cfg.MaxCommandDuration > 30*time.Second {
		return nil, errors.New("voice satellite: max command duration must be between 1s and 30s")
	}
	if cfg.HTTPTimeout == 0 {
		cfg.HTTPTimeout = 30 * time.Second
	}
	if cfg.HTTPTimeout < time.Second || cfg.HTTPTimeout > 2*time.Minute {
		return nil, errors.New("voice satellite: HTTP timeout must be between 1s and 2m")
	}
	if len(cfg.WakeWords) == 0 {
		cfg.WakeWords = []string{"hey_jarvis"}
	}
	if len(cfg.WakeWords) > 8 {
		return nil, errors.New("voice satellite: at most 8 wake-word models can be selected")
	}
	for i, word := range cfg.WakeWords {
		cfg.WakeWords[i] = strings.TrimSpace(word)
		if cfg.WakeWords[i] == "" || len(cfg.WakeWords[i]) > 96 {
			return nil, errors.New("voice satellite: wake-word model names must be 1-96 characters")
		}
	}
	if logger == nil {
		logger = slog.Default()
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/api/v1/voice/process"
	return &VoiceSatelliteAdapter{
		cfg: cfg,
		endpoint: u.String(),
		client: &http.Client{Timeout: cfg.HTTPTimeout},
		logger: logger,
	}, nil
}

func validSatelliteID(value string) bool {
	if len(value) < 1 || len(value) > 64 {
		return false
	}
	for _, r := range value {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_' || r == '-') {
			return false
		}
	}
	return true
}

// netSplitHostPort avoids accepting arbitrary Wyoming URI schemes. Wyoming is
// an unauthenticated TCP protocol and should only be used on a trusted LAN.
func netSplitHostPort(address string) (string, string, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return "", "", err
	}
	if host == "" || port == "" {
		return "", "", errors.New("host and port are required")
	}
	return host, port, nil
}

func (a *VoiceSatelliteAdapter) ID() string   { return a.cfg.SatelliteID }
func (a *VoiceSatelliteAdapter) Type() string { return "voice_satellite" }

func (a *VoiceSatelliteAdapter) Start(parent context.Context) error {
	if _, err := exec.LookPath("arecord"); err != nil {
		return fmt.Errorf("voice satellite: arecord is required (install alsa-utils): %w", err)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.running {
		return nil
	}
	ctx, cancel := context.WithCancel(parent)
	a.cancel = cancel
	a.done = make(chan struct{})
	a.running = true
	go a.run(ctx, a.done)
	a.logger.Info("voice satellite started", "satellite_id", a.cfg.SatelliteID, "audio_device", a.cfg.AudioDevice, "wake_address", a.cfg.WakeAddress)
	return nil
}

func (a *VoiceSatelliteAdapter) Stop(ctx context.Context) error {
	a.mu.Lock()
	cancel := a.cancel
	done := a.done
	a.mu.Unlock()
	if cancel == nil {
		return nil
	}
	cancel()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (a *VoiceSatelliteAdapter) HealthCheck(ctx context.Context) error {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if !a.running {
		return errors.New("voice satellite: stopped")
	}
	if a.lastError != "" {
		return errors.New(a.lastError)
	}
	return nil
}

func (a *VoiceSatelliteAdapter) setError(err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err == nil {
		a.lastError = ""
		return
	}
	a.lastError = err.Error()
}

func (a *VoiceSatelliteAdapter) run(ctx context.Context, done chan struct{}) {
	defer close(done)
	defer func() {
		a.mu.Lock()
		a.running = false
		a.cancel = nil
		a.mu.Unlock()
	}()
	backoff := time.Second
	for ctx.Err() == nil {
		cmd := exec.CommandContext(ctx, "arecord", "-q", "-D", a.cfg.AudioDevice, "-t", "raw", "-f", "S16_LE", "-r", fmt.Sprint(a.cfg.SampleRate), "-c", "1")
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			a.setError(err)
			return
		}
		cmd.Stderr = io.Discard
		if err := cmd.Start(); err != nil {
			a.setError(fmt.Errorf("start arecord: %w", err))
			if !sleepContext(ctx, backoff) { return }
			backoff = min(backoff*2, 30*time.Second)
			continue
		}
		waitDone := make(chan error, 1)
		go func() { waitDone <- cmd.Wait() }()
		a.setError(nil)
		err = a.listen(ctx, stdout)
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		select {
		case <-waitDone:
		case <-time.After(2 * time.Second):
		}
		if ctx.Err() != nil {
			return
		}
		if err == nil {
			err = errors.New("audio capture ended unexpectedly")
		}
		a.setError(err)
		a.logger.Warn("voice satellite audio loop restarting", "satellite_id", a.cfg.SatelliteID, "error", err, "retry_in", backoff)
		if !sleepContext(ctx, backoff) { return }
		backoff = min(backoff*2, 30*time.Second)
	}
}

func (a *VoiceSatelliteAdapter) listen(ctx context.Context, audio io.Reader) error {
	wakeClient := wyoming.NewClient(wyoming.Config{WakeAddr: a.cfg.WakeAddress, Timeout: 5 * time.Second})
	frameBytes := int(float64(a.cfg.SampleRate) * a.cfg.FrameDuration.Seconds() * 2)
	frame := make([]byte, frameBytes)
	var timestampMS int64
	for ctx.Err() == nil {
		session, err := wakeClient.StartWakeSession(ctx, a.cfg.WakeWords, a.cfg.SampleRate)
		if err != nil {
			a.setError(err)
			if !sleepContext(ctx, 2*time.Second) { return ctx.Err() }
			continue
		}
		a.setError(nil)
		detected := false
		sessionFailed := false
		for !detected && !sessionFailed && ctx.Err() == nil {
			if _, err := io.ReadFull(audio, frame); err != nil {
				_ = session.Close()
				return fmt.Errorf("read microphone PCM: %w", err)
			}
			if err := session.WriteAudioChunk(frame, timestampMS); err != nil {
				sessionFailed = true
				continue
			}
			timestampMS += a.cfg.FrameDuration.Milliseconds()
			select {
			case event, ok := <-session.Events():
				if !ok {
					sessionFailed = true
					continue
				}
				if event.Err != nil {
					if !errors.Is(event.Err, wyoming.ErrWakeNotDetected) {
						a.logger.Warn("wake-word service stream error", "error", event.Err)
					}
					sessionFailed = true
					continue
				}
				if event.Name != "" {
					detected = true
					a.logger.Info("wake word detected", "satellite_id", a.cfg.SatelliteID, "model", event.Name)
				}
			default:
			}
		}
		_ = session.Close()
		if ctx.Err() != nil { return ctx.Err() }
		if sessionFailed {
			a.setError(errors.New("wake-word stream disconnected"))
			if !sleepContext(ctx, time.Second) { return ctx.Err() }
			continue
		}
		if !detected { continue }

		pcm, speech, err := a.captureCommand(ctx, audio, frameBytes)
		if err != nil {
			return err
		}
		if !speech {
			a.logger.Debug("wake word received without a speech command", "satellite_id", a.cfg.SatelliteID)
			continue
		}
		if err := a.processCommand(ctx, pcm); err != nil {
			a.logger.Warn("voice command processing failed", "satellite_id", a.cfg.SatelliteID, "error", err)
			a.setError(err)
			continue
		}
		a.setError(nil)
	}
	return ctx.Err()
}

func (a *VoiceSatelliteAdapter) captureCommand(ctx context.Context, audio io.Reader, frameBytes int) ([]byte, bool, error) {
	maxFrames := int(a.cfg.MaxCommandDuration / a.cfg.FrameDuration)
	silenceFrames := int(a.cfg.SilenceDuration / a.cfg.FrameDuration)
	noSpeechTimeoutFrames := int(3 * time.Second / a.cfg.FrameDuration)
	if silenceFrames < 1 { silenceFrames = 1 }
	frame := make([]byte, frameBytes)
	buffer := bytes.NewBuffer(make([]byte, 0, min(maxFrames*frameBytes, maxCommandAudioBytes)))
	speechDetected := false
	silenceCount := 0
	for index := 0; index < maxFrames; index++ {
		if ctx.Err() != nil { return nil, false, ctx.Err() }
		if _, err := io.ReadFull(audio, frame); err != nil {
			return nil, false, fmt.Errorf("capture voice command: %w", err)
		}
		if buffer.Len()+len(frame) > maxCommandAudioBytes {
			return nil, false, errors.New("voice satellite: command audio exceeds 2 MiB limit")
		}
		_, _ = buffer.Write(frame)
		level := appassistant.CalculateRMS(frame)
		if level >= a.cfg.SpeechThreshold {
			speechDetected = true
			silenceCount = 0
		} else if speechDetected {
			silenceCount++
		}
		if speechDetected && silenceCount >= silenceFrames {
			break
		}
		if !speechDetected && index+1 >= noSpeechTimeoutFrames {
			return nil, false, nil
		}
	}
	if !speechDetected { return nil, false, nil }
	return buffer.Bytes(), true, nil
}

func (a *VoiceSatelliteAdapter) processCommand(ctx context.Context, pcm []byte) error {
	if len(pcm) == 0 || len(pcm)%2 != 0 || len(pcm) > maxCommandAudioBytes {
		return errors.New("voice satellite: captured PCM is empty, malformed or too large")
	}
	payload := struct {
		SatelliteID string `json:"satellite_id"`
		AudioBase64 string `json:"audio_base64"`
	}{
		SatelliteID: a.cfg.SatelliteID,
		AudioBase64: base64.StdEncoding.EncodeToString(pcm),
	}
	body, err := json.Marshal(payload)
	if err != nil { return fmt.Errorf("encode voice payload: %w", err) }
	requestCtx, cancel := context.WithTimeout(ctx, a.cfg.HTTPTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, a.endpoint, bytes.NewReader(body))
	if err != nil { return err }
	req.Header.Set("Authorization", "Bearer "+a.cfg.BearerToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := a.client.Do(req)
	if err != nil { return fmt.Errorf("call Domus voice gateway: %w", err) }
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("Domus voice gateway returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(message)))
	}
	var result struct {
		ResponseText string `json:"response_text"`
		AudioFormat string `json:"audio_format"`
		AudioBase64 string `json:"audio_base64"`
	}
	decoder := json.NewDecoder(io.LimitReader(resp.Body, maxVoiceResponseBytes*2))
	if err := decoder.Decode(&result); err != nil {
		return fmt.Errorf("decode Domus voice response: %w", err)
	}
	if result.AudioBase64 == "" {
		a.logger.Info("voice command completed without speech output", "satellite_id", a.cfg.SatelliteID, "has_text_response", strings.TrimSpace(result.ResponseText) != "")
		return nil
	}
	audio, err := base64.StdEncoding.DecodeString(result.AudioBase64)
	if err != nil || len(audio) == 0 || len(audio) > maxVoiceResponseBytes {
		return errors.New("voice satellite: invalid or oversized synthesized audio")
	}
	if result.AudioFormat != "audio/wav" || len(audio) < 44 || string(audio[:4]) != "RIFF" || string(audio[8:12]) != "WAVE" {
		return errors.New("voice satellite: expected a valid WAV speech response")
	}
	return a.playAudio(ctx, audio)
}

func (a *VoiceSatelliteAdapter) playAudio(ctx context.Context, wav []byte) error {
	if _, err := exec.LookPath("aplay"); err != nil {
		a.logger.Warn("aplay is not installed; voice response cannot be played", "install", "alsa-utils")
		return nil
	}
	args := []string{"-q"}
	if a.cfg.SpeakerDevice != "" {
		args = append(args, "-D", a.cfg.SpeakerDevice)
	}
	args = append(args, "-")
	cmd := exec.CommandContext(ctx, "aplay", args...)
	stdin, err := cmd.StdinPipe()
	if err != nil { return err }
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil { return fmt.Errorf("start aplay: %w", err) }
	_, copyErr := io.Copy(stdin, bytes.NewReader(wav))
	closeErr := stdin.Close()
	waitErr := cmd.Wait()
	if copyErr != nil { return copyErr }
	if closeErr != nil { return closeErr }
	if waitErr != nil { return fmt.Errorf("play voice response: %w", waitErr) }
	return nil
}

func sleepContext(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
