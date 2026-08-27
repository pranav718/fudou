package chunker

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
)

type FixedChunker struct {
	defaultChunkSize int64
}

func NewFixedChunker(defaultChunkSize int64) *FixedChunker {
	if defaultChunkSize <= 0 {
		defaultChunkSize = DefaultChunkSize
	}
	return &FixedChunker{
		defaultChunkSize: defaultChunkSize,
	}
}

func (fc *FixedChunker) Split(r io.Reader, chunkSize int64) ([]Chunk, error) {
	if chunkSize <= 0 {
		chunkSize = fc.defaultChunkSize
	}

	var chunks []Chunk
	buf := make([]byte, chunkSize)
	index := 0

	for {
		n, err := io.ReadFull(r, buf)
		if n > 0 {
			chunkData := make([]byte, n)
			copy(chunkData, buf[:n])

			hash := sha256.Sum256(chunkData)
			checksum := hex.EncodeToString(hash[:])
			chunkID := fmt.Sprintf("chk-%d-%s", index, checksum[:16])

			chunks = append(chunks, Chunk{
				Index:    index,
				ID:       chunkID,
				Data:     chunkData,
				Size:     int64(n),
				Checksum: checksum,
			})
			index++
		}

		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}

	return chunks, nil
}

func (fc *FixedChunker) Join(chunks []Chunk, w io.Writer) error {
	reassembler := NewReassembler()
	return reassembler.Join(chunks, w)
}
