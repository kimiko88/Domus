package failover_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/application/failover"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWALStreamer_BroadcastAndApply(t *testing.T) {
	tempDir := t.TempDir()
	leaderWAL := filepath.Join(tempDir, "leader.db-wal")
	standbyWAL := filepath.Join(tempDir, "standby.db-wal")

	streamer := failover.NewWALStreamer(leaderWAL)
	defer streamer.Close()

	receiver := failover.NewWALReceiver(standbyWAL)

	subCh := streamer.Subscribe()
	defer streamer.Unsubscribe(subCh)

	// 1. Broadcast Frame #1 (4KB simulated page)
	page1 := make([]byte, 4096)
	for i := range page1 {
		page1[i] = byte(i % 256)
	}

	frame1, err := streamer.BroadcastFrame(0, page1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), frame1.Sequence)
	assert.NotEmpty(t, frame1.Checksum)

	// Standby receives frame from channel
	receivedFrame1 := <-subCh
	assert.Equal(t, frame1.Sequence, receivedFrame1.Sequence)

	// Standby applies frame
	err = receiver.ApplyFrame(receivedFrame1)
	require.NoError(t, err)

	lastSeq, lagMs, bytesApplied := receiver.Stats()
	assert.Equal(t, int64(1), lastSeq)
	assert.True(t, lagMs >= 0)
	assert.Equal(t, int64(4096), bytesApplied)

	// Verify written file on standby
	writtenData, err := os.ReadFile(standbyWAL)
	require.NoError(t, err)
	assert.Equal(t, page1, writtenData)

	// 2. Broadcast Frame #2
	page2 := []byte("Domus WAL Transaction Commit Record #2")
	frame2, err := streamer.BroadcastFrame(4096, page2)
	require.NoError(t, err)
	assert.Equal(t, int64(2), frame2.Sequence)

	receivedFrame2 := <-subCh
	err = receiver.ApplyFrame(receivedFrame2)
	require.NoError(t, err)

	lastSeq, _, bytesApplied = receiver.Stats()
	assert.Equal(t, int64(2), lastSeq)
	assert.Equal(t, int64(4096+len(page2)), bytesApplied)
}

func TestWALStreamer_TamperedChecksumFails(t *testing.T) {
	tempDir := t.TempDir()
	standbyWAL := filepath.Join(tempDir, "standby_tampered.db-wal")
	receiver := failover.NewWALReceiver(standbyWAL)

	corruptedFrame := failover.WALFrame{
		Sequence:  1,
		Offset:    0,
		Data:      []byte("Corrupted WAL Page Data"),
		Checksum:  "invalid-fake-checksum",
		Timestamp: time.Now().UTC(),
	}

	err := receiver.ApplyFrame(corruptedFrame)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "checksum mismatch")
}

func TestWALStreamer_OutOfSequenceFails(t *testing.T) {
	tempDir := t.TempDir()
	standbyWAL := filepath.Join(tempDir, "standby_seq.db-wal")
	receiver := failover.NewWALReceiver(standbyWAL)

	data1 := []byte("WAL Frame 1")
	frame1 := failover.WALFrame{
		Sequence:  1,
		Offset:    0,
		Data:      data1,
		Checksum:  failover.CalculateChecksum(data1),
		Timestamp: time.Now().UTC(),
	}
	require.NoError(t, receiver.ApplyFrame(frame1))

	// Skip sequence 2 and try applying sequence 3
	data3 := []byte("WAL Frame 3")
	frame3 := failover.WALFrame{
		Sequence:  3,
		Offset:    int64(len(data1)),
		Data:      data3,
		Checksum:  failover.CalculateChecksum(data3),
		Timestamp: time.Now().UTC(),
	}

	err := receiver.ApplyFrame(frame3)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "out of sequence")
}

func TestWALStreamer_SyncFileFromDisk(t *testing.T) {
	tempDir := t.TempDir()
	leaderWAL := filepath.Join(tempDir, "real_leader.db-wal")

	// Create and write to real WAL file
	initialData := []byte("SQLite format 3 WAL Header 32-bytes")
	require.NoError(t, os.WriteFile(leaderWAL, initialData, 0644))

	streamer := failover.NewWALStreamer(leaderWAL)
	defer streamer.Close()

	frames, err := streamer.SyncFile()
	require.NoError(t, err)
	require.Len(t, frames, 1)
	assert.Equal(t, int64(1), frames[0].Sequence)
	assert.Equal(t, initialData, frames[0].Data)

	// Append more data
	f, err := os.OpenFile(leaderWAL, os.O_APPEND|os.O_WRONLY, 0644)
	require.NoError(t, err)
	appendData := []byte("Page 1 WAL Write Frame")
	_, err = f.Write(appendData)
	require.NoError(t, err)
	f.Close()

	// Sync again
	frames2, err := streamer.SyncFile()
	require.NoError(t, err)
	require.Len(t, frames2, 1)
	assert.Equal(t, int64(2), frames2[0].Sequence)
	assert.Equal(t, appendData, frames2[0].Data)
}
