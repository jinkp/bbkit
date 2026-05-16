package bitbucket

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClientGetUsesBasicAuthAndReturnsTypedResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/user", r.URL.Path)
		require.Equal(t, "Basic "+base64.StdEncoding.EncodeToString([]byte("user:token")), r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"username":"testuser"}`)
	}))
	defer server.Close()

	client := &Client{authHeader: "Basic " + base64.StdEncoding.EncodeToString([]byte("user:token")), baseURL: server.URL, httpClient: server.Client()}

	var got struct {
		Username string `json:"username"`
	}
	require.NoError(t, client.Get(context.Background(), "/user", &got))
	require.Equal(t, "testuser", got.Username)
}

func TestPaginateFollowsNextUntilExhausted(t *testing.T) {
	t.Parallel()

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.RawQuery {
		case "":
			_, _ = io.WriteString(w, fmt.Sprintf(`{"values":[{"name":"repo-1"},{"name":"repo-2"}],"next":"%s/repositories/ws?page=2"}`, server.URL))
		case "page=2":
			_, _ = io.WriteString(w, `{"values":[{"name":"repo-3"}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := &Client{authHeader: "Basic test", baseURL: server.URL, httpClient: server.Client()}

	type repo struct {
		Name string `json:"name"`
	}
	items, err := Paginate[repo](context.Background(), client, "/repositories/ws")
	require.NoError(t, err)
	require.Len(t, items, 3)
	require.Equal(t, "repo-3", items[2].Name)
}

func TestClientMapsAuthAndAPIErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		body       string
		assertErr  func(t *testing.T, err error)
	}{
		{
			name:       "401 maps to auth error",
			statusCode: http.StatusUnauthorized,
			body:       "Unauthorized",
			assertErr: func(t *testing.T, err error) {
				var cliErr *CLIError
				require.ErrorAs(t, err, &cliErr)
				require.Equal(t, "Authentication failed. Run `bbk auth login`.", cliErr.Message)
			},
		},
		{
			name:       "403 maps to forbidden error",
			statusCode: http.StatusForbidden,
			body:       "Forbidden",
			assertErr: func(t *testing.T, err error) {
				var cliErr *CLIError
				require.ErrorAs(t, err, &cliErr)
				require.Equal(t, "Permission denied. Check your API token scopes.", cliErr.Message)
			},
		},
		{
			name:       "404 maps to not found error",
			statusCode: http.StatusNotFound,
			body:       "Not Found",
			assertErr: func(t *testing.T, err error) {
				var cliErr *CLIError
				require.ErrorAs(t, err, &cliErr)
				require.Contains(t, cliErr.Message, "Resource not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = io.WriteString(w, tt.body)
			}))
			defer server.Close()

			client := &Client{authHeader: "Basic test", baseURL: server.URL, httpClient: server.Client()}
			var out map[string]any
			err := client.Get(context.Background(), "/user", &out)
			require.Error(t, err)
			tt.assertErr(t, err)
		})
	}
}

func TestClientMapsNetworkErrors(t *testing.T) {
	t.Parallel()

	client := &Client{
		authHeader: "Basic test",
		baseURL:    "https://api.bitbucket.org/2.0",
		httpClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("ENOTFOUND")
		})},
	}

	var out map[string]any
	err := client.Get(context.Background(), "/user", &out)
	require.Error(t, err)

	var cliErr *CLIError
	require.ErrorAs(t, err, &cliErr)
	require.Equal(t, "Unable to reach Bitbucket API. Check your connection.", cliErr.Message)
	require.Equal(t, 2, cliErr.ExitCode)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
