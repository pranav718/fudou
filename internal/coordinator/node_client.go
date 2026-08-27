package coordinator

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var (
	ErrRemoteChunkNotFound = errors.New("chunk not found on remote node")
	ErrNodeUnreachable     = errors.New("storage node unreachable")
)

type NodeClient interface {
	UploadChunk(ctx context.Context, nodeAddress string, chunkID string, data []byte) error
	DownloadChunk(ctx context.Context, nodeAddress string, chunkID string) ([]byte, error)
	DeleteChunk(ctx context.Context, nodeAddress string, chunkID string) error
	CheckHealth(ctx context.Context, nodeAddress string) error
}

type HTTPNodeClient struct {
	client *http.Client
}

func NewHTTPNodeClient(timeout time.Duration) *HTTPNodeClient {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &HTTPNodeClient{
		client: &http.Client{Timeout: timeout},
	}
}

func (c *HTTPNodeClient) formatURL(address string, path string) string {
	base := strings.TrimRight(address, "/")
	return fmt.Sprintf("%s/%s", base, strings.TrimLeft(path, "/"))
}

func (c *HTTPNodeClient) UploadChunk(ctx context.Context, nodeAddress string, chunkID string, data []byte) error {
	url := c.formatURL(nodeAddress, fmt.Sprintf("chunks/%s", chunkID))
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrNodeUnreachable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code from node: %d", resp.StatusCode)
	}

	return nil
}

func (c *HTTPNodeClient) DownloadChunk(ctx context.Context, nodeAddress string, chunkID string) ([]byte, error) {
	url := c.formatURL(nodeAddress, fmt.Sprintf("chunks/%s", chunkID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNodeUnreachable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrRemoteChunkNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code from node: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (c *HTTPNodeClient) DeleteChunk(ctx context.Context, nodeAddress string, chunkID string) error {
	url := c.formatURL(nodeAddress, fmt.Sprintf("chunks/%s", chunkID))
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrNodeUnreachable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("unexpected status code from node on delete: %d", resp.StatusCode)
	}

	return nil
}

func (c *HTTPNodeClient) CheckHealth(ctx context.Context, nodeAddress string) error {
	url := c.formatURL(nodeAddress, "health")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrNodeUnreachable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("node health status code: %d", resp.StatusCode)
	}

	return nil
}
