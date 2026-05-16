package bitbucket

import "fmt"

type CLIError struct {
	Message  string
	ExitCode int
}

func (e *CLIError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

type APIError struct {
	Message    string
	StatusCode int
	URL        string
}

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}
	if e.URL == "" {
		return fmt.Sprintf("bitbucket api error (%d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("bitbucket api error (%d) for %s: %s", e.StatusCode, e.URL, e.Message)
}

func MapAPIError(statusCode int, url, body string) error {
	switch statusCode {
	case 401:
		return &CLIError{Message: "Authentication failed. Run `bbk auth login`.", ExitCode: 1}
	case 403:
		return &CLIError{Message: "Permission denied. Check your API token scopes.", ExitCode: 1}
	case 404:
		message := "Resource not found."
		if url != "" {
			message = fmt.Sprintf("Resource not found: %s.", url)
		}
		return &CLIError{Message: message, ExitCode: 1}
	default:
		if body == "" {
			body = "request failed"
		}
		return &APIError{Message: body, StatusCode: statusCode, URL: url}
	}
}
