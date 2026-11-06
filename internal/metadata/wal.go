package metadata

import (
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
	"os"
	"sync"
)

const (
	WALMagic = byte(0xFD)

	OpSaveFile          byte = 1
	OpDeleteFile        byte = 2
	OpSaveChunkLocation byte = 3
	OpRegisterNode      byte = 4
)

var (
	ErrCorruptWALRecord = errors.New("corrupt wal record detected")
	ErrInvalidWALMagic  = errors.New("invalid wal magic byte")
)

type WAL struct {
	mu   sync.Mutex
	path string
	file *os.File
}

func OpenWAL(path string) (*WAL, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	return &WAL{
		path: path,
		file: file,
	}, nil
}

func (w *WAL) Append(op byte, payload []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	length := uint32(len(payload))
	checksum := crc32.ChecksumIEEE(payload)

	header := make([]byte, 10)
	header[0] = WALMagic
	header[1] = op
	binary.BigEndian.PutUint32(header[2:6], length)
	binary.BigEndian.PutUint32(header[6:10], checksum)

	if _, err := w.file.Write(header); err != nil {
		return err
	}
	if _, err := w.file.Write(payload); err != nil {
		return err
	}
	return nil
}

func (w *WAL) Sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.file.Sync()
}

func (w *WAL) Replay(handler func(op byte, payload []byte) error) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	file, err := os.Open(w.path)
	if err != nil {
		return err
	}
	defer file.Close()

	header := make([]byte, 10)
	for {
		_, err := io.ReadFull(file, header)
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			break
		}
		if err != nil {
			return err
		}

		if header[0] != WALMagic {
			return ErrInvalidWALMagic
		}

		op := header[1]
		length := binary.BigEndian.Uint32(header[2:6])
		expectedChecksum := binary.BigEndian.Uint32(header[6:10])

		payload := make([]byte, length)
		if _, err := io.ReadFull(file, payload); err != nil {
			break
		}

		actualChecksum := crc32.ChecksumIEEE(payload)
		if actualChecksum != expectedChecksum {
			return ErrCorruptWALRecord
		}

		if err := handler(op, payload); err != nil {
			return err
		}
	}

	return nil
}

func (w *WAL) Truncate() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.file.Close(); err != nil {
		return err
	}

	file, err := os.OpenFile(w.path, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	w.file = file
	return nil
}

func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.file.Close()
}
