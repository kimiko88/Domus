package wyoming_test

import (
	"bufio"
	"bytes"
	"context"
	"net"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/plugins/wyoming"
)

func TestWyomingFraming(t *testing.T) {
	orig := wyoming.Message{
		Type:    wyoming.EventAudioChunk,
		Data:    map[string]any{"rate": float64(16000), "width": float64(2)},
		Payload: []byte{0x01, 0x02, 0x03, 0x04},
	}

	var buf bytes.Buffer
	if err := wyoming.WriteMessage(&buf, orig); err != nil {
		t.Fatalf("failed to write message: %v", err)
	}

	reader := bufio.NewReader(&buf)
	decoded, err := wyoming.ReadMessage(reader)
	if err != nil {
		t.Fatalf("failed to read message: %v", err)
	}

	if decoded.Type != orig.Type {
		t.Errorf("type mismatch: got %s, want %s", decoded.Type, orig.Type)
	}
	if !bytes.Equal(decoded.Payload, orig.Payload) {
		t.Errorf("payload mismatch: got %v, want %v", decoded.Payload, orig.Payload)
	}
	if decoded.Data["rate"] != float64(16000) {
		t.Errorf("data mismatch: got %v", decoded.Data)
	}
}

func TestWhisperAndPiperClients(t *testing.T) {
	// 1. Mock Whisper TCP Server
	whisperLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer whisperLn.Close()

	go func() {
		conn, err := whisperLn.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		r := bufio.NewReader(conn)
		for {
			msg, err := wyoming.ReadMessage(r)
			if err != nil {
				return
			}
			if msg.Type == wyoming.EventAudioStop {
				// Reply with transcript
				_ = wyoming.WriteMessage(conn, wyoming.Message{
					Type: wyoming.EventTranscript,
					Data: map[string]any{"text": "accendi la luce in cucina"},
				})
				return
			}
		}
	}()

	// 2. Mock Piper TCP Server
	piperLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer piperLn.Close()

	go func() {
		conn, err := piperLn.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		r := bufio.NewReader(conn)
		msg, err := wyoming.ReadMessage(r)
		if err != nil {
			return
		}
		if msg.Type == wyoming.EventSynthesize {
			// Send audio-start, chunk, and audio-stop
			_ = wyoming.WriteMessage(conn, wyoming.Message{
				Type: wyoming.EventAudioStart,
				Data: map[string]any{"rate": 22050},
			})
			_ = wyoming.WriteMessage(conn, wyoming.Message{
				Type:    wyoming.EventAudioChunk,
				Payload: []byte{0xDE, 0xAD, 0xBE, 0xEF},
			})
			_ = wyoming.WriteMessage(conn, wyoming.Message{
				Type: wyoming.EventAudioStop,
			})
		}
	}()

	client := wyoming.NewClient(wyoming.Config{
		WhisperAddr: whisperLn.Addr().String(),
		PiperAddr:   piperLn.Addr().String(),
		Timeout:     2 * time.Second,
	})

	ctx := context.Background()

	// Test Whisper STT
	text, err := client.Transcribe(ctx, []byte{0x00, 0x01, 0x02, 0x03})
	if err != nil {
		t.Fatalf("Transcribe failed: %v", err)
	}
	if text != "accendi la luce in cucina" {
		t.Errorf("unexpected transcript: got %q", text)
	}

	// Test Piper TTS
	audio, err := client.Synthesize(ctx, "Luce accesa", "it_IT-paola-medium")
	if err != nil {
		t.Fatalf("Synthesize failed: %v", err)
	}
	if !bytes.Equal(audio, []byte{0xDE, 0xAD, 0xBE, 0xEF}) {
		t.Errorf("unexpected audio synthesized: got %v", audio)
	}
}
