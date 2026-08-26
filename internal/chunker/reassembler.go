package chunker

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sort"
)

var (
	ErrEmptyChunkList      = errors.New("no chunks provided for reassembly")
	ErrChunkChecksumFailed = errors.New("chunk checksum mismatch")
	ErrMissingChunkIndex   = errors.New("missing chunk sequence index")
)

type Reassembler struct{}

func NewReassembler() *Reassembler {
	return &Reassembler{}
}

func (ras *Reassembler) Join(chunks []Chunk, w io.Writer) error {
	if len(chunks) == 0 {
		return ErrEmptyChunkList
	}

	sortedChunks := make([]Chunk, len(chunks))
	copy(sortedChunks, chunks)
	sort.Slice(sortedChunks, func(i, j int) bool {
		return sortedChunks[i].Index < sortedChunks[j].Index
	})

	for expectedIdx, chk := range sortedChunks {
		if chk.Index != expectedIdx {
			return fmt.Errorf("%w: expected %d but found %d", ErrMissingChunkIndex, expectedIdx, chk.Index)
		}

		hash := sha256.Sum256(chk.Data)
		actualChecksum := hex.EncodeToString(hash[:])
		if actualChecksum != chk.Checksum {
			return fmt.Errorf("%w: chunk %s expected %s, got %s", ErrChunkChecksumFailed, chk.ID, chk.Checksum, actualChecksum)
		}

		if _, err := w.Write(chk.Data); err != nil {
			return err
		}
	}

	return nil
}
