// Package wyoming implements the Wyoming Voice Protocol for Whisper (STT), Piper (TTS), and OpenWakeWord.
package wyoming

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
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
	var dataBytes []byte
	var err error
	if len(msg.Data) > 0 {
		dataBytes, err = json.Marshal(msg.Data)
		if err != nil {
			return fmt.Errorf("wyoming: failed to marshal data: %w", err)
		}
	}

	header := Header{
		Type:          msg.Type,
		DataLength:    len(dataBytes),
		PayloadLength: len(msg.Payload),
	}

	headerBytes, err := json.Marshal(header)
	if err != nil {
		return fmt.Errorf("wyoming: failed to marshal header: %w", err)
	}

	// Write header line
	if _, err := fmt.Fprintf(w, "%s\n", headerBytes); err != nil {
		return err
	}

	// Write data bytes if any
	if len(dataBytes) > 0 {
		if _, err := w.Write(dataBytes); err != nil {
			return err
		}
	}

	// Write payload bytes if any
	if len(msg.Payload) > 0 {
		if _, err := w.Write(msg.Payload); err != nil {
			return err
		}
	}

	return nil
}

// ReadMessage parses a single Wyoming message from a bufio.Reader.
func ReadMessage(r *bufio.Reader) (*Message, error) {
	line, err := r.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	var header Header
	if err := json.Unmarshal(line, &header); err != nil {
		return nil, fmt.Errorf("wyoming: invalid header: %w", err)
	}

	msg := &Message{
		Type: header.Type,
	}

	// Read Data JSON if present
	if header.DataLength > 0 {
		dataBuf := make([]byte, header.DataLength)
		if _, err := io.ReadFull(r, dataBuf); err != nil {
			return nil, fmt.Errorf("wyoming: reading data failed: %w", err)
		}
		var d map[string]any
		if err := json.Unmarshal(dataBuf, &d); err == nil {
			msg.Data = d
		}
	}

	// Read binary Payload if present
	if header.PayloadLength > 0 {
		msg.Payload = make([]byte, header.PayloadLength)
		if _, err := io.ReadFull(r, msg.Payload); err != nil {
			return nil, fmt.Errorf("wyoming: reading payload failed: %w", err)
		}
	}

	return msg, nil
}
