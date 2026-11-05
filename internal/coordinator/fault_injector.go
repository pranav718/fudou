package coordinator

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrInjectedFault     = errors.New("injected network fault")
	ErrInjectedTimeout   = errors.New("injected network timeout")
	ErrInjectedCorrupted = errors.New("injected corrupted payload")
)

type FaultInjectingNodeClient struct {
	underlying       NodeClient
	mu               sync.RWMutex
	offlineNodes     map[string]bool
	uploadErrors     map[string]error
	downloadErrors   map[string]error
	corruptDownloads map[string]bool
	latencies        map[string]time.Duration
	uploadCalls      atomic.Int64
	downloadCalls    atomic.Int64
	deleteCalls      atomic.Int64
}

func NewFaultInjectingNodeClient(underlying NodeClient) *FaultInjectingNodeClient {
	return &FaultInjectingNodeClient{
		underlying:       underlying,
		offlineNodes:     make(map[string]bool),
		uploadErrors:     make(map[string]error),
		downloadErrors:   make(map[string]error),
		corruptDownloads: make(map[string]bool),
		latencies:        make(map[string]time.Duration),
	}
}

func (f *FaultInjectingNodeClient) SetNodeOffline(nodeAddress string, offline bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.offlineNodes[nodeAddress] = offline
}

func (f *FaultInjectingNodeClient) SetUploadError(nodeAddress string, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err == nil {
		delete(f.uploadErrors, nodeAddress)
	} else {
		f.uploadErrors[nodeAddress] = err
	}
}

func (f *FaultInjectingNodeClient) SetDownloadError(nodeAddress string, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err == nil {
		delete(f.downloadErrors, nodeAddress)
	} else {
		f.downloadErrors[nodeAddress] = err
	}
}

func (f *FaultInjectingNodeClient) SetCorruptDownload(nodeAddress string, corrupt bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.corruptDownloads[nodeAddress] = corrupt
}

func (f *FaultInjectingNodeClient) SetLatency(nodeAddress string, d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if d <= 0 {
		delete(f.latencies, nodeAddress)
	} else {
		f.latencies[nodeAddress] = d
	}
}

func (f *FaultInjectingNodeClient) applyLatency(ctx context.Context, nodeAddress string) error {
	f.mu.RLock()
	d := f.latencies[nodeAddress]
	f.mu.RUnlock()

	if d > 0 {
		select {
		case <-time.After(d):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (f *FaultInjectingNodeClient) UploadChunk(ctx context.Context, nodeAddress string, chunkID string, data []byte) error {
	f.uploadCalls.Add(1)

	if err := f.applyLatency(ctx, nodeAddress); err != nil {
		return err
	}

	f.mu.RLock()
	isOffline := f.offlineNodes[nodeAddress]
	injectedErr := f.uploadErrors[nodeAddress]
	f.mu.RUnlock()

	if isOffline {
		return ErrNodeUnreachable
	}
	if injectedErr != nil {
		return injectedErr
	}

	if f.underlying != nil {
		return f.underlying.UploadChunk(ctx, nodeAddress, chunkID, data)
	}
	return nil
}

func (f *FaultInjectingNodeClient) DownloadChunk(ctx context.Context, nodeAddress string, chunkID string) ([]byte, error) {
	f.downloadCalls.Add(1)

	if err := f.applyLatency(ctx, nodeAddress); err != nil {
		return nil, err
	}

	f.mu.RLock()
	isOffline := f.offlineNodes[nodeAddress]
	injectedErr := f.downloadErrors[nodeAddress]
	shouldCorrupt := f.corruptDownloads[nodeAddress]
	f.mu.RUnlock()

	if isOffline {
		return nil, ErrNodeUnreachable
	}
	if injectedErr != nil {
		return nil, injectedErr
	}

	var data []byte
	var err error
	if f.underlying != nil {
		data, err = f.underlying.DownloadChunk(ctx, nodeAddress, chunkID)
		if err != nil {
			return nil, err
		}
	} else {
		data = []byte("mock-payload")
	}

	if shouldCorrupt && len(data) > 0 {
		corrupted := make([]byte, len(data))
		copy(corrupted, data)
		corrupted[0] ^= 0xff
		return corrupted, nil
	}

	return data, nil
}

func (f *FaultInjectingNodeClient) DeleteChunk(ctx context.Context, nodeAddress string, chunkID string) error {
	f.deleteCalls.Add(1)

	f.mu.RLock()
	isOffline := f.offlineNodes[nodeAddress]
	f.mu.RUnlock()

	if isOffline {
		return ErrNodeUnreachable
	}

	if f.underlying != nil {
		return f.underlying.DeleteChunk(ctx, nodeAddress, chunkID)
	}
	return nil
}

func (f *FaultInjectingNodeClient) CheckHealth(ctx context.Context, nodeAddress string) error {
	f.mu.RLock()
	isOffline := f.offlineNodes[nodeAddress]
	f.mu.RUnlock()

	if isOffline {
		return ErrNodeUnreachable
	}

	if f.underlying != nil {
		return f.underlying.CheckHealth(ctx, nodeAddress)
	}
	return nil
}

func (f *FaultInjectingNodeClient) Stats() (int64, int64, int64) {
	return f.uploadCalls.Load(), f.downloadCalls.Load(), f.deleteCalls.Load()
}
