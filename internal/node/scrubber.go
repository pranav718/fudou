package node

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type ScrubReport struct {
	ScrubbedChunks  int64     `json:"scrubbed_chunks"`
	CorruptedChunks int64     `json:"corrupted_chunks"`
	BytesScrubbed   int64     `json:"bytes_scrubbed"`
	DurationMs      int64     `json:"duration_ms"`
	LastScrubTime   time.Time `json:"last_scrub_time"`
	IsRunning       bool      `json:"is_running"`
}

type BitRotScrubber struct {
	mu         sync.RWMutex
	baseDir    string
	lastReport ScrubReport
}

func NewBitRotScrubber(baseDir string) *BitRotScrubber {
	return &BitRotScrubber{
		baseDir: baseDir,
		lastReport: ScrubReport{
			LastScrubTime: time.Time{},
		},
	}
}

func (s *BitRotScrubber) RunScrub() (*ScrubReport, error) {
	s.mu.Lock()
	if s.lastReport.IsRunning {
		reportCopy := s.lastReport
		s.mu.Unlock()
		return &reportCopy, nil
	}
	s.lastReport.IsRunning = true
	s.mu.Unlock()

	startTime := time.Now()
	var scrubbed int64
	var corrupted int64
	var totalBytes int64

	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		s.mu.Lock()
		s.lastReport.IsRunning = false
		s.mu.Unlock()
		return nil, err
	}

	hasher := sha256.New()
	buf := make([]byte, 32*1024)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		path := filepath.Join(s.baseDir, entry.Name())
		file, err := os.Open(path)
		if err != nil {
			corrupted++
			continue
		}

		hasher.Reset()
		n, err := io.CopyBuffer(hasher, file, buf)
		file.Close()

		if err != nil {
			corrupted++
		} else {
			scrubbed++
			totalBytes += n
			_ = hex.EncodeToString(hasher.Sum(nil))
		}
	}

	duration := time.Since(startTime).Milliseconds()

	s.mu.Lock()
	s.lastReport = ScrubReport{
		ScrubbedChunks:  scrubbed,
		CorruptedChunks: corrupted,
		BytesScrubbed:   totalBytes,
		DurationMs:      duration,
		LastScrubTime:   time.Now(),
		IsRunning:       false,
	}
	reportCopy := s.lastReport
	s.mu.Unlock()

	return &reportCopy, nil
}

func (s *BitRotScrubber) GetReport() ScrubReport {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastReport
}

func (s *BitRotScrubber) StartBackground(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _ = s.RunScrub()
		}
	}
}
