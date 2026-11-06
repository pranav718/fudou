package metadata

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestWALAppendAndReplay(t *testing.T) {
	tempDir := t.TempDir()
	walPath := filepath.Join(tempDir, "test.wal")

	wal, err := OpenWAL(walPath)
	if err != nil {
		t.Fatalf("failed to open wal: %v", err)
	}
	defer wal.Close()

	payload1 := []byte(`{"id":"file-1","name":"report.pdf"}`)
	payload2 := []byte(`{"id":"file-2","name":"dataset.csv"}`)

	if err := wal.Append(OpSaveFile, payload1); err != nil {
		t.Fatalf("append 1 failed: %v", err)
	}
	if err := wal.Append(OpSaveFile, payload2); err != nil {
		t.Fatalf("append 2 failed: %v", err)
	}

	var replayed [][]byte
	var ops []byte
	err = wal.Replay(func(op byte, payload []byte) error {
		ops = append(ops, op)
		replayed = append(replayed, payload)
		return nil
	})
	if err != nil {
		t.Fatalf("replay failed: %v", err)
	}

	if len(replayed) != 2 {
		t.Fatalf("expected 2 records, got %d", len(replayed))
	}
	if ops[0] != OpSaveFile || !bytes.Equal(replayed[0], payload1) {
		t.Fatalf("record 1 mismatch")
	}
	if ops[1] != OpSaveFile || !bytes.Equal(replayed[1], payload2) {
		t.Fatalf("record 2 mismatch")
	}
}

func TestWALCrashRecoveryTruncatedRecord(t *testing.T) {
	tempDir := t.TempDir()
	walPath := filepath.Join(tempDir, "truncated.wal")

	wal, err := OpenWAL(walPath)
	if err != nil {
		t.Fatalf("failed to open wal: %v", err)
	}

	payload := []byte(`{"id":"valid-record"}`)
	if err := wal.Append(OpSaveFile, payload); err != nil {
		t.Fatalf("append failed: %v", err)
	}
	wal.Close()

	file, err := os.OpenFile(walPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("failed to reopen wal for truncation test: %v", err)
	}
	garbageHeader := []byte{WALMagic, OpSaveFile, 0x00, 0x00, 0x01, 0x00, 0xAA, 0xBB, 0xCC, 0xDD, 0x01, 0x02}
	if _, err := file.Write(garbageHeader); err != nil {
		t.Fatalf("failed to write truncated bytes: %v", err)
	}
	file.Close()

	reopened, err := OpenWAL(walPath)
	if err != nil {
		t.Fatalf("failed to reopen: %v", err)
	}
	defer reopened.Close()

	count := 0
	err = reopened.Replay(func(op byte, p []byte) error {
		count++
		return nil
	})
	if err != nil {
		t.Fatalf("expected graceful truncation recovery, got error: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 valid record recovered, got %d", count)
	}
}

func TestWALCorruptedCRC(t *testing.T) {
	tempDir := t.TempDir()
	walPath := filepath.Join(tempDir, "corrupted.wal")

	wal, err := OpenWAL(walPath)
	if err != nil {
		t.Fatalf("failed to open wal: %v", err)
	}

	payload := []byte(`{"data":"important-checksummed-data"}`)
	if err := wal.Append(OpSaveFile, payload); err != nil {
		t.Fatalf("append failed: %v", err)
	}
	wal.Close()

	content, err := os.ReadFile(walPath)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	content[len(content)-1] ^= 0xFF
	if err := os.WriteFile(walPath, content, 0644); err != nil {
		t.Fatalf("tamper write failed: %v", err)
	}

	reopened, err := OpenWAL(walPath)
	if err != nil {
		t.Fatalf("failed to reopen: %v", err)
	}
	defer reopened.Close()

	err = reopened.Replay(func(op byte, p []byte) error {
		return nil
	})
	if !errors.Is(err, ErrCorruptWALRecord) {
		t.Fatalf("expected ErrCorruptWALRecord, got: %v", err)
	}
}

func TestWALTruncate(t *testing.T) {
	tempDir := t.TempDir()
	walPath := filepath.Join(tempDir, "trunc.wal")

	wal, err := OpenWAL(walPath)
	if err != nil {
		t.Fatalf("failed to open wal: %v", err)
	}
	defer wal.Close()

	_ = wal.Append(OpSaveFile, []byte("record1"))
	_ = wal.Append(OpSaveFile, []byte("record2"))

	if err := wal.Truncate(); err != nil {
		t.Fatalf("truncate failed: %v", err)
	}

	count := 0
	_ = wal.Replay(func(op byte, p []byte) error {
		count++
		return nil
	})
	if count != 0 {
		t.Fatalf("expected 0 records after truncate, got %d", count)
	}
}
