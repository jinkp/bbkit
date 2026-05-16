package bitbucket

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const baseURL = "https://api.bitbucket.org/2.0"

type Client struct {
	authHeader string
	baseURL    string
	httpClient *http.Client
}

func NewClient(username, apiToken string) *Client {
	encoded := base64.StdEncoding.EncodeToString([]byte(username + ":" + apiToken))

	return &Client{
		authHeader: "Basic " + encoded,
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) Get(ctx context.Context, path string, out any) error {
	return c.request(ctx, http.MethodGet, path, nil, "application/json", out)
}

func (c *Client) Post(ctx context.Context, path string, body, out any) error {
	return c.request(ctx, http.MethodPost, path, body, "application/json", out)
}

func (c *Client) Patch(ctx context.Context, path string, body, out any) error {
	return c.request(ctx, http.MethodPatch, path, body, "application/json", out)
}

func (c *Client) GetText(ctx context.Context, path string) (string, error) {
	var builder strings.Builder
	err := c.requestRaw(ctx, http.MethodGet, path, nil, "text/plain", func(response *http.Response) error {
		_, err := io.Copy(&builder, response.Body)
		return err
	})
	if err != nil {
		return "", err
	}

	return builder.String(), nil
}

func Paginate[T any](ctx context.Context, client *Client, path string) ([]T, error) {
	var values []T
	currentPath := path

	for currentPath != "" {
		var page PaginatedResponse[T]
		if err := client.Get(ctx, currentPath, &page); err != nil {
			return nil, err
		}

		values = append(values, page.Values...)
		currentPath = page.Next
	}

	return values, nil
}

func (c *Client) request(ctx context.Context, method, path string, body any, accept string, out any) error {
	return c.requestRaw(ctx, method, path, body, accept, func(response *http.Response) error {
		if out == nil || response.StatusCode == http.StatusNoContent {
			_, _ = io.Copy(io.Discard, response.Body)
			return nil
		}

		decoder := json.NewDecoder(response.Body)
		if err := decoder.Decode(out); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("decode response: %w", err)
		}

		return nil
	})
}

func (c *Client) requestRaw(ctx context.Context, method, path string, body any, accept string, handle func(*http.Response) error) error {
	url := c.resolveURL(path)

	requestBody, err := marshalBody(body)
	if err != nil {
		return err
	}

	request, err := http.NewRequestWithContext(ctx, method, url, requestBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	request.Header.Set("Authorization", c.authHeader)
	request.Header.Set("Accept", accept)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return &CLIError{Message: "Unable to reach Bitbucket API. Check your connection.", ExitCode: 2}
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		payload, _ := io.ReadAll(response.Body)
		return MapAPIError(response.StatusCode, url, strings.TrimSpace(string(payload)))
	}

	return handle(response)
}

func (c *Client) resolveURL(path string) string {
	if strings.HasPrefix(path, "https://") || strings.HasPrefix(path, "http://") {
		return path
	}
	return c.baseURL + path
}

func marshalBody(body any) (io.Reader, error) {
	if body == nil {
		return nil, nil
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request body: %w", err)
	}

	return bytes.NewReader(payload), nil
}
