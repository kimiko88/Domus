// Package failover provides High Availability clustering, heartbeat monitoring,
// and real-time SQLite WAL page streaming replication between Leader and Standby hubs.
package failover

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// WALFrame represents a replicated SQLite Write-Ahead Log frame or page.
type WALFrame struct {
	Sequence  int64     `json:"sequence"`
	Offset    int64     `json:"offset"`
	Data      []byte    `json:"data"`
	Checksum  string    `json:"checksum"` // SHA-256 of Data
	Timestamp time.Time `json:"timestamp"`
}

// CalculateChecksum computes SHA-256 hex digest for frame data integrity.
func CalculateChecksum(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// WALStreamer runs on the Leader node and streams WAL frames to standby replicas.
type WALStreamer struct {
	mu           sync.RWMutex
	walPath      string
	lastOffset   int64
	lastSequence int64
	subscribers  map[chan WALFrame]struct{}
	closed       bool
}

// NewWALStreamer initializes a WAL streamer for the master SQLite database.
func NewWALStreamer(walPath string) *WALStreamer {
	return &WALStreamer{
		walPath:     walPath,
		subscribers: make(map[chan WALFrame]struct{}),
	}
}

// Subscribe returns a channel that receives live WAL frames as they are generated.
func (s *WALStreamer) Subscribe() chan WALFrame {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch := make(chan WALFrame, 128)
	s.subscribers[ch] = struct{}{}
	return ch
}

// Unsubscribe closes and unregisters a standby subscriber channel.
func (s *WALStreamer) Unsubscribe(ch chan WALFrame) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.subscribers[ch]; ok {
		delete(s.subscribers, ch)
		close(ch)
	}
}

// BroadcastFrame encodes and dispatches a verified WAL frame to all active standby nodes.
func (s *WALStreamer) BroadcastFrame(offset int64, data []byte) (WALFrame, error) {
	if len(data) == 0 {
		return WALFrame{}, errors.New("streamer: cannot broadcast empty WAL frame")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return WALFrame{}, errors.New("streamer: streamer is closed")
	}

	s.lastSequence++
	s.lastOffset = offset

	frame := WALFrame{
		Sequence:  s.lastSequence,
		Offset:    offset,
		Data:      data,
		Checksum:  CalculateChecksum(data),
		Timestamp: time.Now().UTC(),
	}

	for ch := range s.subscribers {
		select {
		case ch <- frame:
		default:
			// Non-blocking drop if standby channel buffer is saturated
		}
	}

	return frame, nil
}

// SyncFile reads newly appended bytes from the physical WAL file on disk.
func (s *WALStreamer) SyncFile() ([]WALFrame, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := os.Open(s.walPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // WAL file not created yet or truncated
		}
		return nil, fmt.Errorf("streamer: failed to open WAL file: %w", err)
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		return nil, err
	}

	fileSize := fi.Size()
	if fileSize <= s.lastOffset {
		if fileSize < s.lastOffset {
			// WAL was checkpointed/truncated, reset offset
			s.lastOffset = 0
		}
		return nil, nil
	}

	_, err = file.Seek(s.lastOffset, io.SeekStart)
	if err != nil {
		return nil, err
	}

	bytesToRead := fileSize - s.lastOffset
	buf := make([]byte, bytesToRead)
	n, err := io.ReadFull(file, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, err
	}

	if n == 0 {
		return nil, nil
	}

	s.lastSequence++
	frame := WALFrame{
		Sequence:  s.lastSequence,
		Offset:    s.lastOffset,
		Data:      buf[:n],
		Checksum:  CalculateChecksum(buf[:n]),
		Timestamp: time.Now().UTC(),
	}
	s.lastOffset += int64(n)

	for ch := range s.subscribers {
		select {
		case ch <- frame:
		default:
		}
	}

	return []WALFrame{frame}, nil
}

// Close gracefully stops the streamer and terminates subscriber channels.
func (s *WALStreamer) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.closed {
		s.closed = true
		for ch := range s.subscribers {
			close(ch)
		}
		s.subscribers = nil
	}
}

// WALReceiver runs on Standby nodes, receives frames, validates integrity, and writes to replica WAL.
type WALReceiver struct {
	mu           sync.RWMutex
	replicaPath  string
	lastSequence int64
	lagMs        int64
	framesCount  int64
	bytesApplied int64
}

// NewWALReceiver constructs a receiver writing to the local replica WAL file.
func NewWALReceiver(replicaPath string) *WALReceiver {
	return &WALReceiver{
		replicaPath: replicaPath,
	}
}

// ApplyFrame verifies checksum and sequence, and writes the frame payload.
func (r *WALReceiver) ApplyFrame(frame WALFrame) error {
	// 1. Verify Checksum
	expectedChecksum := CalculateChecksum(frame.Data)
	if frame.Checksum != expectedChecksum {
		return fmt.Errorf("receiver: checksum mismatch! expected %s, got %s", expectedChecksum, frame.Checksum)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// 2. Monotonic sequence check
	if r.lastSequence > 0 && frame.Sequence != r.lastSequence+1 {
		return fmt.Errorf("receiver: out of sequence frame! expected %d, got %d", r.lastSequence+1, frame.Sequence)
	}

	// 3. Write to replica file at Offset
	flags := os.O_CREATE | os.O_WRONLY
	file, err := os.OpenFile(r.replicaPath, flags, 0644)
	if err != nil {
		return fmt.Errorf("receiver: cannot open replica WAL: %w", err)
	}
	defer file.Close()

	if frame.Offset >= 0 {
		if _, err := file.Seek(frame.Offset, io.SeekStart); err != nil {
			return fmt.Errorf("receiver: seek error: %w", err)
		}
	}

	n, err := file.Write(frame.Data)
	if err != nil {
		return fmt.Errorf("receiver: write error: %w", err)
	}

	r.lastSequence = frame.Sequence
	r.framesCount++
	r.bytesApplied += int64(n)

	// Calculate replication transit lag
	if !frame.Timestamp.IsZero() {
		transit := time.Since(frame.Timestamp)
		r.lagMs = transit.Milliseconds()
	}

	return nil
}

// Stats returns the real-time synchronization state of the standby receiver.
func (r *WALReceiver) Stats() (lastSeq int64, lagMs int64, bytesApplied int64) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.lastSequence, r.lagMs, r.bytesApplied
}
