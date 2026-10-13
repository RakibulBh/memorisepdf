// Package tika pulls the readable text out of a slide deck or document by
// asking a self-hosted Apache Tika server.
//
// This replaces github.com/google/go-tika, which has not been released since
// 2024 and only ever wrapped these two HTTP calls.
package tika

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	http    *http.Client
}

// Points at a Tika server.
func New(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		http:    &http.Client{Timeout: timeout},
	}
}

// Reads a document and gives back its plain text.
func (c *Client) Extract(ctx context.Context, r io.Reader, contentType string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.baseURL+"/tika", r)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Accept", "text/plain")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("call tika: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read tika response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Tika answers 422 for a file it recognises but cannot read, which is a
		// problem with the upload rather than with the server.
		if resp.StatusCode == http.StatusUnprocessableEntity {
			return "", fmt.Errorf("%w: %s", ErrUnsupported, strings.TrimSpace(string(body)))
		}
		return "", fmt.Errorf("tika returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	return strings.TrimSpace(string(body)), nil
}

// ErrUnsupported means Tika could not read the file at all.
var ErrUnsupported = fmt.Errorf("unsupported document")

// Checks the Tika server is up and answering.
func (c *Client) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/version", nil)
	if err != nil {
		return err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("reach tika: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("tika returned %s", resp.Status)
	}
	return nil
}
