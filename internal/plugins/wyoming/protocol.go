// Package wyoming implements the Wyoming Voice Protocol for Whisper (STT), Piper (TTS), and OpenWakeWord.
package wyoming

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Event types supported by Wyoming.
const (
	EventDescribe   = "describe"
	EventInfo       = "info"
	EventAudioStart = "audio-start"
	EventAudioChunk = "audio-chunk"
	EventAudioStop  = "audio-stop"
	EventTranscript = "transcript"
	EventSynthesize = "synthesize"
	EventDetection  = "detection"
	EventDetect     = "detect"
	EventNotDetected = "not-detected"
)

// Header represents the JSON line prefix of every Wyoming frame.
type Header struct {
	Type          string `json:"type"`
	DataLength    int    `json:"data_length,omitempty"`
	PayloadLength int    `json:"payload_length,omitempty"`
}

// Message represents a decoded Wyoming message.
type Message struct {
	Type    string         `json:"type"`
	Data    map[string]any `json:"data,omitempty"`
	Payload []byte         `json:"-"`
}

// WriteMessage serializes a Wyoming message into an io.Writer.
// Format: JSON line followed by optional Data JSON and optional Payload raw bytes.
func WriteMessage(w io.Writer, msg Message) error {
	if strings.TrimSpace(msg.Type) == "" || len(msg.Payload) > 8<<20 {
		return fmt.Errorf("wyoming: invalid message type or payload size")
	}
	var dataBytes []byte
	var err error
	if len(msg.Data) > 0 {
		dataBytes, err = json.Marshal(msg.Data)
		if err != nil {
			return fmt.Errorf("wyoming: failed to marshal data: %w", err)
		}
	}
	if len(dataBytes) > 1<<20 {
		return fmt.Errorf("wyoming: message data exceeds 1 MiB")
	}
	headerBytes, err := json.Marshal(Header{Type: msg.Type, DataLength: len(dataBytes), PayloadLength: len(msg.Payload)})
	if err != nil {
		return fmt.Errorf("wyoming: failed to marshal header: %w", err)
	}
	if _, err := io.WriteString(w, string(headerBytes)+"\n"); err != nil {
		return err
	}
	if len(dataBytes) > 0 {
		if _, err := w.Write(dataBytes); err != nil {
			return err
		}
	}
	if len(msg.Payload) > 0 {
		if _, err := w.Write(msg.Payload); err != nil {
			return err
		}
	}
	return nil
}

func ReadMessage(r *bufio.Reader) (*Message, error) {
	line, err := r.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	if len(line) > 4096 {
		return nil, fmt.Errorf("wyoming: header exceeds 4096 bytes")
	}
	var header Header
	if err := json.Unmarshal(line, &header); err != nil {
		return nil, fmt.Errorf("wyoming: invalid header: %w", err)
	}
	if strings.TrimSpace(header.Type) == "" || header.DataLength < 0 || header.DataLength > 1<<20 ||
		header.PayloadLength < 0 || header.PayloadLength > 8<<20 {
		return nil, fmt.Errorf("wyoming: invalid message type or frame lengths")
	}
	msg := &Message{Type: header.Type}
	if header.DataLength > 0 {
		dataBuf := make([]byte, header.DataLength)
		if _, err := io.ReadFull(r, dataBuf); err != nil {
			return nil, fmt.Errorf("wyoming: reading data failed: %w", err)
		}
		var data map[string]any
		if err := json.Unmarshal(dataBuf, &data); err != nil {
			return nil, fmt.Errorf("wyoming: invalid message data: %w", err)
		}
		msg.Data = data
	}
	if header.PayloadLength > 0 {
		msg.Payload = make([]byte, header.PayloadLength)
		if _, err := io.ReadFull(r, msg.Payload); err != nil {
			return nil, fmt.Errorf("wyoming: reading payload failed: %w", err)
		}
	}
	return msg, nil
}
