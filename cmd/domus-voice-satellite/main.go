package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/domus-hub/domus/internal/plugins/voice_satellite"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "domus-voice-satellite:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := voice_satellite.Config{
		SatelliteID: os.Getenv("DOMUS_VOICE_SATELLITE_ID"),
		ServerURL: os.Getenv("DOMUS_VOICE_SATELLITE_SERVER_URL"),
		BearerToken: os.Getenv("DOMUS_VOICE_SATELLITE_TOKEN"),
		WakeAddress: envOr("DOMUS_VOICE_SATELLITE_WAKE_ADDR", "127.0.0.1:10400"),
		WakeWords: splitCSV(envOr("DOMUS_VOICE_SATELLITE_WAKE_WORDS", "hey_jarvis")),
		AudioDevice: envOr("DOMUS_VOICE_SATELLITE_AUDIO_DEVICE", "default"),
		SpeakerDevice: os.Getenv("DOMUS_VOICE_SATELLITE_SPEAKER_DEVICE"),
		FrameDuration: durationEnv("DOMUS_VOICE_SATELLITE_FRAME_DURATION", 100*time.Millisecond),
		SilenceDuration: durationEnv("DOMUS_VOICE_SATELLITE_SILENCE_DURATION", 1200*time.Millisecond),
		MaxCommandDuration: durationEnv("DOMUS_VOICE_SATELLITE_MAX_COMMAND_DURATION", 10*time.Second),
		HTTPTimeout: durationEnv("DOMUS_VOICE_SATELLITE_HTTP_TIMEOUT", 30*time.Second),
		SpeechThreshold: floatEnv("DOMUS_VOICE_SATELLITE_RMS_THRESHOLD", 0.015),
	}
	adapter, err := voice_satellite.NewAdapter(cfg, slog.Default())
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := adapter.Start(ctx); err != nil {
		return err
	}
	<-ctx.Done()
	stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return adapter.Stop(stopCtx)
}

func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" { return value }
	return fallback
}

func splitCSV(value string) []string {
	var result []string
	for _, part := range strings.Split(value, ",") {
		if part = strings.TrimSpace(part); part != "" { result = append(result, part) }
	}
	return result
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" { return fallback }
	parsed, err := time.ParseDuration(value)
	if err != nil { slog.Warn("invalid duration environment variable; using default", "key", key, "error", err); return fallback }
	return parsed
}

func floatEnv(key string, fallback float64) float64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" { return fallback }
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil { slog.Warn("invalid float environment variable; using default", "key", key, "error", err); return fallback }
	return parsed
}
