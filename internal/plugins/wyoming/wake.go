package wyoming

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

var ErrWakeNotDetected = errors.New("wyoming: wake word not detected")

type WakeEvent struct {
	Name        string
	TimestampMS int64
	Err         error
}

type WakeSession struct {
	conn      net.Conn
	events    chan WakeEvent
	closed    chan struct{}
	writeMu   sync.Mutex
	closeOnce sync.Once
	rate      int
}

func (s *WakeSession) Events() <-chan WakeEvent { return s.events }

// WriteAudioChunk sends one signed 16-bit little-endian mono PCM frame.
func (s *WakeSession) WriteAudioChunk(pcm []byte, timestampMS int64) error {
	if s == nil || s.conn == nil {
		return errors.New("wyoming: wake session is not initialized")
	}
	if len(pcm) == 0 || len(pcm)%2 != 0 || len(pcm) > 64<<10 {
		return errors.New("wyoming: wake audio frame must be non-empty even-length PCM up to 64 KiB")
	}
	select {
	case <-s.closed:
		return net.ErrClosed
	default:
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	select {
	case <-s.closed:
		return net.ErrClosed
	default:
	}
	if err := s.conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return err
	}
	err := WriteMessage(s.conn, Message{
		Type: EventAudioChunk,
		Data: map[string]any{"rate": s.rate, "width": 2, "channels": 1, "timestamp": timestampMS},
		Payload: pcm,
	})
	_ = s.conn.SetWriteDeadline(time.Time{})
	if err != nil {
		return fmt.Errorf("wyoming: send wake audio chunk: %w", err)
	}
	return nil
}

// Close ends the stream and releases its TCP connection.
func (s *WakeSession) Close() error {
	if s == nil || s.conn == nil {
		return nil
	}
	var result error
	s.closeOnce.Do(func() {
		s.writeMu.Lock()
		select {
		case <-s.closed:
		default:
			_ = s.conn.SetWriteDeadline(time.Now().Add(time.Second))
			result = WriteMessage(s.conn, Message{Type: EventAudioStop, Data: map[string]any{"timestamp": 0}})
		}
		s.writeMu.Unlock()
		close(s.closed)
		_ = s.conn.Close()
	})
	return result
}

func (c *Client) StartWakeSession(ctx context.Context, names []string, sampleRate int) (*WakeSession, error) {
	if strings.TrimSpace(c.cfg.WakeAddr) == "" {
		return nil, errors.New("wyoming: wake address not configured")
	}
	if sampleRate <= 0 || sampleRate > 96000 {
		return nil, errors.New("wyoming: invalid wake sample rate")
	}
	dialer := net.Dialer{Timeout: c.cfg.Timeout}
	conn, err := dialer.DialContext(ctx, "tcp", c.cfg.WakeAddr)
	if err != nil {
		return nil, fmt.Errorf("wyoming: connect to wake service %s: %w", c.cfg.WakeAddr, err)
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetWriteDeadline(deadline)
	} else {
		_ = conn.SetWriteDeadline(time.Now().Add(c.cfg.Timeout))
	}
	detectData := map[string]any{}
	if len(names) > 0 {
		detectData["names"] = append([]string(nil), names...)
	}
	if err := WriteMessage(conn, Message{Type: EventDetect, Data: detectData}); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("wyoming: request wake detection: %w", err)
	}
	if err := WriteMessage(conn, Message{Type: EventAudioStart, Data: map[string]any{
		"rate": sampleRate, "width": 2, "channels": 1, "timestamp": 0,
	}}); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("wyoming: start wake audio stream: %w", err)
	}
	_ = conn.SetWriteDeadline(time.Time{})
	session := &WakeSession{conn: conn, events: make(chan WakeEvent, 4), closed: make(chan struct{}), rate: sampleRate}
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-session.closed:
		}
	}()
	go session.readLoop(bufio.NewReader(conn))
	return session, nil
}

func (s *WakeSession) readLoop(reader *bufio.Reader) {
	defer close(s.events)
	for {
		msg, err := ReadMessage(reader)
		if err != nil {
			select {
			case <-s.closed:
				return
			default:
			}
			select {
			case s.events <- WakeEvent{Err: fmt.Errorf("wyoming: wake stream read: %w", err)}:
			case <-s.closed:
			}
			return
		}
		switch msg.Type {
		case EventDetection:
			event := WakeEvent{}
			if msg.Data != nil {
				event.Name, _ = msg.Data["name"].(string)
				if timestamp, ok := msg.Data["timestamp"].(float64); ok {
					event.TimestampMS = int64(timestamp)
				}
			}
			select {
			case s.events <- event:
			case <-s.closed:
				return
			}
		case EventNotDetected:
			select {
			case s.events <- WakeEvent{Err: ErrWakeNotDetected}:
			case <-s.closed:
			}
		}
	}
}
