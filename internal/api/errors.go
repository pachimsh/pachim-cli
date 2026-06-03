package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// APIError is a structured error returned by the Pachim API.
type APIError struct {
	StatusCode int
	Message    string
	Details    []string
}

func (e *APIError) Error() string {
	var b strings.Builder

	if e.Message != "" {
		b.WriteString(e.Message)
	} else {
		b.WriteString(defaultHTTPMessage(e.StatusCode))
	}

	for _, detail := range e.Details {
		b.WriteString("\n  • ")
		b.WriteString(detail)
	}

	return b.String()
}

func AsAPIError(err error) (*APIError, bool) {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr, true
	}

	return nil, false
}

func parseHTTPError(statusCode int, body []byte) error {
	if statusCode == 401 {
		return &APIError{
			StatusCode: 401,
			Message:    "Authentication failed. Please run: pachim login",
		}
	}

	if statusCode == 429 {
		return &APIError{
			StatusCode: 429,
			Message:    "Rate limited. Please wait and try again",
		}
	}

	var resp struct {
		Message string          `json:"message"`
		Status  int             `json:"status"`
		Errors  json.RawMessage `json:"errors"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		trimmed := strings.TrimSpace(string(body))
		if trimmed == "" {
			return &APIError{StatusCode: statusCode, Message: defaultHTTPMessage(statusCode)}
		}

		return &APIError{StatusCode: statusCode, Message: trimmed}
	}

	details := flattenErrors(resp.Errors)

	if resp.Message == "" {
		var errStr string
		if json.Unmarshal(resp.Errors, &errStr) == nil && errStr != "" {
			return &APIError{StatusCode: statusCode, Message: errStr}
		}
	}

	message := resp.Message
	if message == "" {
		message = defaultHTTPMessage(statusCode)
	}

	return &APIError{
		StatusCode: statusCode,
		Message:    message,
		Details:    details,
	}
}

func defaultHTTPMessage(code int) string {
	switch code {
	case 401:
		return "Authentication failed. Please run: pachim login"
	case 422:
		return "The given data was invalid."
	case 429:
		return "Rate limited. Please wait and try again"
	default:
		return fmt.Sprintf("Request failed (HTTP %d).", code)
	}
}

func flattenErrors(raw json.RawMessage) []string {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}

	var str string
	if err := json.Unmarshal(raw, &str); err == nil {
		if strings.TrimSpace(str) == "" {
			return nil
		}

		return []string{str}
	}

	var list []string
	if err := json.Unmarshal(raw, &list); err == nil {
		return nonEmptyStrings(list)
	}

	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return []string{string(raw)}
	}

	keys := make([]string, 0, len(obj))
	for key := range obj {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var lines []string
	for _, field := range keys {
		lines = append(lines, flattenFieldErrors(field, obj[field])...)
	}

	return lines
}

func flattenFieldErrors(field string, raw json.RawMessage) []string {
	var messages []string
	if err := json.Unmarshal(raw, &messages); err == nil {
		return formatFieldMessages(field, nonEmptyStrings(messages))
	}

	var message string
	if err := json.Unmarshal(raw, &message); err == nil && strings.TrimSpace(message) != "" {
		return formatFieldMessages(field, []string{message})
	}

	var nested map[string]json.RawMessage
	if err := json.Unmarshal(raw, &nested); err == nil {
		var lines []string
		for key, value := range nested {
			nestedField := field
			if field != "" {
				nestedField = field + "." + key
			} else {
				nestedField = key
			}
			lines = append(lines, flattenFieldErrors(nestedField, value)...)
		}
		sort.Strings(lines)

		return lines
	}

	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return nil
	}

	return []string{fmt.Sprintf("%s: %s", field, trimmed)}
}

func formatFieldMessages(field string, messages []string) []string {
	lines := make([]string, 0, len(messages))
	for _, message := range messages {
		if field == "" {
			lines = append(lines, message)
			continue
		}

		lines = append(lines, fmt.Sprintf("%s: %s", field, message))
	}

	return lines
}

func nonEmptyStrings(values []string) []string {
	lines := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			lines = append(lines, value)
		}
	}

	return lines
}
