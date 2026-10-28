// Package client calls the Superlog management API.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultEndpoint is the EU cell. The US cell is https://us.superlog.sh.
const DefaultEndpoint = "https://superlog.sh"

type Client struct {
	apiKey    string
	baseURL   string
	http      *http.Client
	userAgent string
}

func New(endpoint, apiKey, userAgent string) (*Client, error) {
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
		return nil, fmt.Errorf("endpoint must be an HTTP or HTTPS URL, such as %s", DefaultEndpoint)
	}
	return &Client{
		apiKey:    apiKey,
		baseURL:   strings.TrimRight(endpoint, "/") + "/api/v1",
		http:      &http.Client{Timeout: 60 * time.Second},
		userAgent: userAgent,
	}, nil
}

// Error is a failed API response.
type Error struct {
	Status  int
	Code    string `json:"code"`
	Message string `json:"error"`
	Issues  []struct {
		Message string `json:"message"`
		Path    []any  `json:"path"`
	} `json:"issues"`
}

func (e *Error) Error() string {
	message := e.Message
	if message == "" {
		message = http.StatusText(e.Status)
	}
	parts := []string{fmt.Sprintf("%s (HTTP %d", message, e.Status)}
	if e.Code != "" {
		parts[0] += ", " + e.Code
	}
	parts[0] += ")"
	for _, issue := range e.Issues {
		path := make([]string, len(issue.Path))
		for i, part := range issue.Path {
			path[i] = fmt.Sprint(part)
		}
		if len(path) > 0 {
			parts = append(parts, fmt.Sprintf("%s: %s", strings.Join(path, "."), issue.Message))
		} else {
			parts = append(parts, issue.Message)
		}
	}
	return strings.Join(parts, "\n")
}

// IsNotFound reports whether the object no longer exists.
func IsNotFound(err error) bool {
	var apiErr *Error
	return errors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound &&
		strings.HasSuffix(apiErr.Code, "_not_found")
}

func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+c.apiKey)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", c.userAgent)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := c.http.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 32<<20))
	if err != nil {
		return err
	}
	if response.StatusCode >= 300 {
		apiErr := &Error{Status: response.StatusCode}
		_ = json.Unmarshal(data, apiErr)
		return apiErr
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("reading the %s %s response: %w", method, path, err)
	}
	return nil
}

func escape(id string) string { return url.PathEscape(id) }
