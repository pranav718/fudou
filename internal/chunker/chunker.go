package chunker

import (
	"io"
)

const (
	DefaultChunkSize = 4 * 1024 * 1024
)

type Chunk struct {
	Index    int
	ID       string
	Data     []byte
	Size     int64
	Checksum string
}

type Chunker interface {
	Split(r io.Reader, chunkSize int64) ([]Chunk, error)
	Join(chunks []Chunk, w io.Writer) error
}
