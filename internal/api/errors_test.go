package api

import (
	"testing"
)

func TestParseHTTPErrorValidation(t *testing.T) {
	body := []byte(`{
		"message": "The given data was invalid.",
		"status": 422,
		"errors": {
			"project_zip": ["The project zip field must not be greater than 51200 kilobytes."],
			"branch": ["The branch format is invalid."]
		}
	}`)

	err := parseHTTPError(422, body)
	apiErr, ok := AsAPIError(err)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.Message != "The given data was invalid." {
		t.Fatalf("unexpected message: %q", apiErr.Message)
	}

	if len(apiErr.Details) != 2 {
		t.Fatalf("expected 2 details, got %d: %#v", len(apiErr.Details), apiErr.Details)
	}
}

func TestParseHTTPErrorFailMessage(t *testing.T) {
	body := []byte(`{"errors":"File upload failed.","status":500}`)

	err := parseHTTPError(500, body)
	apiErr, ok := AsAPIError(err)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.Message != "File upload failed." {
		t.Fatalf("unexpected message: %q", apiErr.Message)
	}
}

func TestParseHTTPErrorForbiddenWithMessageCode(t *testing.T) {
	body := []byte(`{
		"message": "This site belongs to another Pachim account.",
		"message_code": "site_owner_mismatch",
		"status": 403
	}`)

	err := parseHTTPError(403, body)
	apiErr, ok := AsAPIError(err)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.MessageCode != "site_owner_mismatch" {
		t.Fatalf("unexpected message code: %q", apiErr.MessageCode)
	}

	if len(apiErr.Hints) == 0 {
		t.Fatalf("expected hints for site_owner_mismatch")
	}
}

func TestParseHTTPErrorLegacyUnauthorized(t *testing.T) {
	err := parseHTTPError(403, []byte(`{"message":"unauthorization","status":403}`))
	apiErr, ok := AsAPIError(err)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.MessageCode != "access_denied" {
		t.Fatalf("unexpected message code: %q", apiErr.MessageCode)
	}

	if len(apiErr.Hints) == 0 {
		t.Fatalf("expected hints for access_denied")
	}
}

func TestParseHTTPErrorUnauthorized(t *testing.T) {
	err := parseHTTPError(401, []byte(`{"message":"Unauthenticated","status":401}`))
	apiErr, ok := AsAPIError(err)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.Message != "Authentication failed. Please run: pachim login" {
		t.Fatalf("unexpected message: %q", apiErr.Message)
	}
}
