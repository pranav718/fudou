package tui

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pranav718/fudou/internal/api"
	"github.com/pranav718/fudou/internal/coordinator"
	"github.com/pranav718/fudou/internal/metadata"
)

const clientErrorBodyLimit = 4096

type Snapshot struct {
	Files   []metadata.FileRecord
	Nodes   []metadata.NodeRecord
	Metrics api.ClusterMetrics
}

type Client struct {
	baseURL string
	userID  string
	http    *http.Client
}

func NewClient(baseURL, userID string) (*Client, error) {
	u, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.Opaque != "" {
		return nil, errors.New("base URL must be an absolute http or https URL")
	}
	if u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		return nil, errors.New("base URL must not contain credentials, a query, or a fragment")
	}
	if port := u.Port(); port != "" {
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 {
			return nil, errors.New("base URL has an invalid port")
		}
	} else if strings.HasSuffix(u.Host, ":") {
		return nil, errors.New("base URL has an empty port")
	}
	return &Client{
		baseURL: strings.TrimRight(u.String(), "/"),
		userID:  userID,
		http: &http.Client{
			Timeout: 10 * time.Minute,
			Transport: &http.Transport{
				Proxy:                 http.ProxyFromEnvironment,
				DialContext:           (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
				ForceAttemptHTTP2:     true,
				MaxIdleConns:          10,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ResponseHeaderTimeout: 2 * time.Minute,
				ExpectContinueTimeout: time.Second,
			},
			// Do not forward download keys or silently redirect an upload to a GET.
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}, nil
}

func (c *Client) Snapshot(ctx context.Context) (Snapshot, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var snapshot Snapshot
	query := make(url.Values)
	if c.userID != "" {
		query.Set("user_id", c.userID)
	}
	jobs := []struct {
		route  string
		query  url.Values
		target any
	}{
		{"files", query, &snapshot.Files},
		{"admin/nodes", nil, &snapshot.Nodes},
		{"admin/metrics", nil, &snapshot.Metrics},
	}
	results := make(chan error, len(jobs))
	for _, job := range jobs {
		go func() {
			req, err := c.request(ctx, http.MethodGet, job.route, job.query, nil)
			if err == nil {
				var resp *http.Response
				resp, err = c.do(req)
				if err == nil {
					defer resp.Body.Close()
					err = decodeClientJSON(resp.Body, job.target)
				}
			}
			if err != nil {
				err = fmt.Errorf("fetch %s: %w", job.route, err)
			}
			results <- err
		}()
	}
	var firstErr error
	for range jobs {
		if err := <-results; err != nil && firstErr == nil {
			firstErr = err
			cancel()
		}
	}
	if firstErr != nil {
		return Snapshot{}, firstErr
	}
	sort.SliceStable(snapshot.Files, func(i, j int) bool {
		return snapshot.Files[i].CreatedAt.After(snapshot.Files[j].CreatedAt)
	})
	sort.SliceStable(snapshot.Nodes, func(i, j int) bool {
		return snapshot.Nodes[i].ID < snapshot.Nodes[j].ID
	})
	return snapshot, nil
}

func (c *Client) Upload(ctx context.Context, path string) (coordinator.BackupResult, error) {
	var result coordinator.BackupResult
	if err := ctx.Err(); err != nil {
		return result, err
	}
	path, err := expandClientPath(path)
	if err != nil {
		return result, err
	}
	file, err := os.Open(path)
	if err != nil {
		return result, fmt.Errorf("open upload: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return result, fmt.Errorf("stat upload: %w", err)
	}
	if !info.Mode().IsRegular() {
		return result, errors.New("upload source must be a regular file")
	}

	reader, writer := io.Pipe()
	defer reader.Close()
	form := multipart.NewWriter(writer)
	req, err := c.request(ctx, http.MethodPost, "files", nil, reader)
	if err != nil {
		writer.Close()
		return result, err
	}
	req.Header.Set("Content-Type", form.FormDataContentType())
	userID := c.userID
	if userID == "" {
		userID = "anonymous"
	}
	finished := make(chan error, 1)
	go func() {
		err := form.WriteField("user_id", userID)
		if err == nil {
			var part io.Writer
			part, err = form.CreateFormFile("file", filepath.Base(path))
			if err == nil {
				_, err = io.Copy(part, file)
			}
		}
		if err == nil {
			err = form.Close()
		}
		writer.CloseWithError(err)
		finished <- err
	}()

	resp, requestErr := c.do(req)
	// An early response or cancellation must also unblock the multipart producer.
	reader.Close()
	streamErr := <-finished
	if requestErr != nil {
		return result, fmt.Errorf("upload: %w", requestErr)
	}
	defer resp.Body.Close()
	if streamErr != nil {
		return result, fmt.Errorf("stream upload: %w", streamErr)
	}
	if err := decodeClientJSON(resp.Body, &result); err != nil {
		return coordinator.BackupResult{}, fmt.Errorf("upload: %w", err)
	}
	return result, nil
}

func (c *Client) Restore(ctx context.Context, file metadata.FileRecord, keyHex, destination string) (resultErr error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(keyHex) != 64 {
		return errors.New("restore key must be exactly 32 bytes encoded as hexadecimal")
	}
	if _, err := hex.DecodeString(keyHex); err != nil {
		return errors.New("restore key must be exactly 32 bytes encoded as hexadecimal")
	}
	if file.Size < 0 || len(file.Checksum) != sha256.Size*2 {
		return errors.New("restore metadata must contain a nonnegative size and a SHA256 checksum")
	}
	checksum, err := hex.DecodeString(file.Checksum)
	if err != nil {
		return errors.New("restore metadata has an invalid SHA256 checksum")
	}
	route, err := clientFileRoute(file.ID)
	if err != nil {
		return err
	}
	destination, err = expandClientPath(destination)
	if err != nil {
		return err
	}
	// Lstat also rejects dangling symlinks. The final link enforces this again atomically.
	if _, err := os.Lstat(destination); err == nil {
		return fmt.Errorf("restore destination already exists: %w", os.ErrExist)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect restore destination: %w", err)
	}

	req, err := c.request(ctx, http.MethodGet, route+"/download", url.Values{"key": {keyHex}}, nil)
	if err != nil {
		return err
	}
	resp, err := c.do(req)
	if err != nil {
		return fmt.Errorf("restore: %w", err)
	}
	defer resp.Body.Close()

	temp, err := os.CreateTemp(filepath.Dir(destination), ".fudou-restore-*")
	if err != nil {
		return fmt.Errorf("create restore temporary file: %w", err)
	}
	closed := false
	defer func() {
		if !closed {
			if err := temp.Close(); err != nil {
				resultErr = errors.Join(resultErr, fmt.Errorf("close restore temporary file: %w", err))
			}
		}
		if err := os.Remove(temp.Name()); err != nil && !errors.Is(err, os.ErrNotExist) {
			resultErr = errors.Join(resultErr, fmt.Errorf("remove restore temporary file: %w", err))
		}
	}()
	if err := temp.Chmod(0600); err != nil {
		return fmt.Errorf("set restore permissions: %w", err)
	}

	hasher := sha256.New()
	// Stop at the expected size, then probe for an extra byte without overflowing size+1.
	n, err := io.Copy(io.MultiWriter(temp, hasher), io.LimitReader(resp.Body, file.Size))
	if err != nil {
		return fmt.Errorf("read restore: %w", safeClientError(err, keyHex))
	}
	if n != file.Size {
		return errors.New("restored file size does not match metadata")
	}
	var extra [1]byte
	if n, err := io.ReadFull(resp.Body, extra[:]); n != 0 {
		return errors.New("restored file size exceeds metadata")
	} else if !errors.Is(err, io.EOF) {
		return fmt.Errorf("finish restore: %w", safeClientError(err, keyHex))
	}
	if !bytes.Equal(hasher.Sum(nil), checksum) {
		return errors.New("restored file SHA256 checksum does not match metadata")
	}
	if err := temp.Sync(); err != nil {
		return fmt.Errorf("sync restored file: %w", err)
	}
	err = temp.Close()
	closed = true
	if err != nil {
		return fmt.Errorf("close restored file: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	// A same-directory hard link publishes only verified bytes, without replacing any entry.
	if err := os.Link(temp.Name(), destination); err != nil {
		return fmt.Errorf("publish restored file without overwriting: %w", err)
	}
	return nil
}

func (c *Client) Delete(ctx context.Context, id string) error {
	route, err := clientFileRoute(id)
	if err != nil {
		return err
	}
	req, err := c.request(ctx, http.MethodDelete, route, nil, nil)
	if err != nil {
		return err
	}
	resp, err := c.do(req)
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	return resp.Body.Close()
}

func (c *Client) request(ctx context.Context, method, route string, query url.Values, body io.Reader) (*http.Request, error) {
	endpoint := c.baseURL + "/api/" + route
	if len(query) != 0 {
		endpoint += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("create HTTP request: %w", safeClientError(err, query.Get("key")))
	}
	return req, nil
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	secret := req.URL.Query().Get("key")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", safeClientError(err, secret))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, clientErrorBodyLimit))
		message := strings.TrimSpace(string(body))
		if secret != "" {
			message = strings.ReplaceAll(message, secret, "[redacted]")
		}
		statusErr := fmt.Errorf("HTTP %d %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), message)
		if readErr != nil {
			return nil, errors.Join(statusErr, fmt.Errorf("read error response: %w", safeClientError(readErr, secret)))
		}
		return nil, statusErr
	}
	return resp, nil
}

func decodeClientJSON(body io.Reader, target any) error {
	decoder := json.NewDecoder(body)
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode JSON response: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err == nil {
		return errors.New("invalid trailing data in JSON response")
	} else if !errors.Is(err, io.EOF) {
		return fmt.Errorf("read JSON response trailer: %w", err)
	}
	return nil
}

func clientFileRoute(id string) (string, error) {
	if id == "" || id == "." || id == ".." || strings.Contains(id, "/") {
		return "", errors.New("file ID must be a nonempty path segment")
	}
	return "files/" + url.PathEscape(id), nil
}

func expandClientPath(path string) (string, error) {
	if path == "" {
		return "", errors.New("file path must not be empty")
	}
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("expand home directory: %w", err)
		}
		if path == "~" {
			return home, nil
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}

type clientRequestError struct {
	cause   error
	message string
}

func (e *clientRequestError) Error() string { return e.message }
func (e *clientRequestError) Unwrap() error { return e.cause }

func safeClientError(err error, secret string) error {
	if err == nil {
		return nil
	}
	// net/http includes the full request URL (and hence the restore key) in url.Error.
	for {
		u, ok := err.(*url.Error)
		if !ok {
			break
		}
		err = u.Err
	}
	message := err.Error()
	if secret != "" {
		message = strings.ReplaceAll(message, secret, "[redacted]")
	}
	return &clientRequestError{cause: err, message: message}
}
